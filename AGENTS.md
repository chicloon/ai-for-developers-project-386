# AGENTS.md — Call Booking Project

## Project Overview

Call Booking service (monorepo) inspired by Cal.com. Users can publish availability schedules, manage visibility through **fixed visibility groups** (Family, Work, Friends) or **public profile toggle**, and book 30-minute slots. JWT authentication, visibility groups for access control, no external calendar integrations.

## Tech Stack

- **API Contract:** TypeSpec (generates OpenAPI 3.0)
- **Backend:** Go 1.22+, `chi` router, `pgx` database driver, JWT auth
- **Frontend:** Next.js (App Router), Mantine UI (`@mantine/core`, `@mantine/dates`)
- **Database:** PostgreSQL 16
- **Infrastructure:** Docker, docker-compose

## Project Structure

```
typespec/              # TypeSpec API contract
  main.tsp             # Entry point
  models.tsp           # Data models (User, Schedule, Booking, Group, etc.)
  auth.tsp             # Auth endpoints (/api/auth/*)
  users.tsp            # User endpoints (/api/users/*)
  schedules.tsp        # Schedule endpoints (/api/my/schedules)
  groups.tsp           # Visibility group endpoints (/api/my/groups) — list only, members management
  bookings.tsp         # Booking endpoints (/api/my/bookings)
  users.tsp            # User endpoints (/api/users/*) — list, get, slots, updateMe

cmd/server/            # Go entry point
internal/
  api/                 # HTTP handlers & chi router
    router.go          # Main router with middleware
    handlers_auth.go   # Auth handlers (register, login, me)
    handlers_users.go  # User handlers (list, get, slots)
    handlers_schedules.go  # Schedule CRUD handlers
    handlers_groups.go     # Group member handlers (fixed groups — no CRUD for groups)
    handlers_bookings.go   # Booking handlers
  auth/                # JWT authentication
    jwt.go             # Token generation and validation
    middleware.go      # Auth middleware
  db/                  # DB connection & migrations
  models/              # Go structs
migrations/            # SQL migration files
  001_initial.up.sql   # Initial schema (users, schedules, groups, bookings)
  001_initial.down.sql # Rollback
  002_add_is_public.up.sql    # Add is_public flag to users, remove public group type
  002_add_is_public.down.sql  # Rollback
web/                   # Next.js frontend
  app/                 # App Router pages
  components/          # Mantine UI components
  lib/                 # API client
```

## Development Commands

### TypeSpec
```bash
cd typespec && npm install          # Install deps
cd typespec && npx tsp compile .    # Generate OpenAPI spec
```

### Go Backend
```bash
go build ./cmd/server               # Build binary
go test ./... -v                    # Run all tests
go test ./internal/api/... -v       # Run API tests only
go test ./internal/auth/... -v      # Run auth tests
```

### Next.js Frontend
```bash
cd web && npm install               # Install deps
cd web && npm run dev               # Start dev server
cd web && npm run build             # Production build
```

### Docker
```bash
docker compose up -d                # Start DB + API
docker compose down                 # Stop all
```

## Architecture Rules

### API
- All endpoints under `/api/*` prefix
- Public routes: `/api/auth/*` (no JWT required)
- Protected routes: all other `/api/*` (JWT required via `Authorization` header)
- POST endpoints return `201 Created`
- DELETE endpoints return `204 No Content`
- PUT endpoints return `200 OK`
- Errors return `{ "error": "message" }` with appropriate status code
- JWT token format: `Bearer <token>` in Authorization header

### Groups API (Fixed Groups)
Groups are automatically created for each user on registration. Users cannot create/delete groups — only manage members.
- `GET /api/my/groups` — List user's 3 fixed groups (Family, Work, Friends)
- `GET /api/my/groups/{id}/members` — List members of a group
- `POST /api/my/groups/{id}/members` — Add member by email
- `DELETE /api/my/groups/{id}/members/{memberId}` — Remove member

### User Profile API
- `PUT /api/users/me` — Update current user profile (`isPublic`, `name`)

### Database Schema
- **users**: id, email, password_hash, name, **is_public**, created_at, updated_at
- **schedules**: id, user_id, type (recurring|one-time), day_of_week, date, start_time, end_time, is_blocked
- **visibility_groups**: id, owner_id, name, visibility_level (**family|work|friends**) — 3 fixed groups per user, auto-created on registration
- **group_members**: id, group_id, member_id, added_by, added_at
- **bookings**: id, schedule_id, booker_id, owner_id, status, created_at, cancelled_at

### Visibility Model
- **Public profile** (`is_public = true`): User is visible to all authenticated users in the catalog, anyone can book
- **Private profile** (`is_public = false`): User is only visible to members of their fixed groups (Family, Work, Friends)
- **Fixed groups**: Each user automatically gets 3 groups on registration — "Семья" (family), "Работа" (work), "Друзья" (friends)

### Database
- Migrations run via `embed.FS` from `internal/db/migrate.go`
- No migration tracking table — migrations are idempotent (`CREATE TABLE IF NOT EXISTS`)
- UUIDs generated by `gen_random_uuid()`
- Foreign keys with CASCADE delete where appropriate

### Go Conventions
- Handlers use `pgxpool.Pool` directly (no repository layer — YAGNI)
- JWT auth middleware extracts user ID to context
- `jsonResponse()` and `jsonError()` for consistent response formatting
- Helper functions `ptrInt32()` and `strPtr()` for optional fields in tests

### Frontend Conventions
- All components use Mantine UI — no Tailwind CSS classes
- Use `@mantine/core` for layout (`Paper`, `Stack`, `Group`, `Button`, `TextInput`, `Text`, `Title`)
- Use `@mantine/dates` for `DatePickerInput`
- API client in `web/lib/api.ts` — all fetch calls go through it
- Components are client components (`"use client"`)
- Russian language for all UI text

## Testing

- TDD: write tests before implementation
- Tests use `httptest.NewRecorder` with real DB connection
- Tests skip if database unavailable (`t.Skipf`)
- Test DB: `call_booking_test` on localhost:5432
- Run `go test ./... -v` before committing Go changes

## Git Workflow

- Commit after each plan completion
- Commit message format: `feat: <description>` or `fix: <description>`
- Never commit `node_modules/`, `.next/`, `tsp-output/`, `vendor/`, `.env`
- Run linter/tests before committing
