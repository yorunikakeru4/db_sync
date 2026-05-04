# Отчёт по индивидуальной работе

## Тема: Система синхронизации данных между разнородными БД (CQRS Email Service)

---

## 1. Описание проекта

`db_sync` — учебный монорепозиторий CQRS-системы для почтового домена.

Система разделена на два основных сервиса:

- `backend/` — write-side HTTP API. Выполняет CRUD-операции в PostgreSQL, читает готовые проекции из MongoDB и поднимает worker публикации событий.
- `cqrs/` — read-side consumer. Читает доменные события из Kafka и обновляет MongoDB read model.

Дополнительные части репозитория:

- `vue/` — frontend-консоль на Vue + Vite.
- `datadriven/` — отдельный модуль для data-driven тестов.
- `dbfixture/` — пакет фикстур для БД и интеграционных сценариев.

---

## 2. Используемые технологии и хранилища

### 2.1 PostgreSQL

- Контейнер: `postgres:16-alpine`
- Роль: write model и источник бизнес-событий
- Схема: `init.sql`

Основные таблицы:

- `users`
- `messages`
- `message_files`
- `message_statuses`
- `contacts`
- `users_contacts`
- `deleted_users_log`
- `domain_events`

Таблица `domain_events` играет роль outbox-хранилища: SQL-триггеры складывают туда доменные события после изменений write-модели, а отдельный worker публикует их в Kafka.

### 2.2 MongoDB

- Контейнер: `mongo:7`
- БД: `email_service`
- Основная коллекция read model: `users`

Документ пользователя в read model содержит:

- `id`
- `email`
- `created_at`
- `important_contacts`
- `messages`
- `num_messages`

### 2.3 Kafka

- Контейнер: `apache/kafka:3.9.0`
- Топик по умолчанию: `sync_topic`
- Роль: транспорт доменных событий между write-side и read-side

---

## 3. Архитектура системы

Архитектура системы построена вокруг связки `PostgreSQL triggers -> domain_events -> backend event worker -> Kafka -> cqrs consumer -> MongoDB`.

```text
HTTP client
   |
   v
backend (cmd/api)
   | 1) меняет write model в PostgreSQL
   v
PostgreSQL tables
   | 2) AFTER INSERT/UPDATE/DELETE triggers
   v
domain_events (outbox table)
   | 3) backend/internal/eventworker poll + publish
   v
Kafka (sync_topic)
   | 4) cqrs consumer
   v
MongoDB.users (read model)
   |
   v
backend GET endpoints читают готовые MongoDB-проекции
```

Ключевые свойства схемы:

- HTTP-слой не отвечает за сборку и отправку событий.
- Событие фиксируется в той же PostgreSQL-среде, где произошла мутация.
- Публикацию в Kafka можно повторять до отметки `published_at`.

---

## 4. Компоненты системы

### 4.1 Backend (`backend/`)

Основные зоны ответственности:

- `cmd/api` — точка входа сервиса.
- `internal/transport/http` — HTTP API.
- `internal/storage` — репозитории PostgreSQL и MongoDB.
- `internal/readmodel` — структуры read model, которые backend отдаёт клиенту.
- `internal/eventworker` — polling `domain_events` и публикация событий в Kafka.

`backend` одновременно:

- обслуживает write-side CRUD,
- читает MongoDB-проекции для GET-запросов,
- запускает worker, который публикует непереданные события в Kafka.

### 4.2 CQRS service (`cqrs/`)

Основные зоны ответственности:

- `cmd/db_sync` — точка входа consumer-сервиса.
- `internal/transport/kafka` — Kafka consumer и роутинг событий.
- `internal/service` — прикладная логика проекций.
- `internal/storage` — MongoDB/PostgreSQL репозитории.
- `internal/view` — структуры MongoDB read model.

`cqrs` не выполняет write-side CRUD. Его задача — читать события и поддерживать MongoDB-проекции в актуальном состоянии.

---

## 5. Поток данных

Типовой сценарий выглядит так:

1. Клиент вызывает HTTP endpoint в `backend`.
2. `backend` изменяет запись в PostgreSQL через `bun`.
3. PostgreSQL trigger создаёт запись в `domain_events`.
4. `backend/internal/eventworker` выбирает непубликованные события по `published_at IS NULL`.
5. Worker сериализует payload и публикует событие в Kafka.
6. После успешной публикации worker проставляет `published_at`.
7. `cqrs` читает сообщение из Kafka.
8. Consumer маршрутизирует событие по типу и обновляет MongoDB read model.
9. `GET`-эндпоинты в `backend` читают уже подготовленную проекцию из MongoDB.

---

## 6. Доменные события

### 6.1 События, которые формирует PostgreSQL

По `init.sql` в `domain_events` попадают:

- `user_created`
- `user_updated`
- `user_deleted`
- `contact_added`
- `contact_updated`
- `contact_removed`
- `message_created`
- `message_deleted`

### 6.2 События, которые обрабатывает `cqrs`

`DispatchEvent` в `cqrs/internal/transport/kafka/consumer.go` поддерживает:

- `user_created`
- `user_deleted`
- `contact_added`
- `contact_updated`
- `contact_removed`
- `message_created`
- `message_deleted`

Особенности обработки событий:

- `user_updated` публикуется, но `cqrs` его игнорирует.
- отдельного `message_updated` нет; обновление сообщения генерирует событие типа `message_created` с новым payload.

Это важное ограничение реализации.

---

## 7. HTTP API backend

### 7.1 Endpoints чтения read model

- `GET /users`
- `GET /users/:id`
- `GET /messages`
- `GET /contacts`
- `GET /users/:user_id/contacts`

Эти маршруты читают данные из MongoDB-проекций, а не напрямую из PostgreSQL.

### 7.2 Endpoints мутаций write model

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

- request body декодируется через `github.com/goccy/go-yaml`;
- ответы возвращаются в JSON;
- после мутации событие не публикуется самим хендлером, а фиксируется через SQL-trigger и уходит в Kafka асинхронно через worker.

---

## 8. Read model в MongoDB

MongoDB хранит денормализованную проекцию пользователя.

Один документ пользователя содержит:

- основные поля пользователя;
- массив `important_contacts`;
- массив `messages`;
- счётчик `num_messages`.

Из этих документов backend дополнительно строит плоские представления:

- список пользователей;
- список контактов;
- список сообщений;
- список контактов конкретного пользователя.

Таким образом frontend и API чтения работают не с write-моделью, а с уже подготовленным read model.

---

## 9. Триггеры и ограничения PostgreSQL

В `init.sql` описаны две группы серверной логики.

### 9.1 Бизнес-ограничения и сервисные триггеры

- `trg_messages_auto_status` — автоматически создаёт статус `sent` после вставки сообщения.
- `trg_message_statuses_updated_at` — обновляет `updated_at` при изменении статуса.
- `trg_users_contacts_clamp_importance` — ограничивает `importance` диапазоном `[0, 10]`.
- `trg_messages_no_self_send` — запрещает отправку сообщения самому себе.
- `trg_users_audit_delete` — пишет удалённого пользователя в `deleted_users_log`.

### 9.2 CQRS/outbox триггеры

- `trg_user_domain_event` — пишет `user_*` события в `domain_events`.
- `trg_message_domain_event` — пишет `message_*` события в `domain_events`.
- `trg_contact_domain_event` — пишет `contact_*` события в `domain_events`.

Именно эти триггеры делают `domain_events` центральной точкой интеграции между write-side и event pipeline.

---

## 10. CRUD по слоям

### 10.1 PostgreSQL write side

В `backend` реализованы:

- создание, обновление и удаление пользователей;
- создание, обновление и удаление сообщений;
- создание, изменение и удаление связей пользователя с контактами;
- поддержка связанных доменных записей и снимков состояния для event payload.

### 10.2 MongoDB read side

В `cqrs` реализованы:

- создание MongoDB-документа пользователя по `user_created`;
- удаление документа по `user_deleted`;
- добавление, изменение и удаление вложенных контактов;
- добавление и удаление вложенных сообщений;
- поддержка `num_messages`.

---

## 11. Структура репозитория

```text
db_sync/
├── backend/          # write-side API + outbox publisher worker
├── cqrs/             # read-side Kafka consumer + MongoDB projector
├── vue/              # frontend-консоль
├── datadriven/       # data-driven test utilities
├── dbfixture/        # database fixtures
├── docs/REPORT.md    # отчёт по проекту
├── init.sql          # PostgreSQL schema, triggers, outbox
├── docker-compose.yml
├── Taskfile.yml      # orchestration из корня
└── go.work           # Go workspace
```

---

## 12. Запуск и команды

### 12.1 Быстрый старт

```bash
task init
task up:docker
task dev
```

`task dev` запускает одновременно `cqrs` и `backend`.

### 12.2 Отдельный запуск сервисов

```bash
task backend:run
task cqrs:run
```

### 12.3 Сборка

```bash
task backend:build
task cqrs:build
```

### 12.4 Тесты

```bash
task backend:test
task backend:test:integration
task cqrs:test
task cqrs:test:integration
task test
```

### 12.5 Режим watch

```bash
task backend:watch
task cqrs:watch
task watch
```

---

## 13. Конфигурация

Ключевые переменные окружения:

- PostgreSQL: `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `POSTGRES_HOST`, `POSTGRES_PORT`
- MongoDB: `MONGO_INITDB_ROOT_USERNAME`, `MONGO_INITDB_ROOT_PASSWORD`, `MONGO_HOST`, `MONGO_PORT`, `MONGO_DB_NAME`
- Kafka: `KAFKA_TOPIC`, `KAFKA_HOST`, `KAFKA_PORT`, `KAFKA_BROKERS`, `KAFKA_GROUP_ID`
- Backend: `BACKEND_HTTP_ADDR`

Для локального старта используются `.env.example`, root `Taskfile.yml` и `docker-compose.yml`.

---

## 14. Тестирование

В проекте есть:

- unit-тесты в `backend/internal/...` и `cqrs/internal/...`;
- integration-тесты для HTTP pipeline и CQRS pipeline;
- data-driven сценарии;
- фикстуры для БД.

Интеграционные тесты особенно важны, потому что проект проверяет не только отдельные функции, но и сквозную цепочку:

- PostgreSQL mutation,
- запись в `domain_events`,
- публикацию/чтение событий,
- обновление MongoDB read model.

---

## 15. Итоги и ограничения

Проект представляет собой рабочую учебную CQRS-систему с раздельными write/read потоками и актуальной event-driven схемой через PostgreSQL outbox.

Что уже реализовано:

- write-side API на Go;
- PostgreSQL как источник истины;
- автоматическое формирование доменных событий триггерами;
- backend worker для публикации outbox-событий в Kafka;
- cqrs consumer для обновления MongoDB;
- read-side endpoints, которые читают готовые MongoDB-проекции;
- frontend для работы с пользователями, сообщениями и контактами.

Главные ограничения:

- `user_updated` публикуется, но не проецируется в MongoDB;
- обновление сообщения не имеет отдельного типа `message_updated` и проходит как `message_created`;
- read model строится инкрементально и не содержит механизма полного rebuild из event log;
- GET-эндпоинты читают MongoDB, поэтому между записью в PostgreSQL и появлением данных в read model есть асинхронная задержка.
