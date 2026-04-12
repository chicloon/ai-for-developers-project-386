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

Образы API и Web по умолчанию тянутся с **GitHub Container Registry** (`ghcr.io`). Если пакеты **приватные**, анонимный `docker pull` завершится с **401 Unauthorized**. Возможны два варианта:

**Вариант 1 — локальная сборка (без входа в GHCR):**

```bash
docker compose up -d --build
```

**Вариант 2 — скачивать готовые образы с GHCR:** создайте [Personal Access Token](https://github.com/settings/tokens) с правом **`read:packages`**, затем:

```bash
echo YOUR_GITHUB_TOKEN | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin
docker compose pull
docker compose up -d
```

Переменная `DATABASE_URL` для сервиса `api` необязательна: по умолчанию используется строка из `docker-compose.yml` (подключение к контейнеру `db`). Для своей БД задайте `DATABASE_URL` в окружении или в файле `.env` рядом с compose.

При **Supabase** и других облачных Postgres обычно нужен **`?sslmode=require`** в строке подключения. API при старте **обязательно применяет миграции** из `migrations/` — если миграция не прошла, процесс завершится с ошибкой в логах (раньше сервер мог подняться без схемы и возвращать «database error» при регистрации).

Будут запущены:
- **PostgreSQL**: порт 5432
- **Go API**: порт 8080
- **Next.js**: порт 3000
- **Nginx**: порт 80 (проксирует API и Web)

### Доступ

- **Frontend**: http://localhost:80
- **API через Nginx**: http://localhost:80/api
- **API напрямую** (контейнер `api` проброшен на хост): http://localhost:8080/ и http://localhost:8080/health

### Остановка

```bash
docker compose down
```

### Бесконечный индикатор загрузки

Если в браузере крутится загрузка на защищённых страницах (`/my/...`), чаще всего не отвечает **API** или запрос `/api/auth/me` не доходит (сохранённый JWT в `localStorage`). Проверьте: `docker compose ps`, логи `docker compose logs api`, доступность `http://localhost/api` или при `npm run dev` — что бэкенд слушает порт **8080** и прокси в `web/next.config.ts` указывает на него. После исправления при необходимости выйдите из сессии или очистите `localStorage` для сайта.

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

### 3. Фронтенд

```bash
cd web && npm install
cd web && npm run dev
```

Фронтенд запустится на `http://localhost:3000`.

Клиент обращается к API только по путям `/api/*` (тот же origin). В dev Next.js проксирует их на бэкенд (`web/next.config.ts`). Отдельный `.env.local` для URL API не требуется.

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
| `DATABASE_URL` | Строка подключения PostgreSQL для API при деплое (передаётся в `docker compose` из CI; локально можно не задавать — используется значение по умолчанию) |

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
4. Экспортирует `DATABASE_URL` из секрета репозитория и выполняет `docker compose pull && docker compose up -d`

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
