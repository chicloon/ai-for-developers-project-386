# Call Booking — Сервис записи на звонок

Сервис, где владельцы публикуют правила доступности, а клиенты выбирают свободные 30-минутные слоты и записываются на звонок.

### Hexlet tests and linter status:
[![Actions Status](https://github.com/Chicloon/ai-for-developers-project-386/actions/workflows/hexlet-check.yml/badge.svg)](https://github.com/Chicloon/ai-for-developers-project-386/actions)

## Стек

- **API Contract:** TypeSpec → OpenAPI 3.0
- **Backend:** Go 1.25, `chi` router, `pgx` драйвер
- **Frontend:** Next.js (App Router), Mantine UI
- **Database:** PostgreSQL 16
- **Infrastructure:** Docker, docker-compose

## Запуск через Docker (полный стек)

### Быстрый старт

```bash
docker compose up -d
```

Будут запущены:
- **PostgreSQL**: порт 5432
- **Go API**: порт 8080
- **Next.js**: порт 3000
- **Nginx**: порт 80 (проксирует API и Web)

### Доступ

- **Frontend**: http://localhost:80
- **API**: http://localhost:80/api

### Остановка

```bash
docker compose down
```

## Запуск отдельных сервисов

### 1. База данных

```bash
docker compose up -d db
docker compose build api
docker compose up -d api
```

API будет доступен на `http://localhost:8080`.

### 2. Запустить фронтенд

```bash
cd web && npm install
cd web && npm run dev
```

Фронтенд будет доступен на `http://localhost:3000`.

## Запуск без Docker

### Требования

- Go 1.25+
- Node.js 18+
- PostgreSQL 16

### 1. База данных

Создайте базу данных и примените миграции:

```bash
createdb call_booking
psql -d call_booking -f migrations/001_initial.up.sql
```

### 2. Бэкенд

```bash
go build ./cmd/server
./server
```

Сервер запустится на `http://localhost:8080`.

Переменные окружения:
- `DATABASE_URL` — строка подключения к PostgreSQL (по умолчанию: `postgres://postgres:postgres@localhost:5432/call_booking?sslmode=disable`)
- `PORT` — порт сервера (по умолчанию: `8080`)

Для production с **Supabase** и TLS (`sslmode=require`):

- **Прямое подключение** (`db.<project-ref>.supabase.co:5432`, пользователь `postgres`) — по умолчанию **только IPv6**. На **Render** и других IPv4-only платформах не подойдёт без IPv4 add-on в Supabase.
- **Session pooler** (Supabase Dashboard → **Connect** → **Session mode**) — **IPv4 + IPv6**, подходит для долгоживущего бэкенда на Render. Строка вида:  
  `postgresql://postgres.<project-ref>:<db-password>@aws-0-<region>.pooler.supabase.com:5432/postgres?sslmode=require`
- **Transaction pooler** (порт `6543`) — чаще для serverless; в режиме transaction у Postgres pooler **нет поддержки prepared statements**; для Go `pgx` надёжнее **session pooler** на `5432` или отдельная настройка simple protocol (см. [документацию Supabase](https://supabase.com/docs/guides/database/connecting-to-postgres)).

Скопируйте готовую строку из Dashboard (Connect) и вставьте в переменную `DATABASE_URL` на Render.

### 3. Фронтенд

```bash
cd web && npm install
cd web && npm run dev
```

Фронтенд запустится на `http://localhost:3000`.

Клиент обращается к API только по путям `/api/*` (тот же origin). Запросы проксируются на бэкенд в runtime через [`web/app/api/[[...path]]/route.ts`](web/app/api/[[...path]]/route.ts) и переменную `API_PROXY_URL` (в dev по умолчанию `http://localhost:8080` из окружения при `npm run dev` — при необходимости задайте в `.env.local`).

## Деплой на Render (API + Web)

Нужны **два** Web Service: образ из [`Dockerfile`](Dockerfile) (только API) и из [`Dockerfile.web`](Dockerfile.web) (Next.js).

1. **Blueprint (рекомендуется):** в корне репозитория есть [`render.yaml`](render.yaml). В Render: **New → Blueprint** → выберите репозиторий. После создания сервисов укажите **`DATABASE_URL`** для API (Supabase Session pooler, см. выше). `JWT_SECRET` для API сгенерируется сам; для Web **`API_PROXY_URL`** подставится из публичного URL API.
2. **Вручную:** создайте два сервиса **Docker** с теми же Dockerfile, регион и план по желанию. У API: `DATABASE_URL`, `JWT_SECRET` (≥ 32 символа). У Web: `API_PROXY_URL` = полный URL API, например `https://call-booking-api.onrender.com` (без слэша в конце).

**Открывайте в браузере URL веб-сервиса** — там интерфейс и тот же `/api/*` через прокси. URL только API показывает заглушку на `/` и отвечает на `/health` и `/api/*`.

Render задаёт **`PORT`**; Next.js слушает его в [`Dockerfile.web`](Dockerfile.web), Go — из `PORT` в [`cmd/server/main.go`](cmd/server/main.go).

## Деплой на VPS (Docker)

### Требования

- VPS с Docker и Docker Compose
- GitHub репозиторий с настроенными secrets

### 1. Настройка GitHub Secrets

В репозитории GitHub → Settings → Secrets and variables → Actions:

| Secret | Описание |
|--------|----------|
| `VPS_HOST` | IP адрес сервера |
| `VPS_USER` | Username для SSH |
| `VPS_SSH_KEY` | Приватный SSH ключ |
| `GH_TOKEN` | GitHub PAT с правами `write:packages` |

### 2. Настройка сервера

```bash
# Подключиться по SSH
ssh user@server

# Установить Docker (если нет)
curl -fsSL https://get.docker.com | sh

# Клонировать репозиторий
git clone https://github.com/ваш-ник/ai-for-developers-project-386.git
cd ai-for-developers-project-386
```

### 3. Запуск

```bash
docker compose up -d
```

Все сервисы будут доступны:
- **Frontend**: `http://SERVER_IP`
- **API**: `http://SERVER_IP/api`

### 4. CI/CD

При пуше в ветку `main` GitHub Actions автоматически:
1. Билдит Docker образы (API и Web)
2. Пушит в GitHub Container Registry
3. Подключается к VPS по SSH
4. Выполняет `docker compose pull && docker compose up -d`

---

## Тесты

### Go

```bash
go test ./... -v
```

Тесты требуют базу данных `call_booking_test` на `localhost:5432`.

### Фронтенд

```bash
cd web && npm run build
```

## TypeSpec

```bash
cd typespec && npm install
cd typespec && npx tsp compile .
```

Сгенерированный OpenAPI: `typespec/tsp-output/openapi3/openapi.yaml`.

## API Endpoints

| Метод    | Путь                          | Описание               |
|----------|-------------------------------|------------------------|
| `GET`    | `/api/availability-rules`     | Список правил          |
| `POST`   | `/api/availability-rules`     | Создать правило        |
| `PUT`    | `/api/availability-rules/{id}`| Обновить правило       |
| `DELETE` | `/api/availability-rules/{id}`| Удалить правило        |
| `POST`   | `/api/blocked-days`           | Заблокировать день     |
| `GET`    | `/api/blocked-days`           | Список заблокированных |
| `DELETE` | `/api/blocked-days/{id}`      | Разблокировать день    |
| `GET`    | `/api/slots?date=YYYY-MM-DD`  | Слоты на дату          |
| `POST`   | `/api/bookings`               | Создать бронирование   |
| `GET`    | `/api/bookings`               | Список бронирований    |
| `DELETE` | `/api/bookings/{id}`          | Отменить бронирование  |
