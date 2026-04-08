# Система авторизации и multi-user бронирования - План реализации

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Добавить JWT-аутентификацию, группы видимости и multi-user бронирование в Call Booking сервис

**Architecture:** Полный редизайн с новой схемой БД (users, schedules, visibility_groups, bookings), JWT middleware для защиты API, AuthProvider на фронтенде для управления сессией

**Tech Stack:** Go 1.22, chi router, golang-jwt, bcrypt, PostgreSQL, Next.js 14, Mantine UI, TypeScript

---

## Файловая структура

### Backend (Go)
```
internal/
├── models/
│   └── models.go          # Добавить: User, AuthRequest, AuthResponse, Schedule, VisibilityGroup, etc.
├── auth/
│   ├── password.go        # Хеширование/проверка паролей
│   ├── jwt.go             # Генерация/валидация JWT
│   └── middleware.go      # JWT middleware для chi
├── api/
│   ├── handlers_auth.go   # POST /auth/register, POST /auth/login, GET /auth/me
│   ├── handlers_users.go  # GET /users, GET /users/:id, GET /users/:id/slots
│   ├── handlers_schedules.go # GET/POST/PUT/DELETE /my/schedules
│   ├── handlers_groups.go # CRUD для групп и членов
│   ├── handlers_bookings.go # GET/POST/DELETE для бронирований
│   └── router.go          # Обновить: добавить JWT middleware, новые роуты
```

### Frontend (Next.js)
```
web/
├── app/
│   ├── (auth)/
│   │   ├── login/
│   │   │   └── page.tsx
│   │   └── register/
│   │       └── page.tsx
│   ├── (app)/
│   │   ├── layout.tsx
│   │   ├── page.tsx              # Каталог пользователей
│   │   ├── users/
│   │   │   └── [id]/
│   │   │       └── page.tsx
│   │   └── my/
│   │       ├── schedule/
│   │       │   └── page.tsx
│   │       ├── groups/
│   │       │   └── page.tsx
│   │       └── bookings/
│   │           └── page.tsx
├── components/
│   ├── auth/
│   │   ├── AuthProvider.tsx
│   │   └── ProtectedRoute.tsx
│   └── navigation/
│       └── AppShell.tsx
└── lib/
    └── api.ts                    # Обновить: добавить auth, users, groups endpoints
```

### Database
```
migrations/
├── 001_initial.down.sql          # Удалить старые таблицы
└── 001_initial.up.sql            # Создать новые таблицы
```

---

## Задачи

### Task 1: Database Migration - New Schema

**Files:**
- Modify: `migrations/001_initial.up.sql`
- Modify: `migrations/001_initial.down.sql`

**Цель:** Создать новую схему БД с таблицами users, schedules, visibility_groups, group_members, bookings

- [ ] **Step 1: Обновить up-миграцию**

Заменить содержимое `migrations/001_initial.up.sql`:

```sql
-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Visibility groups
CREATE TABLE IF NOT EXISTS visibility_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    visibility_level VARCHAR(20) NOT NULL CHECK (visibility_level IN ('family', 'work', 'friends', 'public')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Group members
CREATE TABLE IF NOT EXISTS group_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES visibility_groups(id) ON DELETE CASCADE,
    member_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    added_by UUID NOT NULL REFERENCES users(id),
    added_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(group_id, member_id)
);

-- Schedules (replaces availability_rules + blocked_days)
CREATE TABLE IF NOT EXISTS schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('recurring', 'one-time')),
    day_of_week INT CHECK (day_of_week BETWEEN 0 AND 6),
    date DATE,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    is_blocked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CHECK (
        (type = 'recurring' AND day_of_week IS NOT NULL AND date IS NULL) OR
        (type = 'one-time' AND date IS NOT NULL AND day_of_week IS NULL)
    )
);

-- Bookings
CREATE TABLE IF NOT EXISTS bookings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id UUID NOT NULL REFERENCES schedules(id),
    booker_id UUID NOT NULL REFERENCES users(id),
    owner_id UUID NOT NULL REFERENCES users(id),
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'cancelled')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    cancelled_at TIMESTAMP,
    cancelled_by UUID REFERENCES users(id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_schedules_user_id ON schedules(user_id);
CREATE INDEX IF NOT EXISTS idx_schedules_user_date ON schedules(user_id, date);
CREATE INDEX IF NOT EXISTS idx_bookings_booker_id ON bookings(booker_id);
CREATE INDEX IF NOT EXISTS idx_bookings_owner_id ON bookings(owner_id);
CREATE INDEX IF NOT EXISTS idx_bookings_schedule_id ON bookings(schedule_id);
CREATE INDEX IF NOT EXISTS idx_group_members_member_id ON group_members(member_id);
CREATE INDEX IF NOT EXISTS idx_visibility_groups_owner_id ON visibility_groups(owner_id);
```

- [ ] **Step 2: Обновить down-миграцию**

Заменить содержимое `migrations/001_initial.down.sql`:

```sql
DROP TABLE IF EXISTS bookings;
DROP TABLE IF EXISTS schedules;
DROP TABLE IF EXISTS group_members;
DROP TABLE IF EXISTS visibility_groups;
DROP TABLE IF EXISTS users;
```

- [ ] **Step 3: Протестировать миграции**

Run:
```bash
docker compose down -v
docker compose up -d db
sleep 3
psql postgresql://user:password@localhost:5432/call_booking -c "\dt"
```

Expected: Таблицы users, schedules, visibility_groups, group_members, bookings созданы

- [ ] **Step 4: Commit**

```bash
git add migrations/
git commit -m "feat: add database migration for auth system"
```

---

### Task 2: Go Models

**Files:**
- Modify: `internal/models/models.go`

**Цель:** Добавить структуры для User, AuthRequest, AuthResponse, Schedule, VisibilityGroup, Booking

- [ ] **Step 1: Добавить новые модели**

В конец файла `internal/models/models.go` добавить (перед последней строкой с пустыми переменными):

```go
// User represents a registered user
type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	PasswordHash string `json:"-"` // never expose in JSON
	CreatedAt    string `json:"createdAt,omitempty"`
	UpdatedAt    string `json:"updatedAt,omitempty"`
}

// AuthRequest for login/register
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse returned after successful auth
type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// Schedule represents user's availability (replaces AvailabilityRule)
type Schedule struct {
	ID          string `json:"id"`
	UserID      string `json:"userId"`
	Type        string `json:"type"`
	DayOfWeek   *int32 `json:"dayOfWeek,omitempty"`
	Date        *string `json:"date,omitempty"`
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime"`
	IsBlocked   bool   `json:"isBlocked"`
	CreatedAt   string `json:"createdAt,omitempty"`
}

type CreateScheduleRequest struct {
	Type      string  `json:"type"`
	DayOfWeek *int32  `json:"dayOfWeek,omitempty"`
	Date      *string `json:"date,omitempty"`
	StartTime string  `json:"startTime"`
	EndTime   string  `json:"endTime"`
	IsBlocked bool    `json:"isBlocked"`
}

// VisibilityGroup for access control
type VisibilityGroup struct {
	ID               string `json:"id"`
	OwnerID          string `json:"ownerId"`
	Name             string `json:"name"`
	VisibilityLevel  string `json:"visibilityLevel"`
	CreatedAt        string `json:"createdAt,omitempty"`
}

type CreateGroupRequest struct {
	Name            string `json:"name"`
	VisibilityLevel string `json:"visibilityLevel"`
}

type AddMemberRequest struct {
	Email  *string `json:"email,omitempty"`
	UserID *string `json:"userId,omitempty"`
}

// GroupMember with user info
type GroupMember struct {
	ID       string `json:"id"`
	GroupID  string `json:"groupId"`
	Member   User   `json:"member"`
	AddedBy  string `json:"addedBy"`
	AddedAt  string `json:"addedAt"`
}

// Booking with user info
type Booking struct {
	ID           string  `json:"id"`
	ScheduleID   string  `json:"scheduleId"`
	Booker       User    `json:"booker"`
	Owner        User    `json:"owner"`
	Date         string  `json:"date"`
	StartTime    string  `json:"startTime"`
	EndTime      string  `json:"endTime"`
	Status       string  `json:"status"`
	CreatedAt    string  `json:"createdAt,omitempty"`
	CancelledAt  *string `json:"cancelledAt,omitempty"`
}

type CreateBookingRequest struct {
	OwnerID    string `json:"ownerId"`
	ScheduleID string `json:"scheduleId"`
}

// Slot for public display
type Slot struct {
	ID        string `json:"id"`
	Date      string `json:"date"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	IsBooked  bool   `json:"isBooked"`
}
```

- [ ] **Step 2: Убрать старые модели (опционально)**

Удалить или закомментировать старые модели AvailabilityRule, BlockedDay, Booking если они не используются.

- [ ] **Step 3: Проверить компиляцию**

Run:
```bash
go build ./cmd/server
```

Expected: Компиляция без ошибок

- [ ] **Step 4: Commit**

```bash
git add internal/models/models.go
git commit -m "feat: add models for auth, schedules, groups, bookings"
```

---

### Task 3: Auth Package - Password & JWT

**Files:**
- Create: `internal/auth/password.go`
- Create: `internal/auth/jwt.go`

**Цель:** Создать пакет для хеширования паролей и работы с JWT

- [ ] **Step 1: Создать password.go**

```go
package auth

import (
	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword compares a password with a hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
```

- [ ] **Step 2: Создать jwt.go**

```go
package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("your-secret-key-change-in-production")

// Claims represents JWT claims
type Claims struct {
	UserID string `json:"userId"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// GenerateToken creates a new JWT token for a user
func GenerateToken(userID, email string) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateToken validates a JWT token and returns claims
func ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// SetSecret allows setting a custom JWT secret (for testing)
func SetSecret(secret string) {
	jwtSecret = []byte(secret)
}
```

- [ ] **Step 3: Установить зависимости**

Run:
```bash
go get golang.org/x/crypto/bcrypt
go get github.com/golang-jwt/jwt/v5
```

- [ ] **Step 4: Проверить компиляцию**

Run:
```bash
go build ./internal/auth
```

Expected: Компиляция без ошибок

- [ ] **Step 5: Commit**

```bash
git add internal/auth/
git commit -m "feat: add auth package with password hashing and JWT"
```

---

### Task 4: JWT Middleware

**Files:**
- Create: `internal/auth/middleware.go`

**Цель:** Middleware для проверки JWT в защищённых эндпоинтах

- [ ] **Step 1: Создать middleware.go**

```go
package auth

import (
	"context"
	"net/http"
	"strings"

	"call-booking/internal/models"
)

// ContextKey for storing user ID in context
type ContextKey string

const UserIDKey ContextKey = "userID"
const UserEmailKey ContextKey = "userEmail"

// Middleware validates JWT token and adds user info to context
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			jsonError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		// Extract Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			jsonError(w, http.StatusUnauthorized, "invalid authorization header format")
			return
		}

		// Validate token
		claims, err := ValidateToken(parts[1])
		if err != nil {
			jsonError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		// Add user info to context
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UserEmailKey, claims.Email)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

// GetUserEmail extracts user email from context
func GetUserEmail(ctx context.Context) string {
	if email, ok := ctx.Value(UserEmailKey).(string); ok {
		return email
	}
	return ""
}

func jsonError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"error":"` + msg + `"}`))
}
```

- [ ] **Step 2: Проверить компиляцию**

Run:
```bash
go build ./internal/auth
```

Expected: Компиляция без ошибок

- [ ] **Step 3: Commit**

```bash
git add internal/auth/middleware.go
git commit -m "feat: add JWT middleware"
```

---

### Task 5: Auth Handlers

**Files:**
- Create: `internal/api/handlers_auth.go`

**Цель:** Реализовать POST /auth/register, POST /auth/login, GET /auth/me

- [ ] **Step 1: Создать handlers_auth.go**

```go
package api

import (
	"encoding/json"
	"net/http"

	"call-booking/internal/auth"
	"call-booking/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type authHandler struct {
	pool *pgxpool.Pool
}

func authRouter(pool *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()
	h := &authHandler{pool: pool}

	r.Post("/register", h.register)
	r.Post("/login", h.login)
	r.With(auth.Middleware).Get("/me", h.me)

	return r
}

func (h *authHandler) register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" || req.Name == "" {
		jsonError(w, http.StatusBadRequest, "email, password and name are required")
		return
	}

	// Check if user exists
	var exists bool
	err := h.pool.QueryRow(r.Context(),
		"SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)",
		req.Email).Scan(&exists)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if exists {
		jsonError(w, http.StatusConflict, "user with this email already exists")
		return
	}

	// Hash password
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	// Create user
	var user models.User
	err = h.pool.QueryRow(r.Context(),
		"INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3) RETURNING id, email, name, created_at",
		req.Email, hash, req.Name).
		Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Generate token
	token, err := auth.GenerateToken(user.ID, user.Email)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	jsonResponse(w, http.StatusCreated, models.AuthResponse{
		Token: token,
		User:  user,
	})
}

func (h *authHandler) login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" {
		jsonError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	// Find user
	var user models.User
	var passwordHash string
	err := h.pool.QueryRow(r.Context(),
		"SELECT id, email, name, password_hash FROM users WHERE email = $1",
		req.Email).
		Scan(&user.ID, &user.Email, &user.Name, &passwordHash)
	if err != nil {
		if err == pgx.ErrNoRows {
			jsonError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Check password
	if !auth.CheckPassword(req.Password, passwordHash) {
		jsonError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	// Generate token
	token, err := auth.GenerateToken(user.ID, user.Email)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	jsonResponse(w, http.StatusOK, models.AuthResponse{
		Token: token,
		User:  user,
	})
}

func (h *authHandler) me(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	var user models.User
	err := h.pool.QueryRow(r.Context(),
		"SELECT id, email, name, created_at FROM users WHERE id = $1",
		userID).
		Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt)
	if err != nil {
		jsonError(w, http.StatusNotFound, "user not found")
		return
	}

	jsonResponse(w, http.StatusOK, user)
}
```

- [ ] **Step 2: Проверить компиляцию**

Run:
```bash
go build ./internal/api
```

Expected: Компиляция без ошибок

- [ ] **Step 3: Commit**

```bash
git add internal/api/handlers_auth.go
git commit -m "feat: add auth handlers (register, login, me)"
```

---

### Task 6: Users Handlers

**Files:**
- Create: `internal/api/handlers_users.go`

**Цель:** Реализовать GET /users (каталог), GET /users/:id, GET /users/:id/slots

- [ ] **Step 1: Создать handlers_users.go**

```go
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"call-booking/internal/auth"
	"call-booking/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type usersHandler struct {
	pool *pgxpool.Pool
}

func usersRouter(pool *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()
	h := &usersHandler{pool: pool}

	r.Use(auth.Middleware)
	r.Get("/", h.list)
	r.Get("/{id}", h.get)
	r.Get("/{id}/slots", h.slots)

	return r
}

// list returns users visible to current user
func (h *usersHandler) list(w http.ResponseWriter, r *http.Request) {
	currentUserID := auth.GetUserID(r.Context())

	query := `
		SELECT DISTINCT u.id, u.email, u.name FROM users u
		LEFT JOIN visibility_groups vg ON vg.owner_id = u.id
		LEFT JOIN group_members gm ON gm.group_id = vg.id AND gm.member_id = $1
		WHERE u.id != $1
		  AND (
			EXISTS (SELECT 1 FROM visibility_groups WHERE owner_id = u.id AND visibility_level = 'public')
			OR gm.member_id IS NOT NULL
		  )
		ORDER BY u.name
	`

	rows, err := h.pool.Query(r.Context(), query, currentUserID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Email, &user.Name); err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		users = append(users, user)
	}

	if users == nil {
		users = []models.User{}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"users": users})
}

// get returns user profile
func (h *usersHandler) get(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	currentUserID := auth.GetUserID(r.Context())

	// Check visibility
	if !h.canSeeUser(r.Context(), currentUserID, userID) {
		jsonError(w, http.StatusForbidden, "you don't have access to this user")
		return
	}

	var user models.User
	err := h.pool.QueryRow(r.Context(),
		"SELECT id, email, name FROM users WHERE id = $1",
		userID).
		Scan(&user.ID, &user.Email, &user.Name)
	if err != nil {
		jsonError(w, http.StatusNotFound, "user not found")
		return
	}

	jsonResponse(w, http.StatusOK, user)
}

// slots returns available slots for a user on a specific date
func (h *usersHandler) slots(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	currentUserID := auth.GetUserID(r.Context())
	date := r.URL.Query().Get("date")

	if date == "" {
		jsonError(w, http.StatusBadRequest, "date parameter is required")
		return
	}

	// Check visibility
	if !h.canSeeUser(r.Context(), currentUserID, userID) {
		jsonError(w, http.StatusForbidden, "you don't have access to this user")
		return
	}

	// Get schedules for the date
	slots, err := h.getSlotsForDate(r.Context(), userID, date)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"slots": slots})
}

// canSeeUser checks if current user can see target user
func (h *usersHandler) canSeeUser(ctx context.Context, currentUserID, targetUserID string) bool {
	if currentUserID == targetUserID {
		return true
	}

	var visible bool
	query := `
		SELECT EXISTS (
			SELECT 1 FROM visibility_groups vg
			LEFT JOIN group_members gm ON gm.group_id = vg.id AND gm.member_id = $1
			WHERE vg.owner_id = $2
			  AND (vg.visibility_level = 'public' OR gm.member_id IS NOT NULL)
		)
	`
	err := h.pool.QueryRow(ctx, query, currentUserID, targetUserID).Scan(&visible)
	if err != nil {
		return false
	}
	return visible
}

// getSlotsForDate generates 30-min slots from schedules
func (h *usersHandler) getSlotsForDate(ctx context.Context, userID, date string) ([]models.Slot, error) {
	// Get schedules for the date
	rows, err := h.pool.Query(ctx, `
		SELECT id, start_time, end_time, is_blocked 
		FROM schedules 
		WHERE user_id = $1 
		  AND (
			  (type = 'one-time' AND date = $2) 
			  OR 
			  (type = 'recurring' AND day_of_week = EXTRACT(DOW FROM $2::date))
		  )
		ORDER BY start_time
	`, userID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type schedule struct {
		id         string
		startTime  string
		endTime    string
		isBlocked  bool
	}

	var schedules []schedule
	for rows.Next() {
		var s schedule
		if err := rows.Scan(&s.id, &s.startTime, &s.endTime, &s.isBlocked); err != nil {
			return nil, err
		}
		schedules = append(schedules, s)
	}

	// Get booked slots
	bookedRows, err := h.pool.Query(ctx, `
		SELECT s.start_time 
		FROM bookings b
		JOIN schedules s ON s.id = b.schedule_id
		WHERE b.owner_id = $1 
		  AND s.date = $2
		  AND b.status = 'active'
	`, userID, date)
	if err != nil {
		return nil, err
	}
	defer bookedRows.Close()

	bookedTimes := make(map[string]bool)
	for bookedRows.Next() {
		var startTime string
		if err := bookedRows.Scan(&startTime); err != nil {
			return nil, err
		}
		bookedTimes[startTime] = true
	}

	// Generate 30-min slots
	var slots []models.Slot
	slotDuration := 30 * time.Minute

	for _, s := range schedules {
		if s.isBlocked {
			continue
		}

		start, _ := time.Parse("15:04:05", s.startTime)
		end, _ := time.Parse("15:04:05", s.endTime)

		for current := start; current.Before(end); current = current.Add(slotDuration) {
			slotEnd := current.Add(slotDuration)
			if slotEnd.After(end) {
				break
			}

			slotStartStr := current.Format("15:04")
			slots = append(slots, models.Slot{
				ID:        s.id + "_" + slotStartStr,
				Date:      date,
				StartTime: slotStartStr,
				EndTime:   slotEnd.Format("15:04"),
				IsBooked:  bookedTimes[slotStartStr],
			})
		}
	}

	return slots, nil
}
```

- [ ] **Step 2: Проверить компиляцию**

Run:
```bash
go build ./internal/api
```

Expected: Компиляция без ошибок

- [ ] **Step 3: Commit**

```bash
git add internal/api/handlers_users.go
git commit -m "feat: add users handlers (catalog, profile, slots)"
```

---

### Task 7: Schedules Handlers

**Files:**
- Create: `internal/api/handlers_schedules.go`

**Цель:** Реализовать CRUD для /my/schedules

- [ ] **Step 1: Создать handlers_schedules.go**

```go
package api

import (
	"encoding/json"
	"net/http"

	"call-booking/internal/auth"
	"call-booking/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type schedulesHandler struct {
	pool *pgxpool.Pool
}

func schedulesRouter(pool *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()
	h := &schedulesHandler{pool: pool}

	r.Use(auth.Middleware)
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Put("/{id}", h.update)
	r.Delete("/{id}", h.delete)

	return r
}

func (h *schedulesHandler) list(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	rows, err := h.pool.Query(r.Context(),
		"SELECT id, user_id, type, day_of_week, date, start_time, end_time, is_blocked, created_at FROM schedules WHERE user_id = $1 ORDER BY created_at DESC",
		userID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var schedules []models.Schedule
	for rows.Next() {
		var s models.Schedule
		var dayOfWeek *int32
		var date *string
		if err := rows.Scan(&s.ID, &s.UserID, &s.Type, &dayOfWeek, &date, &s.StartTime, &s.EndTime, &s.IsBlocked, &s.CreatedAt); err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.DayOfWeek = dayOfWeek
		s.Date = date
		schedules = append(schedules, s)
	}

	if schedules == nil {
		schedules = []models.Schedule{}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"schedules": schedules})
}

func (h *schedulesHandler) create(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	var req models.CreateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate
	if req.Type != "recurring" && req.Type != "one-time" {
		jsonError(w, http.StatusBadRequest, "type must be 'recurring' or 'one-time'")
		return
	}
	if req.StartTime == "" || req.EndTime == "" {
		jsonError(w, http.StatusBadRequest, "start_time and end_time are required")
		return
	}

	var s models.Schedule
	var dayOfWeek *int32
	var date *string

	err := h.pool.QueryRow(r.Context(),
		"INSERT INTO schedules (user_id, type, day_of_week, date, start_time, end_time, is_blocked) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, user_id, type, day_of_week, date, start_time, end_time, is_blocked, created_at",
		userID, req.Type, req.DayOfWeek, req.Date, req.StartTime, req.EndTime, req.IsBlocked).
		Scan(&s.ID, &s.UserID, &s.Type, &dayOfWeek, &date, &s.StartTime, &s.EndTime, &s.IsBlocked, &s.CreatedAt)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.DayOfWeek = dayOfWeek
	s.Date = date

	jsonResponse(w, http.StatusCreated, s)
}

func (h *schedulesHandler) update(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	scheduleID := chi.URLParam(r, "id")

	var req models.CreateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var s models.Schedule
	var dayOfWeek *int32
	var date *string

	err := h.pool.QueryRow(r.Context(),
		"UPDATE schedules SET type=$1, day_of_week=$2, date=$3, start_time=$4, end_time=$5, is_blocked=$6 WHERE id=$7 AND user_id=$8 RETURNING id, user_id, type, day_of_week, date, start_time, end_time, is_blocked, created_at",
		req.Type, req.DayOfWeek, req.Date, req.StartTime, req.EndTime, req.IsBlocked, scheduleID, userID).
		Scan(&s.ID, &s.UserID, &s.Type, &dayOfWeek, &date, &s.StartTime, &s.EndTime, &s.IsBlocked, &s.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			jsonError(w, http.StatusNotFound, "schedule not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.DayOfWeek = dayOfWeek
	s.Date = date

	jsonResponse(w, http.StatusOK, s)
}

func (h *schedulesHandler) delete(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	scheduleID := chi.URLParam(r, "id")

	result, err := h.pool.Exec(r.Context(),
		"DELETE FROM schedules WHERE id = $1 AND user_id = $2",
		scheduleID, userID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if result.RowsAffected() == 0 {
		jsonError(w, http.StatusNotFound, "schedule not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 2: Проверить компиляцию**

Run:
```bash
go build ./internal/api
```

Expected: Компиляция без ошибок

- [ ] **Step 3: Commit**

```bash
git add internal/api/handlers_schedules.go
git commit -m "feat: add schedules handlers (CRUD)"
```

---

### Task 8: Groups Handlers

**Files:**
- Create: `internal/api/handlers_groups.go`

**Цель:** Реализовать CRUD для групп и управление членами

- [ ] **Step 1: Создать handlers_groups.go**

```go
package api

import (
	"encoding/json"
	"net/http"

	"call-booking/internal/auth"
	"call-booking/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type groupsHandler struct {
	pool *pgxpool.Pool
}

func groupsRouter(pool *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()
	h := &groupsHandler{pool: pool}

	r.Use(auth.Middleware)
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Put("/{id}", h.update)
	r.Delete("/{id}", h.delete)
	r.Get("/{id}/members", h.listMembers)
	r.Post("/{id}/members", h.addMember)
	r.Delete("/{id}/members/{memberId}", h.removeMember)

	return r
}

func (h *groupsHandler) list(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	rows, err := h.pool.Query(r.Context(),
		"SELECT id, owner_id, name, visibility_level, created_at FROM visibility_groups WHERE owner_id = $1 ORDER BY created_at DESC",
		userID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var groups []models.VisibilityGroup
	for rows.Next() {
		var g models.VisibilityGroup
		if err := rows.Scan(&g.ID, &g.OwnerID, &g.Name, &g.VisibilityLevel, &g.CreatedAt); err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		groups = append(groups, g)
	}

	if groups == nil {
		groups = []models.VisibilityGroup{}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"groups": groups})
}

func (h *groupsHandler) create(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	var req models.CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.VisibilityLevel == "" {
		jsonError(w, http.StatusBadRequest, "name and visibility_level are required")
		return
	}

	var g models.VisibilityGroup
	err := h.pool.QueryRow(r.Context(),
		"INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, $2, $3) RETURNING id, owner_id, name, visibility_level, created_at",
		userID, req.Name, req.VisibilityLevel).
		Scan(&g.ID, &g.OwnerID, &g.Name, &g.VisibilityLevel, &g.CreatedAt)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusCreated, g)
}

func (h *groupsHandler) update(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	groupID := chi.URLParam(r, "id")

	var req models.CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var g models.VisibilityGroup
	err := h.pool.QueryRow(r.Context(),
		"UPDATE visibility_groups SET name=$1, visibility_level=$2 WHERE id=$3 AND owner_id=$4 RETURNING id, owner_id, name, visibility_level, created_at",
		req.Name, req.VisibilityLevel, groupID, userID).
		Scan(&g.ID, &g.OwnerID, &g.Name, &g.VisibilityLevel, &g.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			jsonError(w, http.StatusNotFound, "group not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, g)
}

func (h *groupsHandler) delete(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	groupID := chi.URLParam(r, "id")

	result, err := h.pool.Exec(r.Context(),
		"DELETE FROM visibility_groups WHERE id = $1 AND owner_id = $2",
		groupID, userID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if result.RowsAffected() == 0 {
		jsonError(w, http.StatusNotFound, "group not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *groupsHandler) listMembers(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	groupID := chi.URLParam(r, "id")

	// Verify group ownership
	var ownerID string
	err := h.pool.QueryRow(r.Context(),
		"SELECT owner_id FROM visibility_groups WHERE id = $1",
		groupID).Scan(&ownerID)
	if err != nil {
		jsonError(w, http.StatusNotFound, "group not found")
		return
	}
	if ownerID != userID {
		jsonError(w, http.StatusForbidden, "you don't own this group")
		return
	}

	rows, err := h.pool.Query(r.Context(),
		`SELECT gm.id, gm.group_id, u.id, u.email, u.name, gm.added_by, gm.added_at 
		 FROM group_members gm
		 JOIN users u ON u.id = gm.member_id
		 WHERE gm.group_id = $1`,
		groupID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var members []models.GroupMember
	for rows.Next() {
		var m models.GroupMember
		if err := rows.Scan(&m.ID, &m.GroupID, &m.Member.ID, &m.Member.Email, &m.Member.Name, &m.AddedBy, &m.AddedAt); err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		members = append(members, m)
	}

	if members == nil {
		members = []models.GroupMember{}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"members": members})
}

func (h *groupsHandler) addMember(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	groupID := chi.URLParam(r, "id")

	var req models.AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Verify group ownership
	var ownerID string
	err := h.pool.QueryRow(r.Context(),
		"SELECT owner_id FROM visibility_groups WHERE id = $1",
		groupID).Scan(&ownerID)
	if err != nil {
		jsonError(w, http.StatusNotFound, "group not found")
		return
	}
	if ownerID != userID {
		jsonError(w, http.StatusForbidden, "you don't own this group")
		return
	}

	// Find member by email or user_id
	var memberID string
	if req.Email != nil {
		err = h.pool.QueryRow(r.Context(),
			"SELECT id FROM users WHERE email = $1",
			*req.Email).Scan(&memberID)
		if err != nil {
			jsonError(w, http.StatusNotFound, "user not found")
			return
		}
	} else if req.UserID != nil {
		memberID = *req.UserID
		// Verify user exists
		var exists bool
		err = h.pool.QueryRow(r.Context(),
			"SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)",
			memberID).Scan(&exists)
		if err != nil || !exists {
			jsonError(w, http.StatusNotFound, "user not found")
			return
		}
	} else {
		jsonError(w, http.StatusBadRequest, "email or userId is required")
		return
	}

	// Add member
	_, err = h.pool.Exec(r.Context(),
		"INSERT INTO group_members (group_id, member_id, added_by) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING",
		groupID, memberID, userID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *groupsHandler) removeMember(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	groupID := chi.URLParam(r, "id")
	memberID := chi.URLParam(r, "memberId")

	// Verify group ownership
	var ownerID string
	err := h.pool.QueryRow(r.Context(),
		"SELECT owner_id FROM visibility_groups WHERE id = $1",
		groupID).Scan(&ownerID)
	if err != nil {
		jsonError(w, http.StatusNotFound, "group not found")
		return
	}
	if ownerID != userID {
		jsonError(w, http.StatusForbidden, "you don't own this group")
		return
	}

	result, err := h.pool.Exec(r.Context(),
		"DELETE FROM group_members WHERE group_id = $1 AND member_id = $2",
		groupID, memberID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if result.RowsAffected() == 0 {
		jsonError(w, http.StatusNotFound, "member not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 2: Проверить компиляцию**

Run:
```bash
go build ./internal/api
```

Expected: Компиляция без ошибок

- [ ] **Step 3: Commit**

```bash
git add internal/api/handlers_groups.go
git commit -m "feat: add groups handlers (CRUD + members)"
```

---

### Task 9: Bookings Handlers

**Files:**
- Create: `internal/api/handlers_bookings.go`

**Цель:** Реализовать CRUD для бронирований

- [ ] **Step 1: Создать handlers_bookings.go**

```go
package api

import (
	"encoding/json"
	"net/http"

	"call-booking/internal/auth"
	"call-booking/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type bookingsHandler struct {
	pool *pgxpool.Pool
}

func bookingsRouter(pool *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()
	h := &bookingsHandler{pool: pool}

	r.Use(auth.Middleware)
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Delete("/{id}", h.cancel)

	return r
}

func (h *bookingsHandler) list(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	query := `
		SELECT b.id, b.schedule_id, 
		       bu.id, bu.email, bu.name,
		       ou.id, ou.email, ou.name,
		       s.date, s.start_time, s.end_time,
		       b.status, b.created_at, b.cancelled_at
		FROM bookings b
		JOIN users bu ON bu.id = b.booker_id
		JOIN users ou ON ou.id = b.owner_id
		JOIN schedules s ON s.id = b.schedule_id
		WHERE b.booker_id = $1 OR b.owner_id = $1
		ORDER BY b.created_at DESC
	`

	rows, err := h.pool.Query(r.Context(), query, userID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var bookings []models.Booking
	for rows.Next() {
		var b models.Booking
		var cancelledAt *string
		if err := rows.Scan(
			&b.ID, &b.ScheduleID,
			&b.Booker.ID, &b.Booker.Email, &b.Booker.Name,
			&b.Owner.ID, &b.Owner.Email, &b.Owner.Name,
			&b.Date, &b.StartTime, &b.EndTime,
			&b.Status, &b.CreatedAt, &cancelledAt); err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		b.CancelledAt = cancelledAt
		bookings = append(bookings, b)
	}

	if bookings == nil {
		bookings = []models.Booking{}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"bookings": bookings})
}

func (h *bookingsHandler) create(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	var req models.CreateBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.OwnerID == "" || req.ScheduleID == "" {
		jsonError(w, http.StatusBadRequest, "owner_id and schedule_id are required")
		return
	}

	// Check if user can see the owner
	canSee, err := h.canSeeUser(r.Context(), userID, req.OwnerID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !canSee {
		jsonError(w, http.StatusForbidden, "you don't have access to this user")
		return
	}

	// Check if slot is already booked
	var exists bool
	err = h.pool.QueryRow(r.Context(),
		"SELECT EXISTS(SELECT 1 FROM bookings WHERE schedule_id = $1 AND status = 'active')",
		req.ScheduleID).Scan(&exists)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if exists {
		jsonError(w, http.StatusConflict, "this slot is already booked")
		return
	}

	// Create booking
	var b models.Booking
	err = h.pool.QueryRow(r.Context(),
		`INSERT INTO bookings (schedule_id, booker_id, owner_id) 
		 VALUES ($1, $2, $3) 
		 RETURNING id, schedule_id, booker_id, owner_id, status, created_at`,
		req.ScheduleID, userID, req.OwnerID).
		Scan(&b.ID, &b.ScheduleID, &b.Booker.ID, &b.Owner.ID, &b.Status, &b.CreatedAt)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Get user info
	h.pool.QueryRow(r.Context(),
		"SELECT email, name FROM users WHERE id = $1", userID).Scan(&b.Booker.Email, &b.Booker.Name)
	h.pool.QueryRow(r.Context(),
		"SELECT email, name FROM users WHERE id = $1", req.OwnerID).Scan(&b.Owner.Email, &b.Owner.Name)

	// Get schedule info
	h.pool.QueryRow(r.Context(),
		"SELECT date, start_time, end_time FROM schedules WHERE id = $1", req.ScheduleID).
		Scan(&b.Date, &b.StartTime, &b.EndTime)

	jsonResponse(w, http.StatusCreated, b)
}

func (h *bookingsHandler) cancel(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	bookingID := chi.URLParam(r, "id")

	// Verify booking exists and user has permission
	var bookerID, ownerID string
	err := h.pool.QueryRow(r.Context(),
		"SELECT booker_id, owner_id FROM bookings WHERE id = $1",
		bookingID).Scan(&bookerID, &ownerID)
	if err != nil {
		if err == pgx.ErrNoRows {
			jsonError(w, http.StatusNotFound, "booking not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Check permission: booker or owner can cancel
	if bookerID != userID && ownerID != userID {
		jsonError(w, http.StatusForbidden, "you don't have permission to cancel this booking")
		return
	}

	// Update booking status
	_, err = h.pool.Exec(r.Context(),
		"UPDATE bookings SET status = 'cancelled', cancelled_at = NOW(), cancelled_by = $1 WHERE id = $2",
		userID, bookingID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// canSeeUser checks if current user can see target user
func (h *bookingsHandler) canSeeUser(ctx context.Context, currentUserID, targetUserID string) (bool, error) {
	if currentUserID == targetUserID {
		return true, nil
	}

	var visible bool
	query := `
		SELECT EXISTS (
			SELECT 1 FROM visibility_groups vg
			LEFT JOIN group_members gm ON gm.group_id = vg.id AND gm.member_id = $1
			WHERE vg.owner_id = $2
			  AND (vg.visibility_level = 'public' OR gm.member_id IS NOT NULL)
		)
	`
	err := h.pool.QueryRow(ctx, query, currentUserID, targetUserID).Scan(&visible)
	return visible, err
}
```

- [ ] **Step 2: Проверить компиляцию**

Run:
```bash
go build ./internal/api
```

Expected: Компиляция без ошибок

- [ ] **Step 3: Commit**

```bash
git add internal/api/handlers_bookings.go
git commit -m "feat: add bookings handlers (list, create, cancel)"
```

---

### Task 10: Update Router

**Files:**
- Modify: `internal/api/router.go`

**Цель:** Обновить роутер для использования новых handlers

- [ ] **Step 1: Обновить router.go**

Заменить содержимое `internal/api/router.go`:

```go
package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(pool *pgxpool.Pool) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Public routes (no JWT required)
	r.Mount("/api/auth", authRouter(pool))

	// Protected routes (JWT required)
	r.Route("/api", func(r chi.Router) {
		r.Mount("/users", usersRouter(pool))
		r.Mount("/my/schedules", schedulesRouter(pool))
		r.Mount("/my/groups", groupsRouter(pool))
		r.Mount("/my/bookings", bookingsRouter(pool))
	})

	return r
}
```

- [ ] **Step 2: Проверить компиляцию**

Run:
```bash
go build ./cmd/server
```

Expected: Компиляция без ошибок

- [ ] **Step 3: Commit**

```bash
git add internal/api/router.go
git commit -m "feat: update router with new auth endpoints"
```

---

### Task 11: Frontend - Auth Types

**Files:**
- Modify: `web/lib/api.ts`

**Цель:** Добавить TypeScript типы и API методы для auth

- [ ] **Step 1: Обновить api.ts с auth типами**

В начало файла `web/lib/api.ts` добавить:

```typescript
// Auth types
export interface User {
  id: string;
  email: string;
  name: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  name: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

// Token storage
let authToken: string | null = null;

export function setAuthToken(token: string | null) {
  authToken = token;
  if (token) {
    localStorage.setItem('auth_token', token);
  } else {
    localStorage.removeItem('auth_token');
  }
}

export function getAuthToken(): string | null {
  if (!authToken) {
    authToken = localStorage.getItem('auth_token');
  }
  return authToken;
}

// Auth API
export async function register(data: RegisterRequest): Promise<AuthResponse> {
  const res = await fetch("/api/auth/register", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error || "Registration failed");
  }
  const result = await res.json();
  setAuthToken(result.token);
  return result;
}

export async function login(data: LoginRequest): Promise<AuthResponse> {
  const res = await fetch("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error || "Login failed");
  }
  const result = await res.json();
  setAuthToken(result.token);
  return result;
}

export async function getMe(): Promise<User> {
  const token = getAuthToken();
  const res = await fetch("/api/auth/me", {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) throw new Error("Failed to get user");
  return res.json();
}

export function logout() {
  setAuthToken(null);
}
```

- [ ] **Step 2: Проверить компиляцию TypeScript**

Run:
```bash
cd web && npx tsc --noEmit
```

Expected: Без ошибок TypeScript

- [ ] **Step 3: Commit**

```bash
git add web/lib/api.ts
git commit -m "feat: add auth API and types"
```

---

### Task 12: Frontend - AuthProvider

**Files:**
- Create: `web/components/auth/AuthProvider.tsx`

**Цель:** Создать React Context для управления авторизацией

- [ ] **Step 1: Создать AuthProvider.tsx**

```typescript
"use client";

import React, { createContext, useContext, useEffect, useState } from "react";
import { User, getMe, getAuthToken, setAuthToken, logout as apiLogout } from "@/lib/api";

interface AuthContextType {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  login: (token: string, user: User) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    // Check for existing token on mount
    const token = getAuthToken();
    if (token) {
      getMe()
        .then((userData) => {
          setUser(userData);
        })
        .catch(() => {
          // Token invalid, clear it
          apiLogout();
        })
        .finally(() => {
          setIsLoading(false);
        });
    } else {
      setIsLoading(false);
    }
  }, []);

  const login = (token: string, userData: User) => {
    setAuthToken(token);
    setUser(userData);
  };

  const logout = () => {
    apiLogout();
    setUser(null);
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        isLoading,
        isAuthenticated: !!user,
        login,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
```

- [ ] **Step 2: Проверить компиляцию**

Run:
```bash
cd web && npx tsc --noEmit
```

Expected: Без ошибок TypeScript

- [ ] **Step 3: Commit**

```bash
git add web/components/auth/AuthProvider.tsx
git commit -m "feat: add AuthProvider component"
```

---

### Task 13: Frontend - Login Page

**Files:**
- Create: `web/app/(auth)/login/page.tsx`

**Цель:** Создать страницу входа

- [ ] **Step 1: Создать структуру директорий**

```bash
mkdir -p web/app/(auth)/login
```

- [ ] **Step 2: Создать login/page.tsx**

```typescript
"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import {
  Paper,
  Title,
  TextInput,
  PasswordInput,
  Button,
  Stack,
  Text,
  Anchor,
} from "@mantine/core";
import { login } from "@/lib/api";
import { useAuth } from "@/components/auth/AuthProvider";

export default function LoginPage() {
  const router = useRouter();
  const { login: authLogin } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const response = await login({ email, password });
      authLogin(response.token, response.user);
      router.push("/");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ошибка входа");
    } finally {
      setLoading(false);
    }
  };

  return (
    <Paper p="xl" maw={400} mx="auto" mt={100} withBorder>
      <Title order={2} mb="lg" ta="center">
        Вход
      </Title>

      <form onSubmit={handleSubmit}>
        <Stack>
          <TextInput
            label="Email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
          />

          <PasswordInput
            label="Пароль"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />

          {error && (
            <Text c="red" size="sm">
              {error}
            </Text>
          )}

          <Button type="submit" loading={loading} fullWidth>
            Войти
          </Button>

          <Text ta="center" size="sm">
            Нет аккаунта?{" "}
            <Anchor href="/register">Зарегистрироваться</Anchor>
          </Text>
        </Stack>
      </form>
    </Paper>
  );
}
```

- [ ] **Step 3: Проверить компиляцию**

Run:
```bash
cd web && npx tsc --noEmit
```

Expected: Без ошибок TypeScript

- [ ] **Step 4: Commit**

```bash
git add web/app/(auth)/login/
git commit -m "feat: add login page"
```

---

### Task 14: Frontend - Register Page

**Files:**
- Create: `web/app/(auth)/register/page.tsx`

**Цель:** Создать страницу регистрации

- [ ] **Step 1: Создать структуру директорий**

```bash
mkdir -p web/app/(auth)/register
```

- [ ] **Step 2: Создать register/page.tsx**

```typescript
"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import {
  Paper,
  Title,
  TextInput,
  PasswordInput,
  Button,
  Stack,
  Text,
  Anchor,
} from "@mantine/core";
import { register } from "@/lib/api";
import { useAuth } from "@/components/auth/AuthProvider";

export default function RegisterPage() {
  const router = useRouter();
  const { login: authLogin } = useAuth();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const response = await register({ name, email, password });
      authLogin(response.token, response.user);
      router.push("/");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ошибка регистрации");
    } finally {
      setLoading(false);
    }
  };

  return (
    <Paper p="xl" maw={400} mx="auto" mt={100} withBorder>
      <Title order={2} mb="lg" ta="center">
        Регистрация
      </Title>

      <form onSubmit={handleSubmit}>
        <Stack>
          <TextInput
            label="Имя"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
          />

          <TextInput
            label="Email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
          />

          <PasswordInput
            label="Пароль"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />

          {error && (
            <Text c="red" size="sm">
              {error}
            </Text>
          )}

          <Button type="submit" loading={loading} fullWidth>
            Зарегистрироваться
          </Button>

          <Text ta="center" size="sm">
            Уже есть аккаунт?{" "}
            <Anchor href="/login">Войти</Anchor>
          </Text>
        </Stack>
      </form>
    </Paper>
  );
}
```

- [ ] **Step 3: Проверить компиляцию**

Run:
```bash
cd web && npx tsc --noEmit
```

Expected: Без ошибок TypeScript

- [ ] **Step 4: Commit**

```bash
git add web/app/(auth)/register/
git commit -m "feat: add register page"
```

---

### Task 15: Frontend - ProtectedRoute

**Files:**
- Create: `web/components/auth/ProtectedRoute.tsx`

**Цель:** Создать компонент для защиты роутов

- [ ] **Step 1: Создать ProtectedRoute.tsx**

```typescript
"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { Loader, Center } from "@mantine/core";
import { useAuth } from "./AuthProvider";

export default function ProtectedRoute({
  children,
}: {
  children: React.ReactNode;
}) {
  const { isAuthenticated, isLoading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      router.push("/login");
    }
  }, [isLoading, isAuthenticated, router]);

  if (isLoading) {
    return (
      <Center h="100vh">
        <Loader />
      </Center>
    );
  }

  if (!isAuthenticated) {
    return null;
  }

  return <>{children}</>;
}
```

- [ ] **Step 2: Проверить компиляцию**

Run:
```bash
cd web && npx tsc --noEmit
```

Expected: Без ошибок TypeScript

- [ ] **Step 3: Commit**

```bash
git add web/components/auth/ProtectedRoute.tsx
git commit -m "feat: add ProtectedRoute component"
```

---

### Task 16: Frontend - Navigation (AppShell)

**Files:**
- Create: `web/components/navigation/AppShell.tsx`

**Цель:** Создать навигацию с меню

- [ ] **Step 1: Создать директорию**

```bash
mkdir -p web/components/navigation
```

- [ ] **Step 2: Создать AppShell.tsx**

```typescript
"use client";

import {
  AppShell as MantineAppShell,
  Burger,
  Group,
  Button,
  Text,
  Avatar,
  Menu,
  rem,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import Link from "next/link";
import { useAuth } from "@/components/auth/AuthProvider";

export default function AppShell({ children }: { children: React.ReactNode }) {
  const [opened, { toggle }] = useDisclosure();
  const { user, logout } = useAuth();

  return (
    <MantineAppShell
      header={{ height: 60 }}
      navbar={{
        width: 300,
        breakpoint: "sm",
        collapsed: { mobile: !opened },
      }}
      padding="md"
    >
      <MantineAppShell.Header>
        <Group h="100%" px="md" justify="space-between">
          <Group>
            <Burger
              opened={opened}
              onClick={toggle}
              hiddenFrom="sm"
              size="sm"
            />
            <Text component={Link} href="/" fw={700} size="lg">
              Call Booking
            </Text>
          </Group>

          {user && (
            <Group>
              <Menu>
                <Menu.Target>
                  <Group gap="xs" style={{ cursor: "pointer" }}>
                    <Avatar size="sm" color="blue">
                      {user.name.charAt(0).toUpperCase()}
                    </Avatar>
                    <Text size="sm" visibleFrom="sm">
                      {user.name}
                    </Text>
                  </Group>
                </Menu.Target>
                <Menu.Dropdown>
                  <Menu.Item onClick={logout} color="red">
                    Выйти
                  </Menu.Item>
                </Menu.Dropdown>
              </Menu>
            </Group>
          )}
        </Group>
      </MantineAppShell.Header>

      <MantineAppShell.Navbar p="md">
        <Stack gap="xs">
          <Button component={Link} href="/" variant="subtle" justify="start">
            Каталог пользователей
          </Button>
          <Button
            component={Link}
            href="/my/schedule"
            variant="subtle"
            justify="start"
          >
            Моё расписание
          </Button>
          <Button
            component={Link}
            href="/my/groups"
            variant="subtle"
            justify="start"
          >
            Мои группы
          </Button>
          <Button
            component={Link}
            href="/my/bookings"
            variant="subtle"
            justify="start"
          >
            Мои бронирования
          </Button>
        </Stack>
      </MantineAppShell.Navbar>

      <MantineAppShell.Main>{children}</MantineAppShell.Main>
    </MantineAppShell>
  );
}
```

- [ ] **Step 3: Проверить компиляцию**

Run:
```bash
cd web && npx tsc --noEmit
```

Expected: Без ошибок TypeScript

- [ ] **Step 4: Commit**

```bash
git add web/components/navigation/AppShell.tsx
git commit -m "feat: add AppShell navigation component"
```

---

### Task 17: Frontend - (app) Layout

**Files:**
- Create: `web/app/(app)/layout.tsx`

**Цель:** Создать layout для защищённых страниц

- [ ] **Step 1: Создать директорию**

```bash
mkdir -p web/app/(app)
```

- [ ] **Step 2: Создать layout.tsx**

```typescript
import ProtectedRoute from "@/components/auth/ProtectedRoute";
import AppShell from "@/components/navigation/AppShell";

export default function AppLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <ProtectedRoute>
      <AppShell>{children}</AppShell>
    </ProtectedRoute>
  );
}
```

- [ ] **Step 3: Commit**

```bash
git add web/app/(app)/layout.tsx
git commit -m "feat: add protected app layout"
```

---

### Task 18: Frontend - Root Layout with AuthProvider

**Files:**
- Modify: `web/app/layout.tsx`

**Цель:** Добавить AuthProvider в корневой layout

- [ ] **Step 1: Обновить layout.tsx**

Заменить содержимое `web/app/layout.tsx`:

```typescript
import type { Metadata } from "next";
import "@mantine/core/styles.css";
import "@mantine/dates/styles.css";
import {
  MantineProvider,
  ColorSchemeScript,
  createTheme,
} from "@mantine/core";
import { AuthProvider } from "@/components/auth/AuthProvider";

export const metadata: Metadata = {
  title: "Call Booking",
  description: "Бронирование времени для звонков",
};

const theme = createTheme({
  primaryColor: "blue",
});

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="ru">
      <head>
        <ColorSchemeScript />
      </head>
      <body>
        <MantineProvider theme={theme}>
          <AuthProvider>{children}</AuthProvider>
        </MantineProvider>
      </body>
    </html>
  );
}
```

- [ ] **Step 2: Проверить компиляцию**

Run:
```bash
cd web && npx tsc --noEmit
```

Expected: Без ошибок TypeScript

- [ ] **Step 3: Commit**

```bash
git add web/app/layout.tsx
git commit -m "feat: update root layout with AuthProvider"
```

---

### Task 19: Frontend - Users Catalog Page

**Files:**
- Create: `web/app/(app)/page.tsx`

**Цель:** Создать страницу каталога пользователей

- [ ] **Step 1: Создать page.tsx**

```typescript
"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import {
  Paper,
  Title,
  Stack,
  Card,
  Group,
  Text,
  Button,
  Loader,
  Center,
} from "@mantine/core";
import { User, getUsers } from "@/lib/api";

export default function UsersPage() {
  const router = useRouter();
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadUsers();
  }, []);

  const loadUsers = async () => {
    try {
      setLoading(true);
      const data = await getUsers();
      setUsers(data);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <Center h="50vh">
        <Loader />
      </Center>
    );
  }

  return (
    <Stack gap="md">
      <Title order={2}>Каталог пользователей</Title>

      {users.length === 0 ? (
        <Text c="dimmed">Пока нет доступных пользователей</Text>
      ) : (
        users.map((user) => (
          <Card key={user.id} withBorder>
            <Group justify="space-between">
              <div>
                <Text fw={500}>{user.name}</Text>
                <Text size="sm" c="dimmed">
                  {user.email}
                </Text>
              </div>
              <Button
                onClick={() => router.push(`/users/${user.id}`)}
                variant="light"
              >
                Записаться
              </Button>
            </Group>
          </Card>
        ))
      )}
    </Stack>
  );
}
```

- [ ] **Step 2: Добавить getUsers в api.ts**

Добавить в `web/lib/api.ts`:

```typescript
// Users API
export async function getUsers(): Promise<User[]> {
  const token = getAuthToken();
  const res = await fetch("/api/users", {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) throw new Error("Failed to fetch users");
  const data = await res.json();
  return data.users;
}
```

- [ ] **Step 3: Проверить компиляцию**

Run:
```bash
cd web && npx tsc --noEmit
```

Expected: Без ошибок TypeScript

- [ ] **Step 4: Commit**

```bash
git add web/app/(app)/page.tsx web/lib/api.ts
git commit -m "feat: add users catalog page"
```

---

### Task 20: Frontend - User Profile Page

**Files:**
- Create: `web/app/(app)/users/[id]/page.tsx`

**Цель:** Создать страницу профиля пользователя с его слотами

- [ ] **Step 1: Создать структуру директорий**

```bash
mkdir -p web/app/(app)/users/\[id\]
```

- [ ] **Step 2: Создать page.tsx**

```typescript
"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import {
  Paper,
  Title,
  Stack,
  Text,
  Button,
  Group,
  Loader,
  Center,
  Badge,
} from "@mantine/core";
import { DatePickerInput } from "@mantine/dates";
import { User, Slot, getUser, getUserSlots, createBooking } from "@/lib/api";

export default function UserProfilePage() {
  const params = useParams();
  const userId = params.id as string;

  const [user, setUser] = useState<User | null>(null);
  const [slots, setSlots] = useState<Slot[]>([]);
  const [selectedDate, setSelectedDate] = useState<Date>(new Date());
  const [loading, setLoading] = useState(true);
  const [bookingSlot, setBookingSlot] = useState<string | null>(null);

  useEffect(() => {
    loadUser();
  }, [userId]);

  useEffect(() => {
    if (userId && selectedDate) {
      loadSlots();
    }
  }, [userId, selectedDate]);

  const loadUser = async () => {
    try {
      const data = await getUser(userId);
      setUser(data);
    } catch (e) {
      console.error(e);
    }
  };

  const loadSlots = async () => {
    try {
      setLoading(true);
      const dateStr = selectedDate.toISOString().split("T")[0];
      const data = await getUserSlots(userId, dateStr);
      setSlots(data);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  const handleBooking = async (slot: Slot) => {
    try {
      setBookingSlot(slot.id);
      await createBooking({
        ownerId: userId,
        scheduleId: slot.id.split("_")[0], // Extract schedule ID from slot ID
      });
      await loadSlots();
    } catch (e) {
      console.error(e);
    } finally {
      setBookingSlot(null);
    }
  };

  if (!user) {
    return (
      <Center h="50vh">
        <Loader />
      </Center>
    );
  }

  return (
    <Stack gap="xl">
      <div>
        <Title order={2}>{user.name}</Title>
        <Text c="dimmed">{user.email}</Text>
      </div>

      <Paper p="md" withBorder>
        <Title order={4} mb="md">
          Выберите дату
        </Title>
        <DatePickerInput
          value={selectedDate}
          onChange={(date) => date && setSelectedDate(date)}
          locale="ru"
          minDate={new Date()}
        />
      </Paper>

      <Paper p="md" withBorder>
        <Title order={4} mb="md">
          Доступное время
        </Title>

        {loading ? (
          <Center>
            <Loader />
          </Center>
        ) : slots.length === 0 ? (
          <Text c="dimmed">Нет доступных слотов на выбранную дату</Text>
        ) : (
          <Group>
            {slots.map((slot) => (
              <Button
                key={slot.id}
                variant={slot.isBooked ? "light" : "filled"}
                disabled={slot.isBooked}
                loading={bookingSlot === slot.id}
                onClick={() => handleBooking(slot)}
              >
                {slot.startTime} - {slot.endTime}
                {slot.isBooked && (
                  <Badge ml="xs" size="xs">
                    Занято
                  </Badge>
                )}
              </Button>
            ))}
          </Group>
        )}
      </Paper>
    </Stack>
  );
}
```

- [ ] **Step 3: Добавить функции в api.ts**

Добавить в `web/lib/api.ts`:

```typescript
// Users API
export async function getUser(id: string): Promise<User> {
  const token = getAuthToken();
  const res = await fetch(`/api/users/${id}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) throw new Error("Failed to fetch user");
  return res.json();
}

export async function getUserSlots(id: string, date: string): Promise<Slot[]> {
  const token = getAuthToken();
  const res = await fetch(
    `/api/users/${id}/slots?date=${encodeURIComponent(date)}`,
    {
      headers: { Authorization: `Bearer ${token}` },
    }
  );
  if (!res.ok) throw new Error("Failed to fetch slots");
  const data = await res.json();
  return data.slots;
}
```

- [ ] **Step 4: Проверить компиляцию**

Run:
```bash
cd web && npx tsc --noEmit
```

Expected: Без ошибок TypeScript

- [ ] **Step 5: Commit**

```bash
git add web/app/(app)/users/ web/lib/api.ts
git commit -m "feat: add user profile page with slot booking"
```

---

### Task 21: Frontend - My Schedule Page

**Files:**
- Create: `web/app/(app)/my/schedule/page.tsx`

**Цель:** Создать страницу управления расписанием

- [ ] **Step 1: Создать структуру директорий**

```bash
mkdir -p web/app/(app)/my/schedule
```

- [ ] **Step 2: Создать page.tsx**

```typescript
"use client";

import { useEffect, useState } from "react";
import {
  Paper,
  Title,
  Stack,
  Button,
  Group,
  Text,
  Table,
  Loader,
  Center,
  Modal,
  TextInput,
  Select,
  Checkbox,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { DatePickerInput } from "@mantine/dates";
import {
  Schedule,
  CreateScheduleRequest,
  getMySchedules,
  createSchedule,
  deleteSchedule,
} from "@/lib/api";

const DAYS_OF_WEEK = [
  { value: "0", label: "Воскресенье" },
  { value: "1", label: "Понедельник" },
  { value: "2", label: "Вторник" },
  { value: "3", label: "Среда" },
  { value: "4", label: "Четверг" },
  { value: "5", label: "Пятница" },
  { value: "6", label: "Суббота" },
];

export default function MySchedulePage() {
  const [schedules, setSchedules] = useState<Schedule[]>([]);
  const [loading, setLoading] = useState(true);
  const [opened, { open, close }] = useDisclosure(false);

  // Form state
  const [type, setType] = useState<"recurring" | "one-time">("recurring");
  const [dayOfWeek, setDayOfWeek] = useState<string | null>("1");
  const [date, setDate] = useState<Date | null>(null);
  const [startTime, setStartTime] = useState("09:00");
  const [endTime, setEndTime] = useState("18:00");
  const [isBlocked, setIsBlocked] = useState(false);

  useEffect(() => {
    loadSchedules();
  }, []);

  const loadSchedules = async () => {
    try {
      setLoading(true);
      const data = await getMySchedules();
      setSchedules(data);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async () => {
    const data: CreateScheduleRequest = {
      type,
      startTime,
      endTime,
      isBlocked,
    };

    if (type === "recurring" && dayOfWeek) {
      data.dayOfWeek = parseInt(dayOfWeek);
    } else if (type === "one-time" && date) {
      data.date = date.toISOString().split("T")[0];
    }

    try {
      await createSchedule(data);
      close();
      resetForm();
      await loadSchedules();
    } catch (e) {
      console.error(e);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteSchedule(id);
      await loadSchedules();
    } catch (e) {
      console.error(e);
    }
  };

  const resetForm = () => {
    setType("recurring");
    setDayOfWeek("1");
    setDate(null);
    setStartTime("09:00");
    setEndTime("18:00");
    setIsBlocked(false);
  };

  if (loading) {
    return (
      <Center h="50vh">
        <Loader />
      </Center>
    );
  }

  return (
    <Stack gap="md">
      <Group justify="space-between">
        <Title order={2}>Моё расписание</Title>
        <Button onClick={open}>Добавить правило</Button>
      </Group>

      {schedules.length === 0 ? (
        <Text c="dimmed">У вас пока нет настроенных правил</Text>
      ) : (
        <Table>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Тип</Table.Th>
              <Table.Th>День/Дата</Table.Th>
              <Table.Th>Время</Table.Th>
              <Table.Th>Статус</Table.Th>
              <Table.Th>Действия</Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {schedules.map((schedule) => (
              <Table.Tr key={schedule.id}>
                <Table.Td>
                  {schedule.type === "recurring" ? "Повторяется" : "Разовое"}
                </Table.Td>
                <Table.Td>
                  {schedule.dayOfWeek !== undefined
                    ? DAYS_OF_WEEK[schedule.dayOfWeek]?.label
                    : schedule.date}
                </Table.Td>
                <Table.Td>
                  {schedule.startTime} - {schedule.endTime}
                </Table.Td>
                <Table.Td>
                  {schedule.isBlocked ? (
                    <Text c="red">Заблокировано</Text>
                  ) : (
                    <Text c="green">Доступно</Text>
                  )}
                </Table.Td>
                <Table.Td>
                  <Button
                    size="xs"
                    color="red"
                    variant="subtle"
                    onClick={() => handleDelete(schedule.id)}
                  >
                    Удалить
                  </Button>
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
      )}

      <Modal opened={opened} onClose={close} title="Добавить правило">
        <Stack>
          <Select
            label="Тип"
            value={type}
            onChange={(v) => setType(v as "recurring" | "one-time")}
            data={[
              { value: "recurring", label: "Повторяется" },
              { value: "one-time", label: "Разовое" },
            ]}
          />

          {type === "recurring" ? (
            <Select
              label="День недели"
              value={dayOfWeek}
              onChange={setDayOfWeek}
              data={DAYS_OF_WEEK}
            />
          ) : (
            <DatePickerInput
              label="Дата"
              value={date}
              onChange={setDate}
              locale="ru"
              minDate={new Date()}
            />
          )}

          <Group grow>
            <TextInput
              label="Начало"
              type="time"
              value={startTime}
              onChange={(e) => setStartTime(e.target.value)}
            />
            <TextInput
              label="Конец"
              type="time"
              value={endTime}
              onChange={(e) => setEndTime(e.target.value)}
            />
          </Group>

          <Checkbox
            label="Заблокировать (недоступно для бронирования)"
            checked={isBlocked}
            onChange={(e) => setIsBlocked(e.currentTarget.checked)}
          />

          <Group justify="flex-end" mt="md">
            <Button variant="subtle" onClick={close}>
              Отмена
            </Button>
            <Button onClick={handleSubmit}>Сохранить</Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  );
}
```

- [ ] **Step 3: Добавить функции в api.ts**

Добавить в `web/lib/api.ts`:

```typescript
// Schedules API
export interface CreateScheduleRequest {
  type: "recurring" | "one-time";
  dayOfWeek?: number;
  date?: string;
  startTime: string;
  endTime: string;
  isBlocked: boolean;
}

export async function getMySchedules(): Promise<Schedule[]> {
  const token = getAuthToken();
  const res = await fetch("/api/my/schedules", {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) throw new Error("Failed to fetch schedules");
  const data = await res.json();
  return data.schedules;
}

export async function createSchedule(
  data: CreateScheduleRequest
): Promise<Schedule> {
  const token = getAuthToken();
  const res = await fetch("/api/my/schedules", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(data),
  });
  if (!res.ok) throw new Error("Failed to create schedule");
  return res.json();
}

export async function deleteSchedule(id: string): Promise<void> {
  const token = getAuthToken();
  const res = await fetch(`/api/my/schedules/${id}`, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) throw new Error("Failed to delete schedule");
}
```

- [ ] **Step 4: Проверить компиляцию**

Run:
```bash
cd web && npx tsc --noEmit
```

Expected: Без ошибок TypeScript

- [ ] **Step 5: Commit**

```bash
git add web/app/(app)/my/schedule/ web/lib/api.ts
git commit -m "feat: add my schedule page"
```

---

### Task 22: Frontend - My Groups Page

**Files:**
- Create: `web/app/(app)/my/groups/page.tsx`

**Цель:** Создать страницу управления группами видимости

- [ ] **Step 1: Создать структуру директорий**

```bash
mkdir -p web/app/(app)/my/groups
```

- [ ] **Step 2: Создать page.tsx**

```typescript
"use client";

import { useEffect, useState } from "react";
import {
  Paper,
  Title,
  Stack,
  Button,
  Group,
  Text,
  Accordion,
  Badge,
  Loader,
  Center,
  Modal,
  TextInput,
  Select,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import {
  VisibilityGroup,
  GroupMember,
  CreateGroupRequest,
  AddMemberRequest,
  getMyGroups,
  createGroup,
  deleteGroup,
  getGroupMembers,
  addGroupMember,
  removeGroupMember,
} from "@/lib/api";

const VISIBILITY_LEVELS = [
  { value: "family", label: "Семья" },
  { value: "work", label: "Работа" },
  { value: "friends", label: "Друзья" },
  { value: "public", label: "Все" },
];

export default function MyGroupsPage() {
  const [groups, setGroups] = useState<VisibilityGroup[]>([]);
  const [membersMap, setMembersMap] = useState<Record<string, GroupMember[]>>({});
  const [loading, setLoading] = useState(true);
  const [opened, { open, close }] = useDisclosure(false);
  const [memberModal, setMemberModal] = useState<{
    opened: boolean;
    groupId: string | null;
  }>({ opened: false, groupId: null });

  // Form state
  const [name, setName] = useState("");
  const [visibilityLevel, setVisibilityLevel] = useState<string>("work");
  const [memberEmail, setMemberEmail] = useState("");

  useEffect(() => {
    loadGroups();
  }, []);

  const loadGroups = async () => {
    try {
      setLoading(true);
      const data = await getMyGroups();
      setGroups(data);

      // Load members for each group
      const members: Record<string, GroupMember[]> = {};
      for (const group of data) {
        members[group.id] = await getGroupMembers(group.id);
      }
      setMembersMap(members);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateGroup = async () => {
    const data: CreateGroupRequest = {
      name,
      visibilityLevel,
    };

    try {
      await createGroup(data);
      close();
      setName("");
      setVisibilityLevel("work");
      await loadGroups();
    } catch (e) {
      console.error(e);
    }
  };

  const handleDeleteGroup = async (id: string) => {
    try {
      await deleteGroup(id);
      await loadGroups();
    } catch (e) {
      console.error(e);
    }
  };

  const handleAddMember = async () => {
    if (!memberModal.groupId) return;

    const data: AddMemberRequest = { email: memberEmail };

    try {
      await addGroupMember(memberModal.groupId, data);
      setMemberModal({ opened: false, groupId: null });
      setMemberEmail("");
      await loadGroups();
    } catch (e) {
      console.error(e);
    }
  };

  const handleRemoveMember = async (groupId: string, memberId: string) => {
    try {
      await removeGroupMember(groupId, memberId);
      await loadGroups();
    } catch (e) {
      console.error(e);
    }
  };

  if (loading) {
    return (
      <Center h="50vh">
        <Loader />
      </Center>
    );
  }

  return (
    <Stack gap="md">
      <Group justify="space-between">
        <Title order={2}>Мои группы видимости</Title>
        <Button onClick={open}>Создать группу</Button>
      </Group>

      {groups.length === 0 ? (
        <Text c="dimmed">У вас пока нет групп</Text>
      ) : (
        <Accordion>
          {groups.map((group) => (
            <Accordion.Item key={group.id} value={group.id}>
              <Accordion.Control>
                <Group>
                  <Text fw={500}>{group.name}</Text>
                  <Badge>
                    {VISIBILITY_LEVELS.find((l) => l.value === group.visibilityLevel)?.label}
                  </Badge>
                </Group>
              </Accordion.Control>
              <Accordion.Panel>
                <Stack>
                  <Group justify="space-between">
                    <Text fw={500}>Участники:</Text>
                    <Button
                      size="xs"
                      onClick={() => setMemberModal({ opened: true, groupId: group.id })}
                    >
                      Добавить участника
                    </Button>
                  </Group>

                  {membersMap[group.id]?.length === 0 ? (
                    <Text c="dimmed" size="sm">
                      Нет участников
                    </Text>
                  ) : (
                    membersMap[group.id]?.map((member) => (
                      <Group key={member.id} justify="space-between">
                        <Text size="sm">
                          {member.member.name} ({member.member.email})
                        </Text>
                        <Button
                          size="xs"
                          color="red"
                          variant="subtle"
                          onClick={() => handleRemoveMember(group.id, member.member.id)}
                        >
                          Удалить
                        </Button>
                      </Group>
                    ))
                  )}

                  <Button
                    color="red"
                    variant="subtle"
                    mt="md"
                    onClick={() => handleDeleteGroup(group.id)}
                  >
                    Удалить группу
                  </Button>
                </Stack>
              </Accordion.Panel>
            </Accordion.Item>
          ))}
        </Accordion>
      )}

      <Modal opened={opened} onClose={close} title="Создать группу">
        <Stack>
          <TextInput
            label="Название"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
          />

          <Select
            label="Уровень видимости"
            value={visibilityLevel}
            onChange={(v) => v && setVisibilityLevel(v)}
            data={VISIBILITY_LEVELS}
          />

          <Group justify="flex-end" mt="md">
            <Button variant="subtle" onClick={close}>
              Отмена
            </Button>
            <Button onClick={handleCreateGroup}>Создать</Button>
          </Group>
        </Stack>
      </Modal>

      <Modal
        opened={memberModal.opened}
        onClose={() => setMemberModal({ opened: false, groupId: null })}
        title="Добавить участника"
      >
        <Stack>
          <TextInput
            label="Email участника"
            type="email"
            value={memberEmail}
            onChange={(e) => setMemberEmail(e.target.value)}
            placeholder="user@example.com"
            required
          />

          <Group justify="flex-end" mt="md">
            <Button
              variant="subtle"
              onClick={() => setMemberModal({ opened: false, groupId: null })}
            >
              Отмена
            </Button>
            <Button onClick={handleAddMember}>Добавить</Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  );
}
```

- [ ] **Step 3: Добавить функции в api.ts**

Добавить в `web/lib/api.ts`:

```typescript
// Groups API
export interface CreateGroupRequest {
  name: string;
  visibilityLevel: string;
}

export interface AddMemberRequest {
  email?: string;
  userId?: string;
}

export interface VisibilityGroup {
  id: string;
  ownerId: string;
  name: string;
  visibilityLevel: string;
  createdAt?: string;
}

export interface GroupMember {
  id: string;
  groupId: string;
  member: User;
  addedBy: string;
  addedAt: string;
}

export async function getMyGroups(): Promise<VisibilityGroup[]> {
  const token = getAuthToken();
  const res = await fetch("/api/my/groups", {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) throw new Error("Failed to fetch groups");
  const data = await res.json();
  return data.groups;
}

export async function createGroup(data: CreateGroupRequest): Promise<VisibilityGroup> {
  const token = getAuthToken();
  const res = await fetch("/api/my/groups", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(data),
  });
  if (!res.ok) throw new Error("Failed to create group");
  return res.json();
}

export async function deleteGroup(id: string): Promise<void> {
  const token = getAuthToken();
  const res = await fetch(`/api/my/groups/${id}`, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) throw new Error("Failed to delete group");
}

export async function getGroupMembers(groupId: string): Promise<GroupMember[]> {
  const token = getAuthToken();
  const res = await fetch(`/api/my/groups/${groupId}/members`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) throw new Error("Failed to fetch members");
  const data = await res.json();
  return data.members;
}

export async function addGroupMember(
  groupId: string,
  data: AddMemberRequest
): Promise<void> {
  const token = getAuthToken();
  const res = await fetch(`/api/my/groups/${groupId}/members`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(data),
  });
  if (!res.ok) throw new Error("Failed to add member");
}

export async function removeGroupMember(
  groupId: string,
  memberId: string
): Promise<void> {
  const token = getAuthToken();
  const res = await fetch(`/api/my/groups/${groupId}/members/${memberId}`, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) throw new Error("Failed to remove member");
}
```

- [ ] **Step 4: Проверить компиляцию**

Run:
```bash
cd web && npx tsc --noEmit
```

Expected: Без ошибок TypeScript

- [ ] **Step 5: Commit**

```bash
git add web/app/(app)/my/groups/ web/lib/api.ts
git commit -m "feat: add my groups page"
```

---

### Task 23: Frontend - My Bookings Page

**Files:**
- Create: `web/app/(app)/my/bookings/page.tsx`

**Цель:** Создать страницу просмотра бронирований

- [ ] **Step 1: Создать структуру директорий**

```bash
mkdir -p web/app/(app)/my/bookings
```

- [ ] **Step 2: Создать page.tsx**

```typescript
"use client";

import { useEffect, useState } from "react";
import {
  Paper,
  Title,
  Stack,
  Card,
  Group,
  Text,
  Button,
  Badge,
  Loader,
  Center,
} from "@mantine/core";
import { useAuth } from "@/components/auth/AuthProvider";
import { Booking, getMyBookings, cancelBooking } from "@/lib/api";

export default function MyBookingsPage() {
  const { user } = useAuth();
  const [bookings, setBookings] = useState<Booking[]>([]);
  const [loading, setLoading] = useState(true);
  const [cancellingId, setCancellingId] = useState<string | null>(null);

  useEffect(() => {
    loadBookings();
  }, []);

  const loadBookings = async () => {
    try {
      setLoading(true);
      const data = await getMyBookings();
      setBookings(data);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  const handleCancel = async (id: string) => {
    try {
      setCancellingId(id);
      await cancelBooking(id);
      await loadBookings();
    } catch (e) {
      console.error(e);
    } finally {
      setCancellingId(null);
    }
  };

  const isBooker = (booking: Booking) => booking.booker.id === user?.id;

  if (loading) {
    return (
      <Center h="50vh">
        <Loader />
      </Center>
    );
  }

  return (
    <Stack gap="md">
      <Title order={2}>Мои бронирования</Title>

      {bookings.length === 0 ? (
        <Text c="dimmed">У вас пока нет бронирований</Text>
      ) : (
        bookings.map((booking) => (
          <Card key={booking.id} withBorder>
            <Group justify="space-between" align="flex-start">
              <Stack gap="xs">
                <Group>
                  <Text fw={500}>
                    {isBooker(booking)
                      ? `Запись к ${booking.owner.name}`
                      : `Запись от ${booking.booker.name}`}
                  </Text>
                  {booking.status === "active" ? (
                    <Badge color="green">Активно</Badge>
                  ) : (
                    <Badge color="red">Отменено</Badge>
                  )}
                </Group>

                <Text size="sm" c="dimmed">
                  Дата: {booking.date}
                </Text>
                <Text size="sm" c="dimmed">
                  Время: {booking.startTime} - {booking.endTime}
                </Text>

                {booking.cancelledAt && (
                  <Text size="sm" c="red">
                    Отменено: {new Date(booking.cancelledAt).toLocaleDateString("ru-RU")}
                  </Text>
                )}
              </Stack>

              {booking.status === "active" && (
                <Button
                  color="red"
                  variant="subtle"
                  loading={cancellingId === booking.id}
                  onClick={() => handleCancel(booking.id)}
                >
                  Отменить
                </Button>
              )}
            </Group>
          </Card>
        ))
      )}
    </Stack>
  );
}
```

- [ ] **Step 3: Добавить функции в api.ts**

Добавить в `web/lib/api.ts`:

```typescript
// Bookings API
export interface Booking {
  id: string;
  scheduleId: string;
  booker: User;
  owner: User;
  date: string;
  startTime: string;
  endTime: string;
  status: "active" | "cancelled";
  createdAt?: string;
  cancelledAt?: string;
}

export async function getMyBookings(): Promise<Booking[]> {
  const token = getAuthToken();
  const res = await fetch("/api/my/bookings", {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) throw new Error("Failed to fetch bookings");
  const data = await res.json();
  return data.bookings;
}

export async function createBooking(data: {
  ownerId: string;
  scheduleId: string;
}): Promise<Booking> {
  const token = getAuthToken();
  const res = await fetch("/api/my/bookings", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(data),
  });
  if (!res.ok) throw new Error("Failed to create booking");
  return res.json();
}

export async function cancelBooking(id: string): Promise<void> {
  const token = getAuthToken();
  const res = await fetch(`/api/my/bookings/${id}`, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) throw new Error("Failed to cancel booking");
}
```

- [ ] **Step 4: Проверить компиляцию**

Run:
```bash
cd web && npx tsc --noEmit
```

Expected: Без ошибок TypeScript

- [ ] **Step 5: Commit**

```bash
git add web/app/(app)/my/bookings/ web/lib/api.ts
git commit -m "feat: add my bookings page"
```

---

### Task 24: Final Integration & Testing

**Files:**
- All files

**Цель:** Провести финальную интеграцию и тестирование

- [ ] **Step 1: Проверить компиляцию всего проекта**

Run:
```bash
# Backend
go build ./cmd/server

# Frontend
cd web && npm run build
```

Expected: Компиляция без ошибок

- [ ] **Step 2: Запустить тесты**

Run:
```bash
go test ./... -v 2>&1 | head -50
```

Expected: Тесты компилируются (могут быть failures из-за изменений)

- [ ] **Step 3: Обновить go.mod если нужно**

Run:
```bash
go mod tidy
```

- [ ] **Step 4: Финальный коммит**

```bash
git add .
git commit -m "chore: final integration and testing"
```

---

## Самопроверка плана

### 1. Покрытие спецификации

| Секция спецификации | Задачи |
|-------------------|--------|
| Database Schema | Task 1 |
| Models | Task 2 |
| JWT/Password | Task 3-4 |
| Auth API | Task 5 |
| Users API | Task 6 |
| Schedules API | Task 7 |
| Groups API | Task 8 |
| Bookings API | Task 9 |
| Router | Task 10 |
| Frontend Auth | Task 11-14 |
| Frontend Components | Task 15-17 |
| Frontend Pages | Task 19-23 |
| Integration | Task 24 |

### 2. Placeholder скан

- ✅ Нет TBD/TODO
- ✅ Все SQL запросы конкретные
- ✅ Все API endpoints с примерами
- ✅ Все компоненты с полным кодом

### 3. Консистентность типов

- ✅ User модель совпадает в Go и TS
- ✅ Schedule типы консистентны
- ✅ Booking структура одинаковая

---

## Следующий шаг

**План записан в:** `docs/superpowers/plans/2025-01-08-authentication-implementation-plan.md`

**Два варианта выполнения:**

1. **Subagent-Driven (рекомендую)** — Я запускаю свежего subagent на каждую задачу, проверяю между задачами, быстрая итерация

2. **Inline Execution** — Выполняю задачи в этой сессии, batch execution с чекпоинтами

**Какой вариант выбираете?**
