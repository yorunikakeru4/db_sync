# Spec: PostgreSQL Trigger-Based Event Sourcing

## Overview

Replace manual event publishing in HTTP handlers with PostgreSQL triggers that automatically capture domain events on INSERT/UPDATE/DELETE. A background worker polls the events table and publishes to Kafka.

## Current Architecture

```
Handler → DB mutation → handler.Publish(event) → Kafka → CQRS worker → MongoDB
```

**Problems:**
- Event publishing coupled to handler logic
- Possible inconsistency: DB commits but Publish fails
- Manual event creation boilerplate in every handler

## New Architecture

```
Handler → DB mutation → [trigger fires] → domain_events table
                                              ↓
                              Event Worker (goroutine)
                                              ↓
                                           Kafka → CQRS worker → MongoDB
```

**Benefits:**
- Guaranteed event capture (same transaction as mutation)
- Handlers become simpler (just CRUD)
- Single event publishing point
- Audit log built-in

---

## Phase 1: PostgreSQL Schema Changes

### 1.1 New Table: `domain_events`

```sql
CREATE TABLE domain_events (
    id              BIGSERIAL PRIMARY KEY,
    event_type      VARCHAR(64) NOT NULL,
    aggregate_type  VARCHAR(32) NOT NULL,  -- 'user', 'message', 'contact'
    aggregate_id    BIGINT NOT NULL,
    payload         JSONB NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at    TIMESTAMPTZ,           -- NULL = not yet published
    version         INT NOT NULL DEFAULT 1
);

CREATE INDEX idx_domain_events_unpublished
    ON domain_events (id)
    WHERE published_at IS NULL;

CREATE INDEX idx_domain_events_aggregate
    ON domain_events (aggregate_type, aggregate_id);
```

### 1.2 Trigger Function: Generic Event Capture

```sql
CREATE OR REPLACE FUNCTION capture_domain_event()
RETURNS TRIGGER AS $$
DECLARE
    event_type_val VARCHAR(64);
    payload_val JSONB;
BEGIN
    -- Determine event type
    IF TG_OP = 'INSERT' THEN
        event_type_val := TG_ARGV[0] || '_created';
    ELSIF TG_OP = 'UPDATE' THEN
        event_type_val := TG_ARGV[0] || '_updated';
    ELSIF TG_OP = 'DELETE' THEN
        event_type_val := TG_ARGV[0] || '_deleted';
    END IF;

    -- Build payload based on operation
    IF TG_OP = 'DELETE' THEN
        payload_val := to_jsonb(OLD);
    ELSE
        payload_val := to_jsonb(NEW);
    END IF;

    INSERT INTO domain_events (event_type, aggregate_type, aggregate_id, payload)
    VALUES (
        event_type_val,
        TG_ARGV[0],
        CASE
            WHEN TG_OP = 'DELETE' THEN (OLD).id
            ELSE (NEW).id
        END,
        payload_val
    );

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

### 1.3 Triggers Per Table

```sql
-- Users
CREATE TRIGGER users_event_trigger
    AFTER INSERT OR UPDATE OR DELETE ON users
    FOR EACH ROW EXECUTE FUNCTION capture_domain_event('user');

-- Messages
CREATE TRIGGER messages_event_trigger
    AFTER INSERT OR UPDATE OR DELETE ON messages
    FOR EACH ROW EXECUTE FUNCTION capture_domain_event('message');

-- Contacts (users_contacts junction table)
CREATE TRIGGER users_contacts_event_trigger
    AFTER INSERT OR UPDATE OR DELETE ON users_contacts
    FOR EACH ROW EXECUTE FUNCTION capture_domain_event('contact');
```

### 1.4 Special Case: Contact Trigger

`users_contacts` doesn't have `id` column. Need custom trigger:

```sql
CREATE OR REPLACE FUNCTION capture_contact_event()
RETURNS TRIGGER AS $$
DECLARE
    event_type_val VARCHAR(64);
    payload_val JSONB;
    agg_id BIGINT;
BEGIN
    IF TG_OP = 'INSERT' THEN
        event_type_val := 'contact_added';
        agg_id := NEW.user_id;
        payload_val := jsonb_build_object(
            'user_id', NEW.user_id,
            'contact_id', NEW.contact_id,
            'importance', NEW.importance,
            'category', NEW.category
        );
    ELSIF TG_OP = 'UPDATE' THEN
        event_type_val := 'contact_updated';
        agg_id := NEW.user_id;
        payload_val := jsonb_build_object(
            'user_id', NEW.user_id,
            'contact_id', NEW.contact_id,
            'importance', NEW.importance,
            'category', NEW.category,
            'old_importance', OLD.importance,
            'old_category', OLD.category
        );
    ELSIF TG_OP = 'DELETE' THEN
        event_type_val := 'contact_removed';
        agg_id := OLD.user_id;
        payload_val := jsonb_build_object(
            'user_id', OLD.user_id,
            'contact_id', OLD.contact_id
        );
    END IF;

    INSERT INTO domain_events (event_type, aggregate_type, aggregate_id, payload)
    VALUES (event_type_val, 'contact', agg_id, payload_val);

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_contacts_event_trigger
    AFTER INSERT OR UPDATE OR DELETE ON users_contacts
    FOR EACH ROW EXECUTE FUNCTION capture_contact_event();
```

---

## Phase 2: Event Worker (Go)

### 2.1 New Package: `backend/internal/eventworker/`

```
backend/internal/eventworker/
├── worker.go       # Main polling loop
├── publisher.go    # Kafka publishing logic
└── repository.go   # DB queries for events
```

### 2.2 Repository Interface

```go
// repository.go
package eventworker

type DomainEvent struct {
    ID            int64
    EventType     string
    AggregateType string
    AggregateID   int64
    Payload       json.RawMessage
    CreatedAt     time.Time
    PublishedAt   *time.Time
    Version       int
}

type Repository interface {
    // Fetch unpublished events, ordered by ID, limit N
    FetchUnpublished(ctx context.Context, limit int) ([]DomainEvent, error)

    // Mark events as published (batch)
    MarkPublished(ctx context.Context, ids []int64) error
}
```

### 2.3 Worker Implementation

```go
// worker.go
package eventworker

type Worker struct {
    repo      Repository
    producer  kafka.Producer
    pollInterval time.Duration
    batchSize    int
}

func NewWorker(repo Repository, producer kafka.Producer, opts ...Option) *Worker {
    w := &Worker{
        repo:         repo,
        producer:     producer,
        pollInterval: 100 * time.Millisecond,
        batchSize:    100,
    }
    for _, opt := range opts {
        opt(w)
    }
    return w
}

func (w *Worker) Run(ctx context.Context) error {
    ticker := time.NewTicker(w.pollInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            if err := w.processBatch(ctx); err != nil {
                slog.Error("event worker batch failed", "error", err)
            }
        }
    }
}

func (w *Worker) processBatch(ctx context.Context) error {
    events, err := w.repo.FetchUnpublished(ctx, w.batchSize)
    if err != nil {
        return fmt.Errorf("fetch unpublished: %w", err)
    }
    if len(events) == 0 {
        return nil
    }

    publishedIDs := make([]int64, 0, len(events))
    for _, evt := range events {
        if err := w.producer.Publish(ctx, evt.EventType, evt.Payload); err != nil {
            slog.Error("publish failed", "event_id", evt.ID, "error", err)
            continue // Skip this event, retry next cycle
        }
        publishedIDs = append(publishedIDs, evt.ID)
    }

    if len(publishedIDs) > 0 {
        if err := w.repo.MarkPublished(ctx, publishedIDs); err != nil {
            return fmt.Errorf("mark published: %w", err)
        }
    }

    return nil
}
```

### 2.4 Repository Implementation

```go
// repository_pg.go
package eventworker

type PgRepository struct {
    db *bun.DB
}

func (r *PgRepository) FetchUnpublished(ctx context.Context, limit int) ([]DomainEvent, error) {
    var events []DomainEvent
    err := r.db.NewSelect().
        Model(&events).
        Where("published_at IS NULL").
        Order("id ASC").
        Limit(limit).
        Scan(ctx)
    return events, err
}

func (r *PgRepository) MarkPublished(ctx context.Context, ids []int64) error {
    _, err := r.db.NewUpdate().
        Model((*DomainEvent)(nil)).
        Set("published_at = ?", time.Now()).
        Where("id IN (?)", bun.In(ids)).
        Exec(ctx)
    return err
}
```

---

## Phase 3: Handler Simplification

### 3.1 Remove Manual Event Publishing

**Before (user_handler.go):**
```go
func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
    // ... validation, create user in DB ...

    // Manual event publishing - REMOVE THIS
    if err := h.producer.Publish(r.Context(), events.UserCreated, events.UserCreatedPayload{
        ID:        user.ID,
        Email:     user.Email,
        CreatedAt: user.CreatedAt,
    }); err != nil {
        // error handling
    }
}
```

**After:**
```go
func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
    // ... validation, create user in DB ...
    // Event captured automatically by trigger
    respondJSON(w, http.StatusCreated, user)
}
```

### 3.2 Handler Dependencies

Remove `Producer` from `HandlerParams`:

```go
type HandlerParams struct {
    Users     storage.UserRepo
    Messages  storage.MessageRepo
    Contacts  storage.ContactRepo
    UserViews storage.UserViewRepo
    // Producer kafka.Producer  -- REMOVE
}
```

---

## Phase 4: Integration

### 4.1 Main Function Changes

```go
// backend/cmd/api/main.go
func main() {
    // ... existing setup ...

    // Start event worker in background goroutine
    eventRepo := eventworker.NewPgRepository(db)
    eventWorker := eventworker.NewWorker(eventRepo, producer,
        eventworker.WithPollInterval(100*time.Millisecond),
        eventworker.WithBatchSize(100),
    )

    go func() {
        if err := eventWorker.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
            slog.Error("event worker stopped", "error", err)
        }
    }()

    // ... HTTP server ...
}
```

### 4.2 Graceful Shutdown

```go
// Shutdown sequence
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// 1. Stop accepting HTTP requests
server.Shutdown(ctx)

// 2. Stop event worker (cancel context)
workerCancel()

// 3. Flush remaining events (optional: call processBatch one more time)
```

---

## Phase 5: Event Type Mapping

Map trigger event types to existing CQRS consumer expectations:

| Trigger Event Type | Existing Event Type | Notes |
|-------------------|---------------------|-------|
| `user_created` | `user_created` | Match |
| `user_updated` | `user_updated` | Match |
| `user_deleted` | `user_deleted` | Match |
| `message_created` | `message_created` | Match |
| `message_deleted` | `message_deleted` | Match |
| `contact_added` | `contact_added` | Match |
| `contact_updated` | `contact_updated` | Match |
| `contact_removed` | `contact_removed` | Match |

No changes needed in CQRS consumer.

---

## Testing Strategy

### Unit Tests

1. **Trigger tests** (SQL)
   - INSERT creates event with correct type/payload
   - UPDATE creates event with new values
   - DELETE creates event with old values

2. **Worker tests**
   - Polls and publishes unpublished events
   - Marks events as published
   - Handles publish failures gracefully
   - Respects context cancellation

### Integration Tests

1. **End-to-end flow**
   - HTTP POST /users → trigger fires → event in table → worker publishes → Kafka message

2. **Consistency check**
   - Verify all mutations generate events
   - Verify MongoDB projection matches after events processed

### Datadriven Test Example

```
-- testdata/triggers.dd

insert users
email: test@example.com
----
event_type: user_created
payload.email: test@example.com

update users
id: 1
email: new@example.com
----
event_type: user_updated
payload.email: new@example.com

delete users
id: 1
----
event_type: user_deleted
payload.id: 1
```

---

## Migration Plan

### Step 1: Deploy Schema
- Add `domain_events` table
- Add triggers (they're additive, won't break existing flow)

### Step 2: Deploy Worker
- Add event worker goroutine
- Both old (handler publish) and new (trigger → worker) will work simultaneously

### Step 3: Remove Handler Publishing
- Remove `producer.Publish()` calls from handlers
- Remove `Producer` from handler dependencies

### Step 4: Cleanup
- Remove old event structs from handlers if unused
- Update tests

---

## Files to Modify

| File | Change |
|------|--------|
| `init.sql` | Add `domain_events` table, trigger function, triggers |
| `backend/internal/eventworker/worker.go` | NEW: Worker implementation |
| `backend/internal/eventworker/repository.go` | NEW: DB queries |
| `backend/cmd/api/main.go` | Start worker goroutine |
| `backend/internal/transport/http/user_handler.go` | Remove Publish calls |
| `backend/internal/transport/http/message_handler.go` | Remove Publish calls |
| `backend/internal/transport/http/contact_handler.go` | Remove Publish calls |
| `backend/internal/transport/http/handler.go` | Remove Producer dependency |

---

## Open Questions

1. **Retry strategy**: How many times to retry failed publishes before giving up?
2. **Dead letter**: Should failed events go to DLQ table?
3. **Event retention**: How long to keep published events? Add cleanup job?
4. **Ordering guarantees**: Current design preserves order per aggregate (by ID). Sufficient?
5. **Backpressure**: What if worker can't keep up? Alert threshold?

---

## Effort Estimate

- Phase 1 (Schema): 1-2 hours
- Phase 2 (Worker): 2-3 hours
- Phase 3 (Handler cleanup): 1 hour
- Phase 4 (Integration): 1 hour
- Phase 5 (Testing): 2-3 hours

**Total: ~8-10 hours**
