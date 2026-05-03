# CQRS Sync Service

Учебный Go-сервис, который читает доменные события из Kafka и проецирует их в MongoDB read-model. Источник записи предполагается во внешнем PostgreSQL write-model, но сам сервис не пишет в PostgreSQL: он только поднимает подключения, читает события и обновляет MongoDB.

## Что есть сейчас

- один бинарь: [`cmd/db_sync/main.go`](/home/yorunikakeru/Documents/Education/2/DMS/db_sync/cmd/db_sync/main.go:1)
- consumer Kafka на `segmentio/kafka-go`
- read-model в MongoDB, коллекция `users`
- PostgreSQL schema в [`init.sql`](/home/yorunikakeru/Documents/Education/2/DMS/db_sync/init.sql:1)
- обработка событий:
  - `user_created`
  - `user_deleted`
  - `email_added`
  - `email_updated`
  - `email_removed`
  - `message_created`
  - `message_deleted`
- unit-тесты для сервисов, middleware и dispatch-роутинга
- integration-тесты с build tag `integration`

## Поток данных

1. Внешняя write-side система меняет данные в PostgreSQL.
2. Она публикует событие в Kafka topic `sync_topic`.
3. Сервис читает событие и декодирует envelope из [`internal/application/events`](/home/yorunikakeru/Documents/Education/2/DMS/db_sync/internal/application/events/event.go:1).
4. Один из сервисов в [`internal/service`](/home/yorunikakeru/Documents/Education/2/DMS/db_sync/internal/service/sync_service.go:1) обновляет MongoDB read-model.

Текущая read-model хранится в документе `users`:

- базовые поля пользователя
- `important_contacts`
- `messages`

## Структура

```text
.
├── cmd/db_sync
├── internal/app
├── internal/application/events
├── internal/application/mapper
├── internal/config
├── internal/domain
├── internal/middleware
├── internal/models
├── internal/service
├── internal/storage
├── internal/testutil
├── internal/transport/kafka
├── internal/view
├── init.sql
├── docker-compose.yml
└── Taskfile.yml
```

## Конфигурация

Нужен Go `1.26`.

Локальная инфраструктура поднимается через [`docker-compose.yml`](/home/yorunikakeru/Documents/Education/2/DMS/db_sync/docker-compose.yml:1) или задачи из [`Taskfile.yml`](/home/yorunikakeru/Documents/Education/2/DMS/db_sync/Taskfile.yml:1).

Основные переменные:

- `POSTGRES_USER`
- `POSTGRES_PASSWORD`
- `POSTGRES_DB`
- `MONGO_INITDB_ROOT_USERNAME`
- `MONGO_INITDB_ROOT_PASSWORD`
- `KAFKA_TOPIC`
- `KAFKA_GROUP_ID`

Быстрый старт:

```bash
task init
task up:docker
go build -o db_sync ./cmd/db_sync/...
./db_sync
```

Если используется Podman:

```bash
task up:podman
task run
```

Mongo Express поднимается на `http://localhost:8081`.

## Тесты

Unit:

```bash
task test
```

Integration:

```bash
task test:integration
```

Integration-тесты подключаются к реальным PostgreSQL и MongoDB. Если базы недоступны, helper в [`internal/testutil/integration.go`](/home/yorunikakeru/Documents/Education/2/DMS/db_sync/internal/testutil/integration.go:1) пропускает тесты.

## Ограничения текущей реализации

- сервис читает только из Kafka и не содержит write-side API
- формат payload жёстко завязан на структуры из `internal/application/events`
- проекции в MongoDB делаются точечными `UpdateOne`, без транзакций и без восстановления из event log
- часть read-model полей объявлена, но не поддерживается полноценно

## Полезные файлы

- точка входа: [`cmd/db_sync/main.go`](/home/yorunikakeru/Documents/Education/2/DMS/db_sync/cmd/db_sync/main.go:1)
- wiring приложения: [`internal/app/app.go`](/home/yorunikakeru/Documents/Education/2/DMS/db_sync/internal/app/app.go:1)
- Kafka consumer: [`internal/transport/kafka/consumer.go`](/home/yorunikakeru/Documents/Education/2/DMS/db_sync/internal/transport/kafka/consumer.go:1)
- Mongo/Postgres repositories: [`internal/storage`](/home/yorunikakeru/Documents/Education/2/DMS/db_sync/internal/storage/user_repo.go:1)
- integration tests: [`internal/integration`](/home/yorunikakeru/Documents/Education/2/DMS/db_sync/internal/integration/user_test.go:1)
