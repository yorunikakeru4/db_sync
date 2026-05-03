# Monorepo

Корень репозитория теперь общий для нескольких сервисов.

## Layout

- `cqrs/` — отдельный Go-модуль с текущим CQRS sync-сервисом
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

## Дальше

Следующий сервис можно добавлять рядом с `cqrs/` как отдельный модуль, например `backend/`.
