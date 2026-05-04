# db_sync monorepo

Учебный CQRS-проект для синхронизации данных между **PostgreSQL (write model)** и **MongoDB (read model)** через **Kafka**.

## Что в репозитории

- `backend/` — write-side HTTP API (Go): пишет в PostgreSQL и публикует события в Kafka.
- `cqrs/` — read-side sync service (Go): читает события из Kafka и обновляет MongoDB-проекции.
- `init.sql` — схема PostgreSQL, индексы и триггеры.
- `docker-compose.yml` — локальная инфраструктура (PostgreSQL, MongoDB, Kafka, Mongo Express).
- `vue/` — фронтенд-консоль (Vue + Vite).
- `datadriven/` — отдельный модуль data-driven тестов.
- `dbfixture/` — пакет фикстур для БД.

## Архитектура потока

1. Клиент вызывает `backend` (HTTP).
2. `backend` выполняет мутацию в PostgreSQL.
3. `backend` публикует доменное событие в Kafka (`sync_topic`).
4. `cqrs` читает событие и применяет проекцию в MongoDB (`email_service.users`).
5. `GET /users/:id` в `backend` возвращает готовую MongoDB-проекцию.

## Быстрый старт

```bash
task init
task up:docker
task dev
```

`task dev` запускает одновременно `cqrs` и `backend`.

Для hot-reload обоих сервисов:

```bash
task watch
```

## Основные команды

Инфраструктура:

```bash
task up:docker
task down:docker
task up:podman
task down:podman
```

CQRS:

```bash
task cqrs:build
task cqrs:run
task cqrs:test
task cqrs:test:integration
```

Backend:

```bash
task backend:build
task backend:run
task backend:test
task backend:test:integration
```

Watch только backend/cqrs по отдельности:

```bash
task backend:watch
task cqrs:watch
```

Все тесты:

```bash
task test
```

## API backend (кратко)

- `GET /users/:id`
- `POST /users`
- `PUT /users/:id`
- `DELETE /users/:id`
- `POST /users/:user_id/contacts`
- `PUT /users/:user_id/contacts/:contact_id`
- `DELETE /users/:user_id/contacts/:contact_id`
- `POST /messages`
- `PUT /messages/:id`
- `DELETE /messages/:id`

Важно: request body в текущей реализации декодируется как YAML (`go-yaml`), а ответы отдаются в JSON.

## Доменные события

Публикуются backend:

- `user_created`, `user_updated`, `user_deleted`
- `contact_added`, `contact_updated`, `contact_removed`
- `message_created`, `message_deleted`

Обрабатываются cqrs:

- `user_created`, `user_deleted`
- `contact_added`, `contact_updated`, `contact_removed`
- `message_created`, `message_deleted`

На текущий момент `user_updated` публикуется backend, но не маршрутизируется в cqrs-consumer.

## Переменные окружения

Скопируйте `.env.example` в `.env`:

```bash
task init
```

Ключевые переменные:

- PostgreSQL: `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `POSTGRES_HOST`, `POSTGRES_PORT`
- MongoDB: `MONGO_INITDB_ROOT_USERNAME`, `MONGO_INITDB_ROOT_PASSWORD`, `MONGO_HOST`, `MONGO_PORT`, `MONGO_DB_NAME`
- Kafka: `KAFKA_TOPIC`, `KAFKA_HOST`, `KAFKA_PORT`, `KAFKA_BROKERS`, `KAFKA_GROUP_ID`
- Backend: `BACKEND_HTTP_ADDR`

## Frontend (`vue/`)

```bash
cd vue
pnpm install
cp .env.example .env.local
# VITE_API_BASE_URL=http://localhost:8080
pnpm dev
```

Сборка:

```bash
cd vue
pnpm build
```
