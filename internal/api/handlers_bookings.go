package api

import (
	"context"
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
		       b.status, TO_CHAR(b.created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), TO_CHAR(b.cancelled_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
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
		 RETURNING id, schedule_id, booker_id, owner_id, status, TO_CHAR(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`,
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
