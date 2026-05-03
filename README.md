# Monorepo

Корень репозитория теперь общий для нескольких сервисов.

## Layout

- `cqrs/` — отдельный Go-модуль с текущим CQRS sync-сервисом
- `backend/` — отдельный Go-модуль с write-side HTTP API
- `docker-compose.yml` — общая локальная инфраструктура
- `.env` / `.env.example` — общие переменные для локального запуска

## Команды

Инфраструктура:

```bash
task init
task up:docker
```

CQRS сервис:

```bash
task cqrs:build
task cqrs:run
task cqrs:test
task cqrs:test:integration
```

Прямой вход в модуль тоже рабочий:

```bash
cd cqrs
task build
task run
```

Backend API:

```bash
task backend:build
task backend:run
task backend:test
task backend:test:integration
```

Оба сервиса вместе:

```bash
task dev
```

## Дальше

Сервисы уже разделены на `cqrs/` и `backend/`, оба живут на общей инфраструктуре из корня.
