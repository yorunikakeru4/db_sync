
# CQRS Sync Service (PostgreSQL → MongoDB)

Базовая CQRS-система на Go для синхронизации данных между **PostgreSQL** (write-модель) и **MongoDB** (read-модель).

## Идея

- **PostgreSQL** — источник истины (Command side)
- **MongoDB** — оптимизированное хранилище для чтения (Query side)
- Синхронизация выполняется асинхронно через события

Архитектура следует принципам **CQRS** (Command Query Responsibility Segregation).

## Архитектура

Client -> API (Write) -> : 
- Write Postgre
- Kafka request -> MongoDB write

Client -> API (Read) -> Read MongoDB

## Основные компоненты

- **Commands**  
  Изменяют состояние системы. Работают только с PostgreSQL.

- **Events**  
  Факты, описывающие произошедшие изменения (Created, Updated, Deleted).

- **Projectors / Consumers**  
  Обрабатывают события и обновляют MongoDB.

- **Queries**  
  Чтение данных исключительно из MongoDB.

## Технологии

- Go 1.22+
- PostgreSQL
- MongoDB
- Kafka / RabbitMQ / NATS

## Структура проекта

```

.
├── cmd
│   └── db_sync
│       └── main.go
├── db_sync
├── Dockerfile
├── go.mod
├── go.sum
└── internal
    ├── app
    │   └── app.go
    ├── application
    │   ├── events
    │   │   ├── email_events.go
    │   │   ├── event.go
    │   │   ├── message_events.go
    │   │   └── user_events.go
    │   └── mapper
    │       └── email_mapper.go
    ├── config
    │   ├── kafka.go
    │   ├── mongo.go
    │   └── postgres.go
    ├── domain
    │   ├── email.go
    │   ├── message.go
    │   └── user.go
    ├── models
    │   ├── email.go
    │   ├── file.go
    │   ├── message.go
    │   └── user.go
    ├── service
    │   ├── email_service.go
    │   ├── message_service.go
    │   ├── sync_sevice.go
    │   └── user_service.go
    ├── storage
    │   ├── email_repo.go
    │   ├── init_mongo.go
    │   ├── init_postgre.go
    │   ├── message_repo.go
    │   └── user_repo.go
    ├── transport
    │   └── kafka
    │       └── consumer.go
    └── view
        ├── message_view.go
        └── user_view.go

````

## Гарантии

- PostgreSQL — **strong consistency**
- MongoDB — **eventual consistency**
- Возможна повторная репликация (idempotent projectors)

## Запуск (пример)

```bash
docker compose up -d
go run cmd/app/main.go
````

## Когда использовать

* Высокая нагрузка на чтение
* Сложные read-модели
* Необходимость масштабировать чтение отдельно от записи

## Ограничения

* Нет мгновенной консистентности
* Более сложная инфраструктура
* Требует контроля версий событий

## Статус

Минимальный шаблон / учебный проект.
Расширяется под конкретный домен.


