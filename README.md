# db_sync

CQRS-система для синхронизации данных между PostgreSQL write model и MongoDB read model через Kafka.

Проект разделен на два основных сервиса:

- `backend/` — write-side HTTP API на Go. Выполняет CRUD-операции в PostgreSQL, читает готовые MongoDB-проекции и публикует доменные события через outbox worker.
- `cqrs/` — read-side consumer на Go. Читает события из Kafka и обновляет MongoDB read model.

Дополнительные части репозитория:

- `vue/` — frontend-консоль на Vue + Vite
- `datadriven/` — data-driven тесты и сценарии
- `dbfixture/` — фикстуры БД для интеграционных тестов
- `init.sql` — схема PostgreSQL, ограничения, триггеры и outbox

## Архитектура

Поток данных построен вокруг схемы:

```text
HTTP client
   |
   v
backend
   | 1) mutation в PostgreSQL
   v
PostgreSQL tables
   | 2) trigger пишет событие в domain_events
   v
domain_events
   | 3) backend event worker публикует в Kafka
   v
Kafka
   | 4) cqrs consumer читает и применяет проекцию
   v
MongoDB users read model
   |
   v
backend GET endpoints читают готовую проекцию
```

Ключевая идея:

- write-side не публикует события напрямую из HTTP handler
- событие сначала фиксируется в PostgreSQL через outbox таблицу `domain_events`
- отдельный worker публикует его в Kafka
- `cqrs` асинхронно обновляет MongoDB read model

## Компоненты

### `backend/`

Основные зоны ответственности:

- HTTP API для write-side операций
- чтение MongoDB read model для `GET`-маршрутов
- PostgreSQL репозитории через `bun`
- event worker, который выгружает `domain_events` в Kafka

### `cqrs/`

Основные зоны ответственности:

- Kafka consumer
- маршрутизация доменных событий
- обновление MongoDB read model
- проекции пользователей, контактов и сообщений

### `vue/`

Frontend-консоль для ручной работы со сценарием:

- создание, обновление и удаление пользователей
- создание, обновление и удаление сообщений
- создание и удаление контактных связей
- просмотр read model и request log

## Технологии

- Go
- PostgreSQL
- MongoDB
- Kafka
- Vue 3 + Vite
- Taskfile
- Docker Compose

## Быстрый старт

Подготовка окружения:

```bash
task init
task up:docker
task dev
```

`task dev` запускает `backend` и `cqrs`.

Для hot reload:

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

Backend:

```bash
task backend:build
task backend:run
task backend:test
task backend:test:integration
task backend:watch
```

CQRS:

```bash
task cqrs:build
task cqrs:run
task cqrs:test
task cqrs:test:integration
task cqrs:watch
```

Все тесты:

```bash
task test
```

## HTTP API

### Read model endpoints

- `GET /users`
- `GET /users/:id`
- `GET /messages`
- `GET /contacts`
- `GET /users/:user_id/contacts`

### Write model endpoints

- `POST /users`
- `PUT /users/:id`
- `DELETE /users/:id`
- `POST /messages`
- `PUT /messages/:id`
- `DELETE /messages/:id`
- `POST /users/:user_id/contacts`
- `PUT /users/:user_id/contacts/:contact_id`
- `DELETE /users/:user_id/contacts/:contact_id`

Особенности API:

- request body для write-side endpoint декодируется как YAML
- ответы возвращаются в JSON
- CORS разрешен для локального frontend dev origin

## Доменные события

PostgreSQL формирует события через триггеры и складывает их в `domain_events`.

Основные типы:

- `user_created`
- `user_updated`
- `user_deleted`
- `contact_added`
- `contact_updated`
- `contact_removed`
- `message_created`
- `message_deleted`

Read-side обрабатывает те же события, с одной важной оговоркой:

- отдельного `message_updated` нет
- обновление сообщения продолжает приходить как `message_created`
- в текущей реализации CQRS трактует `message_created` как upsert сообщения в проекции

## Read model

Основная MongoDB-коллекция — `users`.

Документ пользователя хранит:

- `id`
- `email`
- `created_at`
- `important_contacts`
- `messages`
- `num_messages`

Из этих документов backend отдает:

- список пользователей
- одного пользователя
- список сообщений
- список контактов
- контакты конкретного пользователя

## PostgreSQL: ограничения и триггеры

В `init.sql` описаны:

- базовая схема таблиц `users`, `messages`, `contacts`, `users_contacts`
- каскадное удаление зависимых сущностей там, где это нужно
- ограничения домена, например запрет отправки сообщения самому себе
- технические триггеры для статусов и outbox pipeline

Outbox pipeline строится на таблице `domain_events`, которую заполняют SQL-trigger-ы после insert/update/delete операций в write model.

## Переменные окружения

После `task init` создается `.env` на основе шаблона.

Ключевые переменные:

- PostgreSQL: `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `POSTGRES_HOST`, `POSTGRES_PORT`
- MongoDB: `MONGO_INITDB_ROOT_USERNAME`, `MONGO_INITDB_ROOT_PASSWORD`, `MONGO_HOST`, `MONGO_PORT`, `MONGO_DB_NAME`
- Kafka: `KAFKA_TOPIC`, `KAFKA_HOST`, `KAFKA_PORT`, `KAFKA_BROKERS`, `KAFKA_GROUP_ID`
- Backend: `BACKEND_HTTP_ADDR`

## Frontend

Локальный запуск:

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

## Структура репозитория

```text
db_sync/
├── backend/
├── cqrs/
├── vue/
├── datadriven/
├── dbfixture/
├── docs/
├── init.sql
├── docker-compose.yml
├── Taskfile.yml
└── go.work
```
