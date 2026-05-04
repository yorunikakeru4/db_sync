-- Schema for the email service write model (PostgreSQL).
-- Executed once on first container start via docker-entrypoint-initdb.d.

CREATE TABLE IF NOT EXISTS users (
    id         SERIAL PRIMARY KEY,
    email      VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS messages (
    id          SERIAL PRIMARY KEY,
    external_id VARCHAR(255) UNIQUE NOT NULL,
    sender_id   INTEGER      NOT NULL REFERENCES users(id),
    receiver_id INTEGER      NOT NULL REFERENCES users(id),
    subject     VARCHAR(500),
    text        TEXT,
    date_sent   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS message_files (
    id         SERIAL PRIMARY KEY,
    message_id INTEGER      NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    file_path  VARCHAR(512) NOT NULL,
    type       VARCHAR(100),
    size       INTEGER      NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS message_statuses (
    id         SERIAL PRIMARY KEY,
    message_id INTEGER     NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    status     VARCHAR(50) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS contacts (
    id            SERIAL PRIMARY KEY,
    contact_value VARCHAR(255) UNIQUE NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users_contacts (
    id         SERIAL PRIMARY KEY,
    user_id    INTEGER     NOT NULL REFERENCES users(id)  ON DELETE CASCADE,
    contact_id INTEGER     NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    importance INTEGER     NOT NULL DEFAULT 0,
    category   INTEGER     NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, contact_id)
);

-- Indices
CREATE INDEX IF NOT EXISTS idx_messages_sender_id       ON messages(sender_id);
CREATE INDEX IF NOT EXISTS idx_messages_receiver_id     ON messages(receiver_id);
CREATE INDEX IF NOT EXISTS idx_messages_date_sent       ON messages(date_sent);
CREATE INDEX IF NOT EXISTS idx_message_files_message_id ON message_files(message_id);
CREATE INDEX IF NOT EXISTS idx_message_statuses_msg_id  ON message_statuses(message_id);
CREATE INDEX IF NOT EXISTS idx_contacts_value            ON contacts(contact_value);
CREATE INDEX IF NOT EXISTS idx_users_contacts_user_id    ON users_contacts(user_id);
CREATE INDEX IF NOT EXISTS idx_users_contacts_contact_id ON users_contacts(contact_id);
CREATE INDEX IF NOT EXISTS idx_users_contacts_importance ON users_contacts(importance);

-- ── Audit log (populated by trigger, not by the application) ─────────────────

-- Stores a record for every deleted user so history is preserved.
CREATE TABLE IF NOT EXISTS deleted_users_log (
    id         SERIAL PRIMARY KEY,
    user_id    INTEGER      NOT NULL,
    email      VARCHAR(255) NOT NULL,
    deleted_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ── Trigger functions ─────────────────────────────────────────────────────────

-- 1. Auto-insert the initial 'sent' status whenever a message is created.
--    Guarantees every message always has at least one status row.
CREATE OR REPLACE FUNCTION trg_fn_messages_auto_status()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO message_statuses (message_id, status, updated_at)
    VALUES (NEW.id, 'sent', NOW());
    RETURN NEW;
END;
$$;

CREATE OR REPLACE TRIGGER trg_messages_auto_status
    AFTER INSERT ON messages
    FOR EACH ROW EXECUTE FUNCTION trg_fn_messages_auto_status();

-- 2. Keep message_statuses.updated_at fresh on every status change.
--    The application never needs to pass a timestamp manually.
CREATE OR REPLACE FUNCTION trg_fn_message_statuses_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$;

CREATE OR REPLACE TRIGGER trg_message_statuses_updated_at
    BEFORE UPDATE ON message_statuses
    FOR EACH ROW EXECUTE FUNCTION trg_fn_message_statuses_updated_at();

-- 3. Clamp users_contacts.importance to [0, 10] on every write.
--    Enforces the domain invariant at the DB level regardless of caller.
CREATE OR REPLACE FUNCTION trg_fn_users_contacts_clamp_importance()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.importance := GREATEST(0, LEAST(10, NEW.importance));
    RETURN NEW;
END;
$$;

CREATE OR REPLACE TRIGGER trg_users_contacts_clamp_importance
    BEFORE INSERT OR UPDATE ON users_contacts
    FOR EACH ROW EXECUTE FUNCTION trg_fn_users_contacts_clamp_importance();

-- 4. Prevent a user from sending a message to themselves.
--    A domain rule that belongs in the DB, not scattered across services.
CREATE OR REPLACE FUNCTION trg_fn_messages_no_self_send()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.sender_id = NEW.receiver_id THEN
        RAISE EXCEPTION 'sender and receiver must be different (user_id=%)', NEW.sender_id;
    END IF;
    RETURN NEW;
END;
$$;

CREATE OR REPLACE TRIGGER trg_messages_no_self_send
    BEFORE INSERT ON messages
    FOR EACH ROW EXECUTE FUNCTION trg_fn_messages_no_self_send();

-- 5. Write a row to deleted_users_log whenever a user is deleted.
--    Enables audit / recovery without relying on application-side logging.
CREATE OR REPLACE FUNCTION trg_fn_users_audit_delete()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO deleted_users_log (user_id, email, deleted_at)
    VALUES (OLD.id, OLD.email, NOW());
    RETURN OLD;
END;
$$;

CREATE OR REPLACE TRIGGER trg_users_audit_delete
    AFTER DELETE ON users
    FOR EACH ROW EXECUTE FUNCTION trg_fn_users_audit_delete();

-- ── Domain Events (CQRS Event Sourcing) ─────────────────────────────────────

-- Stores all domain events for outbox pattern publishing to Kafka.
CREATE TABLE IF NOT EXISTS domain_events (
    id              BIGSERIAL PRIMARY KEY,
    event_type      VARCHAR(64) NOT NULL,
    aggregate_type  VARCHAR(32) NOT NULL,
    aggregate_id    BIGINT NOT NULL,
    payload         JSONB NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_domain_events_unpublished
    ON domain_events (id)
    WHERE published_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_domain_events_aggregate
    ON domain_events (aggregate_type, aggregate_id);

-- 6. Capture user domain events on INSERT/UPDATE/DELETE.
CREATE OR REPLACE FUNCTION trg_fn_user_domain_event()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE
    event_type_val VARCHAR(64);
    payload_val JSONB;
    agg_id BIGINT;
BEGIN
    IF TG_OP = 'INSERT' THEN
        event_type_val := 'user_created';
        agg_id := NEW.id;
        payload_val := jsonb_build_object(
            'id', NEW.id,
            'email', NEW.email,
            'created_at', NEW.created_at
        );
    ELSIF TG_OP = 'UPDATE' THEN
        event_type_val := 'user_updated';
        agg_id := NEW.id;
        payload_val := jsonb_build_object(
            'id', NEW.id,
            'email', NEW.email
        );
    ELSIF TG_OP = 'DELETE' THEN
        event_type_val := 'user_deleted';
        agg_id := OLD.id;
        payload_val := jsonb_build_object('id', OLD.id);
    END IF;

    INSERT INTO domain_events (event_type, aggregate_type, aggregate_id, payload)
    VALUES (event_type_val, 'user', agg_id, payload_val);

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;

CREATE OR REPLACE TRIGGER trg_user_domain_event
    AFTER INSERT OR UPDATE OR DELETE ON users
    FOR EACH ROW EXECUTE FUNCTION trg_fn_user_domain_event();

-- 7. Capture message domain events on INSERT/UPDATE/DELETE.
CREATE OR REPLACE FUNCTION trg_fn_message_domain_event()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE
    event_type_val VARCHAR(64);
    payload_val JSONB;
    agg_id BIGINT;
BEGIN
    IF TG_OP = 'INSERT' THEN
        event_type_val := 'message_created';
        agg_id := NEW.id;
        payload_val := jsonb_build_object(
            'message_id', NEW.id,
            'user_id', NEW.sender_id,
            'subject', NEW.subject,
            'content', NEW.text,
            'date_sent', NEW.date_sent
        );
    ELSIF TG_OP = 'UPDATE' THEN
        event_type_val := 'message_created';
        agg_id := NEW.id;
        payload_val := jsonb_build_object(
            'message_id', NEW.id,
            'user_id', NEW.sender_id,
            'subject', NEW.subject,
            'content', NEW.text,
            'date_sent', NEW.date_sent
        );
    ELSIF TG_OP = 'DELETE' THEN
        event_type_val := 'message_deleted';
        agg_id := OLD.id;
        payload_val := jsonb_build_object(
            'message_id', OLD.id,
            'user_id', OLD.sender_id
        );
    END IF;

    INSERT INTO domain_events (event_type, aggregate_type, aggregate_id, payload)
    VALUES (event_type_val, 'message', agg_id, payload_val);

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;

CREATE OR REPLACE TRIGGER trg_message_domain_event
    AFTER INSERT OR UPDATE OR DELETE ON messages
    FOR EACH ROW EXECUTE FUNCTION trg_fn_message_domain_event();

-- 8. Capture contact domain events on INSERT/UPDATE/DELETE of users_contacts.
--    Joins with contacts table to get the contact value.
CREATE OR REPLACE FUNCTION trg_fn_contact_domain_event()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE
    event_type_val VARCHAR(64);
    payload_val JSONB;
    agg_id BIGINT;
    contact_value VARCHAR(255);
    old_contact_value VARCHAR(255);
BEGIN
    IF TG_OP = 'INSERT' THEN
        SELECT c.contact_value INTO contact_value FROM contacts c WHERE c.id = NEW.contact_id;
        event_type_val := 'contact_added';
        agg_id := NEW.user_id;
        payload_val := jsonb_build_object(
            'contact_id', NEW.contact_id,
            'user_id', NEW.user_id,
            'value', contact_value,
            'category', NEW.category::text,
            'importance', NEW.importance,
            'created_at', NEW.created_at
        );
    ELSIF TG_OP = 'UPDATE' THEN
        SELECT c.contact_value INTO contact_value FROM contacts c WHERE c.id = NEW.contact_id;
        event_type_val := 'contact_updated';
        agg_id := NEW.user_id;
        payload_val := jsonb_build_object(
            'contact_id', NEW.contact_id,
            'value', contact_value,
            'old_value', contact_value,
            'user_id', NEW.user_id,
            'old_category', OLD.category::text,
            'new_category', NEW.category::text,
            'old_importance', OLD.importance,
            'new_importance', NEW.importance,
            'updated_at', NOW()
        );
    ELSIF TG_OP = 'DELETE' THEN
        SELECT c.contact_value INTO contact_value FROM contacts c WHERE c.id = OLD.contact_id;
        event_type_val := 'contact_removed';
        agg_id := OLD.user_id;
        payload_val := jsonb_build_object(
            'contact_id', OLD.contact_id,
            'value', contact_value,
            'user_id', OLD.user_id
        );
    END IF;

    INSERT INTO domain_events (event_type, aggregate_type, aggregate_id, payload)
    VALUES (event_type_val, 'contact', agg_id, payload_val);

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;

CREATE OR REPLACE TRIGGER trg_contact_domain_event
    AFTER INSERT OR UPDATE OR DELETE ON users_contacts
    FOR EACH ROW EXECUTE FUNCTION trg_fn_contact_domain_event();
