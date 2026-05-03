# Отчёт по индивидуальной работе
## Тема: Система синхронизации данных между разнородными базами данных (CQRS Email Service)

---

## 1. Описание приложения

**db_sync** — система обработки событий для почтового сервиса, построенная на архитектурном паттерне **CQRS** (Command Query Responsibility Segregation — разделение операций записи и чтения).

Система состоит из двух отдельных частей:

- **Write-сторона (Backend API)** — принимает HTTP-запросы, сохраняет данные в PostgreSQL и публикует события в Apache Kafka.
- **Read-сторона (CQRS Sync Service)** — потребляет события из Kafka и строит денормализованные проекции в MongoDB для быстрого чтения.

Такое разделение позволяет масштабировать операции чтения и записи независимо друг от друга.

---

## 2. Используемые базы данных

В проекте используются **две базы данных разных типов**: реляционная и документная.

### 2.1 PostgreSQL (реляционная, SQL)

**Роль:** модель записи — единственный источник истины (source of truth).

**Версия:** PostgreSQL 16  
**Драйвер Go:** `github.com/jackc/pgx/v5`, `github.com/jmoiron/sqlx`

**Схема данных (6 таблиц):**

| Таблица | Назначение |
|---|---|
| `users` | Пользователи системы |
| `messages` | Письма (отправитель, получатель, тема, текст) |
| `message_files` | Вложения к письмам |
| `message_statuses` | Статус доставки/прочтения письма |
| `emails` | Адреса электронной почты |
| `users_emails` | Связь пользователя с важными контактами (importance, category) |
| `deleted_users_log` | Журнал удалённых пользователей (заполняется триггером) |

### 2.2 MongoDB (документная, NoSQL)

**Роль:** модель чтения — денормализованные проекции для быстрых запросов.

**Версия:** MongoDB 7  
**Драйвер Go:** `go.mongodb.org/mongo-driver/v2`  
**База данных:** `email_service`  
**Коллекция:** `users`

Каждый документ в MongoDB — это один пользователь со всеми его данными, вложенными прямо в документ:

```json
{
  "id": 1,
  "email": "user@example.com",
  "num_messages": 3,
  "created_at": "2025-01-01T00:00:00Z",
  "important_contacts": [
    { "email": "boss@corp.com", "category": "work", "importance": 9 }
  ],
  "messages": [
    { "id": 42, "subject": "Hello", "text": "Hi there", "created_at": "..." }
  ]
}
```

### 2.3 Apache Kafka (брокер событий)

**Роль:** шина событий между write- и read-стороной.  
**Версия:** Apache Kafka 3.9.0 (KRaft — без ZooKeeper)  
**Топик:** `sync_topic`

Kafka не является базой данных, но обеспечивает надёжную доставку событий с гарантией порядка.

---

## 3. Архитектура системы

```
┌─────────────────┐    HTTP     ┌──────────────────┐
│   HTTP Client   │ ──────────► │   Backend API    │
└─────────────────┘             └────────┬─────────┘
                                         │ INSERT/SELECT
                                         ▼
                                ┌──────────────────┐
                                │   PostgreSQL     │  ← write model (источник истины)
                                └────────┬─────────┘
                                         │ публикует события
                                         ▼
                                ┌──────────────────┐
                                │   Apache Kafka   │  ← топик sync_topic
                                └────────┬─────────┘
                                         │ потребляет события
                                         ▼
                                ┌──────────────────┐
                                │  CQRS Sync Svc   │
                                └────────┬─────────┘
                                         │ UPSERT/DELETE
                                         ▼
                                ┌──────────────────┐
                                │    MongoDB       │  ← read model (проекции)
                                └──────────────────┘
```

**Поток данных (пример: создание пользователя):**

1. `POST /users` → Backend API вставляет запись в `users` (PostgreSQL)
2. Backend публикует событие `user_created` в топик Kafka
3. CQRS Sync Service читает событие из Kafka
4. `KafkaConsumer.DispatchEvent` маршрутизирует его в `UserService.HandleUserCreated`
5. `MongoUserViewRepository.CreateUserView` выполняет upsert в MongoDB
6. Клиент при последующих GET-запросах читает готовую проекцию из MongoDB

---

## 4. Функции приложения

Приложение реализует **7 бизнес-функций**, сгруппированных по трём доменам:

### 4.1 Управление пользователями

**Функция 1 — Создание пользователя** (`HandleUserCreated`)

При получении события `user_created` создаётся документ в MongoDB с пустыми списками контактов и сообщений. Операция идемпотентна: повторное событие игнорируется (`$setOnInsert`).

```go
// cqrs/internal/service/user_service.go
func (us *UserService) HandleUserCreated(ctx context.Context, event events.UserCreatedPayload) error {
    user := view.UserView{
        ID:                event.ID,
        Email:             event.Email,
        NumMessages:       0,
        CreatedAt:         event.CreatedAt,
        ImportantContacts: []view.ImportantContactView{},
        Messages:          []view.MessageView{},
    }
    return us.userViewRepo.CreateUserView(ctx, user)
}
```

**Функция 2 — Удаление пользователя** (`HandleUserDeleted`)

При получении события `user_deleted` документ пользователя удаляется из MongoDB. Параллельно PostgreSQL-триггер `trg_users_audit_delete` записывает строку в `deleted_users_log`.

```go
// cqrs/internal/service/user_service.go
func (us *UserService) HandleUserDeleted(ctx context.Context, event events.UserDeletedPayload) error {
    return us.userViewRepo.DeleteUserView(ctx, event.ID)
}
```

### 4.2 Управление контактами (важные письма)

**Функция 3 — Добавление контакта** (`HandleEmailAddedToUser`)

Добавляет объект контакта в массив `important_contacts` документа пользователя в MongoDB (`$push`).

**Функция 4 — Обновление контакта** (`HandleEmailUpdatedForUser`)

Обновляет поля `category` и `importance` существующего контакта по его адресу (`$set` на элементе массива).

**Функция 5 — Удаление контакта** (`HandleEmailRemovedFromUser`)

Удаляет контакт из массива по адресу электронной почты (`$pull`).

```go
// cqrs/internal/service/email_service.go
func (es *EmailService) HandleEmailAddedToUser(ctx context.Context, event events.EmailAddedPayload) error {
    email := view.ImportantContactView{
        Email:      event.Address,
        Category:   event.Category,
        Importance: event.Importance,
    }
    return es.mongoEmailViewRepo.AddEmailToUser(ctx, event.UserID, email)
}
```

### 4.3 Управление сообщениями

**Функция 6 — Добавление сообщения** (`HandleMessageAddedToUser`)

Добавляет сводку письма в массив `messages` и увеличивает счётчик `num_messages` атомарно (`$push` + `$inc`).

**Функция 7 — Удаление сообщения** (`HandleMessageDeletedFromUser`)

Удаляет письмо из массива и уменьшает счётчик (`$pull` + `$inc`). Если письмо не найдено, но пользователь существует — операция идемпотентна.

```go
// cqrs/internal/storage/message_repo.go
func (r *MongoMessageViewRepository) AddMessageToUser(ctx context.Context, userID int, message view.MessageView) error {
    result, err := r.DB.Collection("users").UpdateOne(
        ctx,
        bson.M{"id": userID},
        bson.M{
            "$push": bson.M{"messages": message},
            "$inc":  bson.M{"num_messages": 1},
        },
    )
    // ...
}
```

---

## 5. CRUD-операции

### 5.1 PostgreSQL — Write Model

#### CREATE — вставка пользователя

```sql
INSERT INTO users (email, created_at) VALUES ('user@example.com', NOW())
RETURNING id;
```

При вставке письма триггер `trg_messages_auto_status` **автоматически** создаёт запись `'sent'` в `message_statuses`:

```sql
INSERT INTO messages (external_id, sender_id, receiver_id, subject, text, date_sent)
VALUES ('uuid-...', 1, 2, 'Привет', 'Текст письма', NOW());
-- триггер сам добавит: INSERT INTO message_statuses (message_id, status) VALUES (новый_id, 'sent')
```

#### READ — чтение пользователя с его письмами

```sql
SELECT id, email, created_at FROM users WHERE id = $1;

SELECT id, external_id, sender_id, receiver_id, subject, text, date_sent, created_at
FROM messages
WHERE sender_id = $1 OR receiver_id = $1
ORDER BY date_sent DESC;
```

#### UPDATE — изменение важности контакта

```sql
UPDATE users_emails
SET importance = $1, category = $2
WHERE user_id = $3 AND email_id = $4;
-- триггер trg_users_emails_clamp_importance зажмёт importance в [0, 10]
```

#### DELETE — удаление пользователя

```sql
DELETE FROM users WHERE id = $1;
-- FK ON DELETE CASCADE автоматически удаляет: messages (если sender), users_emails
-- триггер trg_users_audit_delete запишет строку в deleted_users_log
```

Проверить журнал удалений:

```sql
SELECT * FROM deleted_users_log ORDER BY deleted_at DESC;
```

### 5.2 MongoDB — Read Model

#### CREATE — создание документа пользователя

```go
// Idempotent upsert: создаёт документ только если его нет
collection.UpdateOne(
    ctx,
    bson.M{"id": userView.ID},
    bson.M{"$setOnInsert": userView},
    options.UpdateOne().SetUpsert(true),
)
```

#### READ — получение проекции пользователя

```go
var userView view.UserView
r.DB.Collection("users").
    FindOne(ctx, bson.M{"id": id}).
    Decode(&userView)
```

#### UPDATE — добавление/изменение вложенных объектов

```go
// Добавить контакт в массив
collection.UpdateOne(ctx,
    bson.M{"id": userID},
    bson.M{"$push": bson.M{"important_contacts": email}},
)

// Обновить поля вложенного объекта по значению (positional operator $)
collection.UpdateOne(ctx,
    bson.M{"id": userID, "important_contacts.email": email.Email},
    bson.M{"$set": bson.M{
        "important_contacts.$.category":   email.Category,
        "important_contacts.$.importance": email.Importance,
    }},
)

// Атомарно добавить сообщение и увеличить счётчик
collection.UpdateOne(ctx,
    bson.M{"id": userID},
    bson.M{
        "$push": bson.M{"messages": message},
        "$inc":  bson.M{"num_messages": 1},
    },
)
```

#### DELETE — удаление документа и вложенных объектов

```go
// Удалить весь документ
collection.DeleteOne(ctx, bson.M{"id": id})

// Удалить контакт из массива ($pull)
collection.UpdateOne(ctx,
    bson.M{"id": userID},
    bson.M{"$pull": bson.M{"important_contacts": bson.M{"email": emailAddress}}},
)

// Удалить сообщение из массива и уменьшить счётчик
collection.UpdateOne(ctx,
    bson.M{"id": userID, "messages.id": messageID},
    bson.M{
        "$pull": bson.M{"messages": bson.M{"id": messageID}},
        "$inc":  bson.M{"num_messages": -1},
    },
)
```

---

## 6. PostgreSQL Триггеры

В `init.sql` определены 5 триггеров, обеспечивающих целостность данных на уровне базы без участия приложения:

| # | Триггер | Таблица | Событие | Описание |
|---|---|---|---|---|
| 1 | `trg_messages_auto_status` | `messages` | AFTER INSERT | Автоматически создаёт начальный статус `'sent'` |
| 2 | `trg_message_statuses_updated_at` | `message_statuses` | BEFORE UPDATE | Обновляет `updated_at` при каждом изменении статуса |
| 3 | `trg_users_emails_clamp_importance` | `users_emails` | BEFORE INSERT/UPDATE | Зажимает `importance` в диапазон [0, 10] |
| 4 | `trg_messages_no_self_send` | `messages` | BEFORE INSERT | Запрещает отправку письма самому себе |
| 5 | `trg_users_audit_delete` | `users` | AFTER DELETE | Записывает удалённого пользователя в `deleted_users_log` |

**Пример — триггер автоматического статуса:**

```sql
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
```

**Пример — триггер защиты от самоотправки:**

```sql
CREATE OR REPLACE FUNCTION trg_fn_messages_no_self_send()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.sender_id = NEW.receiver_id THEN
        RAISE EXCEPTION 'sender and receiver must be different (user_id=%)', NEW.sender_id;
    END IF;
    RETURN NEW;
END;
$$;
```

---

## 7. Типы событий Kafka

Система обрабатывает 7 типов событий:

| Тип события | Payload | Действие в MongoDB |
|---|---|---|
| `user_created` | `{id, email, created_at}` | Upsert документа пользователя |
| `user_deleted` | `{id}` | DeleteOne документа |
| `email_added` | `{user_id, email_id, address, category, importance}` | `$push` в `important_contacts` |
| `email_updated` | `{user_id, address, new_category, new_importance}` | `$set` на элементе массива |
| `email_removed` | `{user_id, address}` | `$pull` из `important_contacts` |
| `message_created` | `{user_id, message_id, subject, content, date_sent}` | `$push` в `messages` + `$inc num_messages` |
| `message_deleted` | `{user_id, message_id}` | `$pull` из `messages` + `$inc num_messages: -1` |

Конверт события:

```json
{
  "event_id":   "uuid",
  "event_type": "user_created",
  "version":    1,
  "timestamp":  "2025-01-01T12:00:00Z",
  "payload":    { "id": 1, "email": "user@example.com", "created_at": "..." }
}
```

---

## 8. Структура проекта

```
db_sync/
├── cqrs/                        # CQRS Sync Service (Read-сторона)
│   ├── cmd/db_sync/main.go      # Точка входа
│   ├── init.sql                 # Схема PostgreSQL + триггеры
│   ├── internal/
│   │   ├── app/app.go           # Инициализация и главный цикл
│   │   ├── application/
│   │   │   └── events/          # Типы событий (UserCreatedPayload и др.)
│   │   ├── service/             # Бизнес-логика (UserService, EmailService, MessageService)
│   │   ├── storage/             # Репозитории (PostgreSQL + MongoDB)
│   │   ├── transport/kafka/     # Kafka Consumer + диспетчер событий
│   │   ├── middleware/          # Декоратор логирования событий
│   │   ├── view/                # Типы MongoDB-документов (UserView и др.)
│   │   └── config/              # Конфигурация подключений
├── backend/                     # Backend Write API (Write-сторона)
│   └── internal/
│       ├── transport/http/      # HTTP-обработчики (users, emails, messages)
│       ├── storage/             # Репозитории PostgreSQL (bun ORM)
│       └── kafka/               # Kafka Producer
├── docker-compose.yml           # PostgreSQL + MongoDB + Kafka + Mongo Express
├── Taskfile.yml                 # Команды запуска
└── go.work                      # Go workspace (монорепозиторий)
```

---

## 9. Запуск приложения

**Требования:** Docker (или Podman), Go 1.26+, Task

```bash
# 1. Скопировать конфигурацию
cp .env.example .env

# 2. Запустить инфраструктуру (PostgreSQL, MongoDB, Kafka)
task up:docker

# 3. Запустить CQRS Sync Service
task cqrs:run
```

**Доступные интерфейсы после запуска:**

| Сервис | Адрес |
|---|---|
| PostgreSQL | `localhost:5432` |
| MongoDB | `localhost:27017` |
| Kafka | `localhost:9092` |
| Mongo Express (UI) | `http://localhost:8081` |

**Переменные окружения (`.env`):**

```env
POSTGRES_USER=postgres
POSTGRES_PASSWORD=secret
POSTGRES_DB=email_service

MONGO_INITDB_ROOT_USERNAME=admin
MONGO_INITDB_ROOT_PASSWORD=secret

KAFKA_TOPIC=sync_topic
KAFKA_GROUP_ID=db_sync_group
```

---

## 10. Ключевые технические решения

### Идемпотентность

Событие `user_created` может прийти повторно (при перезапуске или сбое Kafka consumer). MongoDB-операция использует `$setOnInsert` — при дубликате документ не перезаписывается.

### Атомарность обновлений MongoDB

Добавление письма и увеличение счётчика выполняются в одной операции `UpdateOne` с двумя операторами (`$push` и `$inc`). Это исключает рассинхронизацию счётчика с реальным размером массива.

### Каскадные удаления PostgreSQL + аудит

FK `ON DELETE CASCADE` на `message_files` и `message_statuses` обеспечивает ссылочную целостность. Триггер `trg_users_audit_delete` дополнительно сохраняет удалённого пользователя в `deleted_users_log` — данные не теряются.

### Разделение моделей

Write model (PostgreSQL) нормализован — данные не дублируются. Read model (MongoDB) денормализован — вся информация о пользователе в одном документе, нет JOIN-ов при чтении.

---

## 11. Использованные технологии

| Технология | Версия | Роль |
|---|---|---|
| Go | 1.26+ | Язык разработки |
| PostgreSQL | 16 | Write model (SQL) |
| MongoDB | 7 | Read model (NoSQL документная БД) |
| Apache Kafka | 3.9.0 | Брокер событий |
| `jackc/pgx` | v5 | PostgreSQL-драйвер |
| `jmoiron/sqlx` | latest | SQL-утилиты |
| `uptrace/bun` | latest | ORM для PostgreSQL |
| `mongo-driver` | v2 | MongoDB-драйвер |
| `segmentio/kafka-go` | latest | Kafka-клиент |
| Docker / Podman | — | Контейнеризация инфраструктуры |

---

## 12. Выводы

В ходе работы реализована система синхронизации данных между двумя разнородными базами данных — **PostgreSQL** (реляционная SQL) и **MongoDB** (документная NoSQL) — через шину событий Apache Kafka по паттерну CQRS.

**Достигнутые цели:**
- Используются 2 базы данных разных типов (SQL + NoSQL)
- Реализованы 7 функций обработки доменных событий
- Покрыты все CRUD-операции для обеих баз данных
- Добавлены 5 PostgreSQL-триггеров, обеспечивающих целостность данных на уровне БД
- Обеспечена идемпотентность обработки событий
- Разделены модели записи и чтения для независимого масштабирования
