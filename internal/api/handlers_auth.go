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
		jsonError(w, http.StatusInternalServerError, "database error")
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
		"INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3) RETURNING id, email, name, TO_CHAR(created_at, 'YYYY-MM-DD\"T\"HH24:MI:SS\"Z\"')",
		req.Email, hash, req.Name).
		Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "database error")
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
		"SELECT id, email, name, password_hash, TO_CHAR(created_at, 'YYYY-MM-DD\"T\"HH24:MI:SS\"Z\"') FROM users WHERE email = $1",
		req.Email).
		Scan(&user.ID, &user.Email, &user.Name, &passwordHash, &user.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Run dummy bcrypt to prevent timing attacks
			auth.CheckPassword("dummy", "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")
			jsonError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		jsonError(w, http.StatusInternalServerError, "database error")
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
		"SELECT id, email, name, TO_CHAR(created_at, 'YYYY-MM-DD\"T\"HH24:MI:SS\"Z\"') FROM users WHERE id = $1",
		userID).
		Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt)
	if err != nil {
		jsonError(w, http.StatusNotFound, "user not found")
		return
	}

	jsonResponse(w, http.StatusOK, user)
}
