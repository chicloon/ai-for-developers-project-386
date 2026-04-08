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
		       CASE WHEN s.type = 'recurring' THEN '' ELSE TO_CHAR(s.date, 'YYYY-MM-DD') END, s.start_time, s.end_time,
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

	if req.OwnerID == "" || req.ScheduleID == "" || req.SlotDate == "" {
		jsonError(w, http.StatusBadRequest, "owner_id, schedule_id and slot_date are required")
		return
	}

	// Check if user can see the owner and can book this specific schedule
	canBook, err := h.canBookSchedule(r.Context(), userID, req.OwnerID, req.ScheduleID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !canBook {
		jsonError(w, http.StatusForbidden, "you don't have access to book this schedule")
		return
	}

	// Check if slot is already booked for this specific date and time
	var exists bool
	err = h.pool.QueryRow(r.Context(),
		"SELECT EXISTS(SELECT 1 FROM bookings WHERE schedule_id = $1 AND slot_date = $2 AND slot_start_time = $3 AND status = 'active')",
		req.ScheduleID, req.SlotDate, req.SlotStartTime).Scan(&exists)
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
		`INSERT INTO bookings (schedule_id, booker_id, owner_id, slot_start_time, slot_date)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, schedule_id, booker_id, owner_id, status, TO_CHAR(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`,
		req.ScheduleID, userID, req.OwnerID, req.SlotStartTime, req.SlotDate).
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

// canBookSchedule checks if current user can book a specific schedule
// Logic:
// - If schedule has no group associations: visible if owner is public OR user is self
// - If schedule has group associations: visible if user is member of at least one group OR user is self
func (h *bookingsHandler) canBookSchedule(ctx context.Context, currentUserID, ownerID, scheduleID string) (bool, error) {
	if currentUserID == ownerID {
		return true, nil
	}

	// Get schedule's group associations
	rows, err := h.pool.Query(ctx,
		"SELECT group_id FROM schedule_visibility_groups WHERE schedule_id = $1",
		scheduleID)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	var groupIDs []string
	for rows.Next() {
		var gid string
		if err := rows.Scan(&gid); err != nil {
			continue
		}
		groupIDs = append(groupIDs, gid)
	}

	// If schedule has no groups - check if owner is public
	if len(groupIDs) == 0 {
		var isPublic bool
		err := h.pool.QueryRow(ctx,
			"SELECT is_public FROM users WHERE id = $1",
			ownerID).Scan(&isPublic)
		if err != nil {
			return false, err
		}
		return isPublic, nil
	}

	// Schedule has groups - check if user is member of any group
	// Note: groups belong to the owner, so we need to check if current user is in any of the owner's groups
	query := `
		SELECT EXISTS (
			SELECT 1 FROM group_members gm
			JOIN visibility_groups vg ON vg.id = gm.group_id
			WHERE vg.owner_id = $1
			  AND gm.member_id = $2
			  AND gm.group_id = ANY($3)
		)
	`
	var isMember bool
	err = h.pool.QueryRow(ctx, query, ownerID, currentUserID, groupIDs).Scan(&isMember)
	return isMember, err
}
