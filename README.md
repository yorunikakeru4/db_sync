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

## Frontend

`vue/` contains a small Vue + Vite console for planned CQRS HTTP endpoints.

Run locally:

```bash
cd vue
pnpm install
cp .env.example .env.local
# set VITE_API_BASE_URL in .env.local to your backend URL
pnpm dev
```

Build:

```bash
cd vue
pnpm build
```
