# Plan: backend/ microservice (bunrouter + bun + Kafka producer) + bun models & datadriven tests in cqrs/

## Context

The `cqrs` module (`db_sync`) is a pure Kafka→MongoDB sync service with a PostgreSQL write-model accessed via `sqlx`. No HTTP layer exists. This task adds:

1. A new **`backend/`** microservice — separate Go module (`backend`) with bunrouter HTTP handlers (POST/PUT/DELETE), `bun` ORM writes to PostgreSQL, and a **Kafka producer** that publishes events after each DB mutation (so the existing `cqrs` consumer picks them up).
2. **bun models** in `backend/internal/bunmodel/` (HTTP layer) and `cqrs/internal/bunmodel/` (for tests in cqrs). They're independent copies — no cross-module import.
3. **Stripped-down dbfixture loader** in `cqrs/internal/testutil/dbfixture.go` — PostgreSQL-only wrapper around the local `dbfixture/` package. The existing `testutil/integration.go` is NOT changed (it owns the MongoDB+PostgreSQL setup for existing integration tests). Remove any MongoDB model loaders; keep only the four PostgreSQL models.
4. **Datadriven integration tests** in both `backend/internal/transport/http/` and `cqrs/internal/storage/` using a lightweight custom `TestRunner` that understands `create-user`, `http`, and `check-kafka` commands. Output is YAML.

No `github.com/uptrace/uptrace/pkg/...` internal packages used anywhere. `github.com/uptrace/bun` and `github.com/uptrace/bunrouter` are allowed.

---

## Repository layout after the change

```
repo/
├── go.work                                   ← add "use ./backend"
├── cqrs/                                     (existing module — db_sync)
│   ├── go.mod                                ← add bun, local dbfixture, local datadriven, yaml
│   └── internal/
│       ├── bunmodel/                         (NEW) bun-tagged mirror of domain/
│       │   ├── user.go
│       │   ├── message.go
│       │   └── email.go
│       ├── storage/
│       │   ├── init_bun.go                   (NEW) OpenBunDB() using same env vars as init_postgre.go
│       │   └── testdata/                     (NEW) datadriven cases — storage layer
│       │       ├── fixtures/initial.yml
│       │       ├── users.txt
│       │       ├── messages.txt
│       │       └── emails.txt
│       └── testutil/
│           ├── integration.go                (UNCHANGED)
│           └── dbfixture.go                  (NEW) PostgreSQL-only fixture loader
└── backend/                                  (NEW module — backend)
    ├── go.mod
    ├── cmd/api/main.go
    └── internal/
        ├── bunmodel/                         bun-tagged models (same schema)
        │   ├── user.go
        │   ├── message.go
        │   └── email.go
        ├── events/                           Kafka event envelope + typed payloads
        │   ├── event.go                      (mirrors cqrs/internal/application/events/)
        │   ├── user_events.go
        │   ├── message_events.go
        │   └── email_events.go
        ├── kafka/
        │   ├── producer.go                   KafkaProducer interface + kafka-go impl
        │   └── mem_producer.go               MemProducer for tests
        ├── storage/
        │   └── init_bun.go                   OpenBunDB()
        └── transport/http/
            ├── handler.go                    Handler struct, NewHandler(), ServeHTTP
            ├── user_handler.go
            ├── message_handler.go
            ├── email_handler.go
            ├── runner_test.go                TestRunner + datadriven walk (build: integration)
            └── testdata/
                ├── fixtures/initial.yml
                ├── users.txt
                ├── messages.txt
                └── emails.txt
```

---

## Data flow

```
HTTP client  (YAML body)
    │
    ▼
bunrouter.Router
    ├── POST   /users                 → createUser   → bun INSERT → Kafka user_created
    ├── PUT    /users/:id             → updateUser   → bun UPDATE → Kafka user_updated *
    ├── DELETE /users/:id             → deleteUser   → bun DELETE → Kafka user_deleted
    ├── POST   /messages              → createMessage ...
    ├── PUT    /messages/:id          → updateMessage ...
    ├── DELETE /messages/:id          → deleteMessage ...
    ├── POST   /users/:uid/emails     → addEmail    → bun upsert emails + insert users_emails → Kafka email_added
    ├── PUT    /users/:uid/emails/:id → updateEmail → bun UPDATE users_emails → Kafka email_updated
    └── DELETE /users/:uid/emails/:id → removeEmail → bun DELETE users_emails → Kafka email_removed

* user_updated not in cqrs consumer today — add the event type but the consumer can ignore it
```

---

## Implementation steps

### Step 1 — `go.work` update

```
go 1.26

use ./cqrs
use ./backend
```

---

### Step 2 — `backend/go.mod`

```
module backend

go 1.26

require (
    github.com/uptrace/bunrouter        v1.x
    github.com/uptrace/bun              v1.x
    github.com/uptrace/bun/dialect/pgdialect v1.x
    github.com/uptrace/bun/driver/pgdriver  v1.x
    github.com/segmentio/kafka-go       v0.4.x
    github.com/goccy/go-yaml            v1.x
    github.com/stretchr/testify         v1.x
)
```

---

### Step 3 — bun models (same for both `backend/internal/bunmodel/` and `cqrs/internal/bunmodel/`)

Mirror the `db_sync` domain types with bun struct tags instead of `db:` tags:

```go
// user.go
type User struct {
    bun.BaseModel `bun:"table:users,alias:u"`
    ID        int64     `bun:"id,pk,autoincrement"`
    Email     string    `bun:"email,notnull"`
    CreatedAt time.Time `bun:"created_at,notnull,default:now()"`
}

// message.go
type Message struct {
    bun.BaseModel `bun:"table:messages,alias:m"`
    ID         int64     `bun:"id,pk,autoincrement"`
    ExternalID string    `bun:"external_id,notnull"`
    SenderID   int64     `bun:"sender_id,notnull"`
    ReceiverID int64     `bun:"receiver_id,notnull"`
    Subject    string    `bun:"subject"`
    Text       string    `bun:"text"`
    DateSent   time.Time `bun:"date_sent"`
    CreatedAt  time.Time `bun:"created_at,notnull,default:now()"`
}

// email.go
type Email struct {
    bun.BaseModel `bun:"table:emails,alias:e"`
    ID        int64     `bun:"id,pk,autoincrement"`
    Address   string    `bun:"email_address,notnull"`
    CreatedAt time.Time `bun:"created_at,notnull,default:now()"`
}

type UserEmail struct {
    bun.BaseModel `bun:"table:users_emails,alias:ue"`
    ID         int64     `bun:"id,pk,autoincrement"`
    UserID     int64     `bun:"user_id,notnull"`
    EmailID    int64     `bun:"email_id,notnull"`
    Importance int       `bun:"importance,default:0"`
    Category   int       `bun:"category,default:0"`
    CreatedAt  time.Time `bun:"created_at,notnull,default:now()"`
}
```

---

### Step 4 — `backend/internal/events/` — Kafka event types

Mirror the structure from `cqrs/internal/application/events/` (four files):

```go
// event.go
type Event struct {
    EventID   string    `json:"event_id"`
    EventType string    `json:"event_type"`
    Version   int       `json:"version"`
    Timestamp time.Time `json:"timestamp"`
    Payload   any       `json:"payload"`
}

// user_events.go
const (
    UserCreated = "user_created"
    UserDeleted = "user_deleted"
)
type UserCreatedPayload struct {
    ID        int64     `json:"id"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}
type UserDeletedPayload struct { ID int64 `json:"id"` }
```

Similarly for message and email events (matching existing event type strings exactly so the cqrs consumer routes them correctly).

---

### Step 5 — `backend/internal/kafka/producer.go`

```go
type KafkaProducer interface {
    Publish(ctx context.Context, eventType string, payload any) error
    Close() error
}

type kafkaProducer struct {
    writer *kafka.Writer
    topic  string
}

func NewKafkaProducer(brokers []string, topic string) KafkaProducer {
    return &kafkaProducer{
        writer: &kafka.Writer{Addr: kafka.TCP(brokers...), Topic: topic},
        topic:  topic,
    }
}

func (p *kafkaProducer) Publish(ctx context.Context, eventType string, payload any) error {
    ev := events.Event{
        EventID:   uuid.New().String(),   // use crypto/rand for UUID, avoid external deps
        EventType: eventType,
        Version:   1,
        Timestamp: time.Now().UTC(),
        Payload:   payload,
    }
    b, err := json.Marshal(ev)
    if err != nil { return err }
    return p.writer.WriteMessages(ctx, kafka.Message{Value: b})
}
```

`uuid.New()` → use `fmt.Sprintf("%x-%x-...", rand bytes)` or `github.com/google/uuid` (lightweight, already likely indirect dep). Prefer `crypto/rand` to avoid new deps.

---

### Step 6 — `backend/internal/kafka/mem_producer.go`

```go
type MemProducer struct {
    mu     sync.Mutex
    Events []events.Event
}

func (p *MemProducer) Publish(ctx context.Context, eventType string, payload any) error {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.Events = append(p.Events, events.Event{EventType: eventType, Payload: payload})
    return nil
}
func (p *MemProducer) Close() error { return nil }
```

---

### Step 7 — `backend/internal/storage/init_bun.go`

```go
func OpenBunDB() (*bun.DB, error) {
    dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
        os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"),
        getEnv("POSTGRES_HOST", "email_postgres"),
        getEnv("POSTGRES_PORT", "5432"),
        os.Getenv("POSTGRES_DB"))
    sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
    db := bun.NewDB(sqldb, pgdialect.New())
    return db, db.PingContext(context.Background())
}
```

`cqrs/internal/storage/init_bun.go` mirrors this, using the existing `config.SQLConnect` struct (same env vars already read there).

---

### Step 8 — `backend/internal/transport/http/handler.go`

```go
type Handler struct {
    db       *bun.DB
    producer kafka.KafkaProducer
    router   *bunrouter.Router
}

func NewHandler(db *bun.DB, producer kafka.KafkaProducer) *Handler {
    h := &Handler{db: db, producer: producer, router: bunrouter.New()}
    h.router.POST("/users", h.createUser)
    h.router.PUT("/users/:id", h.updateUser)
    h.router.DELETE("/users/:id", h.deleteUser)
    // ... messages, emails
    return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { h.router.ServeHTTP(w, r) }

// Helpers (no external deps)
func readYAML(r *http.Request, dst any) error       { return yaml.NewDecoder(r.Body).Decode(dst) }
func writeYAML(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/yaml")
    w.WriteHeader(status)
    yaml.NewEncoder(w).Encode(v)
}
```

---

### Step 9 — Handler files

Each handler: decode YAML body → bun DB op → publish Kafka event → encode YAML response.

**`user_handler.go`:**

```go
func (h *Handler) createUser(w http.ResponseWriter, req bunrouter.Request) error {
    var body struct { Email string `yaml:"email"` }
    if err := readYAML(req.Request, &body); err != nil { return err }
    user := &bunmodel.User{Email: body.Email}
    if _, err := h.db.NewInsert().Model(user).Exec(req.Context()); err != nil { return err }
    _ = h.producer.Publish(req.Context(), events.UserCreated, events.UserCreatedPayload{
        ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt,
    })
    writeYAML(w, 201, user)
    return nil
}
// updateUser, deleteUser follow the same pattern
```

**`message_handler.go`**, **`email_handler.go`** — same pattern for their respective tables and event types.

---

### Step 10 — `cqrs/internal/testutil/dbfixture.go` (PostgreSQL-only, no MongoDB)

```go
// LoadFixtures seeds PostgreSQL tables from YAML fixture files via local dbfixture helpers.
// Only the four PostgreSQL models are registered; MongoDB is intentionally excluded.
func LoadFixtures(ctx context.Context, db *bun.DB, fsys fs.FS, names ...string) error {
    f := dbfixture.New(db, dbfixture.WithRecreateTables())
    f.AddModel(&bunmodel.User{})
    f.AddModel(&bunmodel.Message{})
    f.AddModel(&bunmodel.Email{})
    f.AddModel(&bunmodel.UserEmail{})
    return f.Load(ctx, fsys, names...)
}
```

Local `dbfixture` wiring must register only PostgreSQL loaders. No MongoDB loaders. Test setup should keep cases isolated without manual cross-test cleanup.

---

### Step 11 — `testdata/fixtures/initial.yml`

Seed rows that both backends use as starting state:

```yaml
- model: User
  rows:
    - id: 1
      email: alice@example.com
      created_at: "2024-01-01T00:00:00Z"

- model: Message
  rows:
    - id: 1
      external_id: ext-001
      sender_id: 1
      receiver_id: 1
      subject: Hello
      text: World
      date_sent: "2024-01-01T00:00:00Z"
      created_at: "2024-01-01T00:00:00Z"
```

---

### Step 12 — `TestRunner` and the test format (`backend/internal/transport/http/runner_test.go`)

The TestRunner is a lightweight datadriven adapter. It supports three commands:

| Command       | Args                                   | Body                     | Action                                      |
| ------------- | -------------------------------------- | ------------------------ | ------------------------------------------- |
| `create-user` | `name=alice`                           | YAML user fields         | insert via bun, store ref `user:alice` → id |
| `http`        | `method=GET path=/users/1 format=body` | YAML body (for POST/PUT) | round-trip via httptest                     |
| `check-kafka` | —                                      | —                        | drain MemProducer, emit events as YAML      |

Named refs in paths: `{user:alice.id}` → substituted with stored ID before request.

```go
//go:build integration

func TestHTTPHandlers(t *testing.T) {
    db := openTestDB(t)            // skip if Postgres unavailable
    mem := &kafka.MemProducer{}
    srv := httptest.NewServer(http.NewHandler(db, mem))
    t.Cleanup(srv.Close)

    runner := &TestRunner{db: db, srv: srv, mem: mem, refs: map[string]string{}}

    // Register YAML formatter
    runner.RegisterFormatter("yaml", func(t *testing.T, cmd string, rec *httptest.ResponseRecorder) string {
        if rec.Code >= 400 {
            return fmt.Sprintf("%d\n%s", rec.Code, strings.TrimSpace(rec.Body.String()))
        }
        // decode response body YAML and re-encode for stable output
        var v any
        yaml.Unmarshal(rec.Body.Bytes(), &v)
        b, _ := yaml.Marshal(v)
        return fmt.Sprintf("%d\n%s", rec.Code, strings.TrimSpace(string(b)))
    })

    datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
        datadriven.RunTest(t, path, runner.Run)
    })
}
```

`TestRunner.Run` dispatches on `d.Cmd`:

- `create-user`: decode body YAML, POST `/users` via srv, parse response, store `refs["user:name.id"] = strconv.Itoa(id)`, return YAML output
- `http`: resolve `{user:name.id}` refs in `path` arg, build request, record response via `httptest.ResponseRecorder`, call registered formatter
- `check-kafka`: iterate `mem.Events`, marshal each as YAML, clear mem, return multi-doc YAML block

---

### Step 13 — Datadriven test files (YAML in / YAML out)

**`testdata/users.txt`:**

```
create-user name=bob
email: bob@example.com
----
id: 2
email: bob@example.com

http method=PUT path=/users/{user:bob.id} format=yaml
email: bob2@example.com
----
200
id: 2
email: bob2@example.com

check-kafka
----
- event_type: user_created
  payload:
    id: 2
    email: bob@example.com
- event_type: user_updated
  payload:
    id: 2
    email: bob2@example.com

http method=DELETE path=/users/{user:bob.id}
----
204
```

**`testdata/messages.txt`** and **`testdata/emails.txt`** follow the same structure.

---

### Step 14 — `cqrs/` storage-layer datadriven tests

In `cqrs/internal/storage/bun_storage_test.go` (build tag: `integration`), a simpler runner that omits HTTP and kafka — just `insert-user` / `get-user` / `delete-user` commands that call bun directly and emit YAML output. Uses `testutil.LoadFixtures` for seeding.

---

### Step 15 — `cqrs/go.mod` additions

```
github.com/uptrace/bun                  v1.x
github.com/uptrace/bun/dialect/pgdialect v1.x
github.com/uptrace/bun/driver/pgdriver  v1.x
local `dbfixture`
local `datadriven`
gopkg.in/yaml.v3                        v3.x
```

---

## Files modified

| File          | Change                            |
| ------------- | --------------------------------- |
| `go.work`     | add `use ./backend`               |
| `cqrs/go.mod` | add 6 new deps, run `go mod tidy` |

## Files NOT modified

| File                                        | Reason                                                                    |
| ------------------------------------------- | ------------------------------------------------------------------------- |
| `cqrs/internal/domain/`                     | existing sqlx types stay; bun models are in new `bunmodel/` subpackage    |
| `cqrs/internal/storage/*_repo.go`           | sqlx repos unchanged                                                      |
| `cqrs/internal/app/app.go`                  | Kafka loop unchanged                                                      |
| `cqrs/internal/testutil/integration.go`     | existing TestDB (PostgreSQL+MongoDB) stays for existing integration tests |
| `cqrs/internal/transport/kafka/consumer.go` | consumer unchanged; just receives events from backend                     |

---

## Verification

```bash
# Build both modules
cd backend && go build ./...
cd cqrs    && go build ./...

# Unit tests (no infra)
cd cqrs && go test ./internal/...

# Integration — needs docker-compose stack (postgres + kafka)
cd backend && go test -tags integration -v ./internal/transport/http/...
cd cqrs    && go test -tags integration -v ./internal/storage/...
```
