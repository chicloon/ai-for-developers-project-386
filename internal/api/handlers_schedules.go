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
		jsonError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	var schedules []models.Schedule
	for rows.Next() {
		var s models.Schedule
		var dayOfWeek *int32
		var date *string
		if err := rows.Scan(&s.ID, &s.UserID, &s.Type, &dayOfWeek, &date, &s.StartTime, &s.EndTime, &s.IsBlocked, &s.CreatedAt); err != nil {
			jsonError(w, http.StatusInternalServerError, "database error")
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

	// Format time to PostgreSQL TIME format (HH:MM:SS)
	startTime := req.StartTime
	if len(startTime) == 5 {
		startTime = startTime + ":00"
	}
	endTime := req.EndTime
	if len(endTime) == 5 {
		endTime = endTime + ":00"
	}

	var s models.Schedule
	var dayOfWeek *int32
	var date *string

	err := h.pool.QueryRow(r.Context(),
		"INSERT INTO schedules (user_id, type, day_of_week, date, start_time, end_time, is_blocked) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, user_id, type, day_of_week, date, start_time, end_time, is_blocked, created_at",
		userID, req.Type, req.DayOfWeek, req.Date, startTime, endTime, req.IsBlocked).
		Scan(&s.ID, &s.UserID, &s.Type, &dayOfWeek, &date, &s.StartTime, &s.EndTime, &s.IsBlocked, &s.CreatedAt)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "database error: "+err.Error())
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

	// Format time to PostgreSQL TIME format (HH:MM:SS)
	startTime := req.StartTime
	if len(startTime) == 5 {
		startTime = startTime + ":00"
	}
	endTime := req.EndTime
	if len(endTime) == 5 {
		endTime = endTime + ":00"
	}

	var s models.Schedule
	var dayOfWeek *int32
	var date *string

	err := h.pool.QueryRow(r.Context(),
		"UPDATE schedules SET type=$1, day_of_week=$2, date=$3, start_time=$4, end_time=$5, is_blocked=$6 WHERE id=$7 AND user_id=$8 RETURNING id, user_id, type, day_of_week, date, start_time, end_time, is_blocked, created_at",
		req.Type, req.DayOfWeek, req.Date, startTime, endTime, req.IsBlocked, scheduleID, userID).
		Scan(&s.ID, &s.UserID, &s.Type, &dayOfWeek, &date, &s.StartTime, &s.EndTime, &s.IsBlocked, &s.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			jsonError(w, http.StatusNotFound, "schedule not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "database error: "+err.Error())
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
		jsonError(w, http.StatusInternalServerError, "database error")
		return
	}

	if result.RowsAffected() == 0 {
		jsonError(w, http.StatusNotFound, "schedule not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
