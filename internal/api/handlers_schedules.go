package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"call-booking/internal/auth"
	"call-booking/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
		`SELECT s.id, s.user_id, s.type, s.day_of_week, s.date, s.start_time, s.end_time, s.is_blocked,
			TO_CHAR(s.created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			COALESCE(
				ARRAY_AGG(svg.group_id) FILTER (WHERE svg.group_id IS NOT NULL),
				ARRAY[]::UUID[]
			) as group_ids
		 FROM schedules s
		 LEFT JOIN schedule_visibility_groups svg ON svg.schedule_id = s.id
		 WHERE s.user_id = $1
		 GROUP BY s.id, s.user_id, s.type, s.day_of_week, s.date, s.start_time, s.end_time, s.is_blocked, s.created_at
		 ORDER BY s.created_at DESC`,
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
		var groupIDs []string
		if err := rows.Scan(&s.ID, &s.UserID, &s.Type, &dayOfWeek, &date, &s.StartTime, &s.EndTime, &s.IsBlocked, &s.CreatedAt, &groupIDs); err != nil {
			jsonError(w, http.StatusInternalServerError, "database error: "+err.Error())
			return
		}
		s.DayOfWeek = dayOfWeek
		s.Date = date
		// Filter out empty UUID entries
		s.GroupIDs = filterEmptyUUIDs(groupIDs)
		schedules = append(schedules, s)
	}

	if schedules == nil {
		schedules = []models.Schedule{}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"schedules": schedules})
}

// filterEmptyUUIDs removes empty/zero UUIDs from the array
func filterEmptyUUIDs(ids []string) []string {
	var result []string
	emptyUUID := "00000000-0000-0000-0000-000000000000"
	for _, id := range ids {
		if id != "" && id != emptyUUID {
			result = append(result, id)
		}
	}
	return result
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

	// Start transaction
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "database error: "+err.Error())
		return
	}
	defer tx.Rollback(r.Context())

	var s models.Schedule
	var dayOfWeek *int32
	var date *string

	err = tx.QueryRow(r.Context(),
		`INSERT INTO schedules (user_id, type, day_of_week, date, start_time, end_time, is_blocked)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, user_id, type, day_of_week, date, start_time, end_time, is_blocked, TO_CHAR(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`,
		userID, req.Type, req.DayOfWeek, req.Date, startTime, endTime, req.IsBlocked).
		Scan(&s.ID, &s.UserID, &s.Type, &dayOfWeek, &date, &s.StartTime, &s.EndTime, &s.IsBlocked, &s.CreatedAt)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "database error: "+err.Error())
		return
	}
	s.DayOfWeek = dayOfWeek
	s.Date = date

	// Insert group associations if provided
	if len(req.GroupIDs) > 0 {
		for _, groupID := range req.GroupIDs {
			// Verify the group belongs to the user
			var ownerID string
			err := tx.QueryRow(r.Context(),
				"SELECT owner_id FROM visibility_groups WHERE id = $1",
				groupID).Scan(&ownerID)
			if err != nil {
				continue // Skip invalid groups
			}
			if ownerID != userID {
				continue // Skip groups not owned by user
			}

			_, err = tx.Exec(r.Context(),
				"INSERT INTO schedule_visibility_groups (schedule_id, group_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
				s.ID, groupID)
			if err != nil {
				// Log error but continue
				continue
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(r.Context()); err != nil {
		jsonError(w, http.StatusInternalServerError, "database error: "+err.Error())
		return
	}

	// Fetch the group IDs we just inserted
	rows, err := h.pool.Query(r.Context(),
		"SELECT group_id FROM schedule_visibility_groups WHERE schedule_id = $1",
		s.ID)
	if err == nil {
		defer rows.Close()
		var groupIDs []string
		for rows.Next() {
			var gid string
			if err := rows.Scan(&gid); err == nil {
				groupIDs = append(groupIDs, gid)
			}
		}
		s.GroupIDs = groupIDs
	}

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

	// Start transaction
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "database error: "+err.Error())
		return
	}
	defer tx.Rollback(r.Context())

	var s models.Schedule
	var dayOfWeek *int32
	var date *string

	err = tx.QueryRow(r.Context(),
		`UPDATE schedules SET type=$1, day_of_week=$2, date=$3, start_time=$4, end_time=$5, is_blocked=$6
		 WHERE id=$7 AND user_id=$8
		 RETURNING id, user_id, type, day_of_week, date, start_time, end_time, is_blocked, TO_CHAR(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`,
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

	// Delete existing group associations
	_, err = tx.Exec(r.Context(),
		"DELETE FROM schedule_visibility_groups WHERE schedule_id = $1",
		scheduleID)
	if err != nil {
		// Continue even if delete fails
	}

	// Insert new group associations if provided
	if len(req.GroupIDs) > 0 {
		for _, groupID := range req.GroupIDs {
			// Verify the group belongs to the user
			var ownerID string
			err := tx.QueryRow(r.Context(),
				"SELECT owner_id FROM visibility_groups WHERE id = $1",
				groupID).Scan(&ownerID)
			if err != nil {
				continue
			}
			if ownerID != userID {
				continue
			}

			_, err = tx.Exec(r.Context(),
				"INSERT INTO schedule_visibility_groups (schedule_id, group_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
				scheduleID, groupID)
			if err != nil {
				continue
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(r.Context()); err != nil {
		jsonError(w, http.StatusInternalServerError, "database error: "+err.Error())
		return
	}

	// Fetch the current group IDs
	rows, err := h.pool.Query(r.Context(),
		"SELECT group_id FROM schedule_visibility_groups WHERE schedule_id = $1",
		s.ID)
	if err == nil {
		defer rows.Close()
		var groupIDs []string
		for rows.Next() {
			var gid string
			if err := rows.Scan(&gid); err == nil {
				groupIDs = append(groupIDs, gid)
			}
		}
		s.GroupIDs = groupIDs
	}

	jsonResponse(w, http.StatusOK, s)
}

func (h *schedulesHandler) delete(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	scheduleID := chi.URLParam(r, "id")

	// Group associations will be deleted by CASCADE
	result, err := h.pool.Exec(r.Context(),
		"DELETE FROM schedules WHERE id = $1 AND user_id = $2",
		scheduleID, userID)
	if err != nil {
		// Check if it's a foreign key constraint violation
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23503" {
			// Still delete the schedule - the associations should cascade
			_, _ = h.pool.Exec(r.Context(),
				"DELETE FROM schedule_visibility_groups WHERE schedule_id = $1",
				scheduleID)
			result, err = h.pool.Exec(r.Context(),
				"DELETE FROM schedules WHERE id = $1 AND user_id = $2",
				scheduleID, userID)
		}
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "database error")
			return
		}
	}

	if result.RowsAffected() == 0 {
		jsonError(w, http.StatusNotFound, "schedule not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// isValidUUID checks if a string is a valid UUID format
func isValidUUID(u string) bool {
	if len(u) != 36 {
		return false
	}
	parts := strings.Split(u, "-")
	if len(parts) != 5 {
		return false
	}
	return true
}
