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

CREATE TABLE IF NOT EXISTS emails (
    id            SERIAL PRIMARY KEY,
    email_address VARCHAR(255) UNIQUE NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users_emails (
    id         SERIAL PRIMARY KEY,
    user_id    INTEGER     NOT NULL REFERENCES users(id)  ON DELETE CASCADE,
    email_id   INTEGER     NOT NULL REFERENCES emails(id) ON DELETE CASCADE,
    importance INTEGER     NOT NULL DEFAULT 0,
    category   INTEGER     NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, email_id)
);

-- Indices
CREATE INDEX IF NOT EXISTS idx_messages_sender_id       ON messages(sender_id);
CREATE INDEX IF NOT EXISTS idx_messages_receiver_id     ON messages(receiver_id);
CREATE INDEX IF NOT EXISTS idx_messages_date_sent       ON messages(date_sent);
CREATE INDEX IF NOT EXISTS idx_message_files_message_id ON message_files(message_id);
CREATE INDEX IF NOT EXISTS idx_message_statuses_msg_id  ON message_statuses(message_id);
CREATE INDEX IF NOT EXISTS idx_emails_address           ON emails(email_address);
CREATE INDEX IF NOT EXISTS idx_users_emails_user_id     ON users_emails(user_id);
CREATE INDEX IF NOT EXISTS idx_users_emails_email_id    ON users_emails(email_id);
CREATE INDEX IF NOT EXISTS idx_users_emails_importance  ON users_emails(importance);
