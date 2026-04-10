package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"call-booking/internal/auth"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
)

func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func NewRouter(pool *pgxpool.Pool) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware - ALL middleware must be defined before routes
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

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, http.StatusOK, map[string]string{"status": "ok", "service": "api"})
	})

	// Root: this deploy is API-only (see Dockerfile). Browsers opening "/" get a hint instead of chi's plain 404.
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		if u := os.Getenv("PUBLIC_WEB_URL"); u != "" {
			http.Redirect(w, r, u, http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `<!DOCTYPE html><html lang="ru"><head><meta charset="utf-8"><title>Call Booking API</title></head><body>
<h1>Call Booking — API</h1>
<p>На этом URL работает только бэкенд (маршруты <code>/api/*</code> и <a href="/health">/health</a>).</p>
<p>Интерфейс (Next.js) нужно задеплоить отдельным Web Service в Render из <code>Dockerfile.web</code>, с переменной <code>API_PROXY_URL</code> = базовый URL этого API (например <code>https://%s</code> без слэша в конце).</p>
</body></html>`, r.Host)
	})

	// Public routes (no JWT required)
	r.Mount("/api/auth", authRouter(pool))

	// Protected routes (JWT required)
	r.Route("/api", func(r chi.Router) {
		r.Use(auth.Middleware)
		r.Mount("/users", usersRouter(pool))
		r.Mount("/my/schedules", schedulesRouter(pool))
		r.Mount("/my/groups", groupsRouter(pool))
		r.Mount("/my/bookings", bookingsRouter(pool))

		// Available users for group member addition (all users except current)
		r.Get("/my/available-users", func(w http.ResponseWriter, r *http.Request) {
			h := &usersHandler{pool: pool}
			h.availableUsers(w, r)
		})
	})

	// Route for current user profile updates
	r.With(auth.Middleware).Put("/api/users/me", func(w http.ResponseWriter, r *http.Request) {
		h := &usersHandler{pool: pool}
		h.updateMe(w, r)
	})

	return r
}
