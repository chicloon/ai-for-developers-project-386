package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
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
		jsonError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Email, &user.Name); err != nil {
			jsonError(w, http.StatusInternalServerError, "database error")
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
		jsonError(w, http.StatusInternalServerError, "database error")
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
		id        string
		startTime string
		endTime   string
		isBlocked bool
	}

	var schedules []schedule
	for rows.Next() {
		var s schedule
		if err := rows.Scan(&s.id, &s.startTime, &s.endTime, &s.isBlocked); err != nil {
			return nil, err
		}
		schedules = append(schedules, s)
	}

	// Get booked slots - use slot_date to properly track bookings on recurring schedules
	bookedRows, err := h.pool.Query(ctx, `
		SELECT slot_start_time
		FROM bookings
		WHERE owner_id = $1
		  AND slot_date = $2
		  AND status = 'active'
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

	// #region agent log
	// Debug logging for slot checking
	go func(dateStr string, times map[string]bool) {
		f, _ := os.OpenFile("/home/user/git/ai-for-developers-project-386/.cursor/debug-5ccf59.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if f != nil {
			defer f.Close()
			timesList := make([]string, 0, len(times))
			for t := range times {
				timesList = append(timesList, t)
			}
			logLine := fmt.Sprintf(`{"id":"log_%d","timestamp":%d,"location":"handlers_users.go:210","message":"Booked times for date","data":{"date":"%s","bookedTimes":%q},"hypothesisId":"H2"}`+"\n",
				time.Now().UnixNano(), time.Now().UnixMilli(), dateStr, timesList)
			f.WriteString(logLine)
		}
	}(date, bookedTimes)
	// #endregion

	// Generate 30-min slots
	var slots []models.Slot
	slotDuration := 30 * time.Minute

	for _, s := range schedules {
		if s.isBlocked {
			continue
		}

		start, err := time.Parse("15:04:05", s.startTime)
		if err != nil {
			return nil, fmt.Errorf("invalid start time format: %w", err)
		}
		end, err := time.Parse("15:04:05", s.endTime)
		if err != nil {
			return nil, fmt.Errorf("invalid end time format: %w", err)
		}

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
