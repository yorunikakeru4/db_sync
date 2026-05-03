# Backend API

Write-side HTTP API for the email domain. This service:

- writes to PostgreSQL through `bun`
- publishes domain events to Kafka
- serves `GET /users/:id` from the MongoDB CQRS read model

## Layout

- `cmd/api` — binary entrypoint
- `internal/transport/http` — HTTP handlers
- `internal/storage` — PostgreSQL write-side + MongoDB read-side repositories
- `internal/events` — event envelope and payloads emitted to Kafka

## Runtime

Environment defaults follow the shared root `.env.example`.

Extra backend-specific variables:

- `BACKEND_HTTP_ADDR` — listen address, default `:8080`
- `KAFKA_BROKERS` — comma-separated brokers, default `localhost:9092`
- `KAFKA_HOST` / `KAFKA_PORT` — used when `KAFKA_BROKERS` is unset
- `POSTGRES_HOST` / `POSTGRES_PORT` — PostgreSQL address, defaults `localhost:5432`
- `MONGO_HOST` / `MONGO_PORT` — MongoDB address, defaults `localhost:27017`
- `MONGO_DB_NAME` — MongoDB database, default `email_service`

## Tasks

From repo root:

```bash
task backend:build
task backend:run
task backend:test
task backend:test:integration
```

Directly in the module:

```bash
cd backend
task build
task run
```

## Local flow

1. Start infra: `task up:docker`
2. Start both services: `task dev`
3. Backend API listens on `http://localhost:8080`

`POST/PUT/DELETE` mutate PostgreSQL and publish Kafka events. `GET /users/:id` reads MongoDB projections updated by the CQRS consumer.
