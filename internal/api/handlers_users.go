package api

import (
	"context"
	"encoding/json"
	"fmt"
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
		SELECT DISTINCT u.id, u.email, u.name, u.is_public FROM users u
		LEFT JOIN visibility_groups vg ON vg.owner_id = u.id
		LEFT JOIN group_members gm ON gm.group_id = vg.id AND gm.member_id = $1
		WHERE u.id != $1
		  AND (
			u.is_public = true
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
		if err := rows.Scan(&user.ID, &user.Email, &user.Name, &user.IsPublic); err != nil {
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

	// Get schedules for the date with visibility filtering
	slots, err := h.getSlotsForDate(r.Context(), currentUserID, userID, date)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "database error")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"slots": slots})
}

// updateMe updates the current user's profile
func (h *usersHandler) updateMe(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Build dynamic update query
	var setFields []string
	var args []interface{}
	argIdx := 1

	if req.Name != nil {
		setFields = append(setFields, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}

	if req.IsPublic != nil {
		setFields = append(setFields, fmt.Sprintf("is_public = $%d", argIdx))
		args = append(args, *req.IsPublic)
		argIdx++
	}

	if len(setFields) == 0 {
		jsonError(w, http.StatusBadRequest, "no fields to update")
		return
	}

	// Add user ID as the last argument
	args = append(args, userID)

	query := fmt.Sprintf(
		"UPDATE users SET %s, updated_at = NOW() WHERE id = $%d RETURNING id, email, name, is_public, TO_CHAR(created_at, 'YYYY-MM-DD\"T\"HH24:MI:SS\"Z\"')",
		joinStrings(setFields, ", "), argIdx)

	var user models.User
	err := h.pool.QueryRow(r.Context(), query, args...).
		Scan(&user.ID, &user.Email, &user.Name, &user.IsPublic, &user.CreatedAt)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "database error")
		return
	}

	jsonResponse(w, http.StatusOK, user)
}

// joinStrings joins a slice of strings with a separator
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// canSeeUser checks if current user can see target user
func (h *usersHandler) canSeeUser(ctx context.Context, currentUserID, targetUserID string) bool {
	if currentUserID == targetUserID {
		return true
	}

	var visible bool
	query := `
		SELECT EXISTS (
			SELECT 1 FROM users u
			LEFT JOIN visibility_groups vg ON vg.owner_id = u.id
			LEFT JOIN group_members gm ON gm.group_id = vg.id AND gm.member_id = $1
			WHERE u.id = $2
			  AND (u.is_public = true OR gm.member_id IS NOT NULL)
		)
	`
	err := h.pool.QueryRow(ctx, query, currentUserID, targetUserID).Scan(&visible)
	if err != nil {
		return false
	}
	return visible
}

// getSlotsForDate generates 30-min slots from schedules with visibility filtering
func (h *usersHandler) getSlotsForDate(ctx context.Context, currentUserID, ownerID, date string) ([]models.Slot, error) {
	// Check if owner is public
	var isOwnerPublic bool
	err := h.pool.QueryRow(ctx, "SELECT is_public FROM users WHERE id = $1", ownerID).Scan(&isOwnerPublic)
	if err != nil {
		return nil, err
	}

	// Get user's group memberships with the owner
	groupRows, err := h.pool.Query(ctx, `
		SELECT vg.id 
		FROM visibility_groups vg
		JOIN group_members gm ON gm.group_id = vg.id
		WHERE vg.owner_id = $1 AND gm.member_id = $2
	`, ownerID, currentUserID)
	if err != nil {
		return nil, err
	}
	defer groupRows.Close()

	memberGroupIDs := make(map[string]bool)
	for groupRows.Next() {
		var gid string
		if err := groupRows.Scan(&gid); err != nil {
			continue
		}
		memberGroupIDs[gid] = true
	}


	// Get schedules for the date with visibility filtering
	// A schedule is visible if:
	// 1. It has no group associations (general schedule) AND user is member of at least one of owner's groups
	// 2. It has group associations and current user is a member of at least one of those specific groups
	// isPublic only affects catalog visibility (canSeeUser), not schedule visibility
	rows, err := h.pool.Query(ctx, `
		SELECT s.id, s.start_time, s.end_time, s.is_blocked,
			COALESCE(
				ARRAY_AGG(svg.group_id) FILTER (WHERE svg.group_id IS NOT NULL),
				ARRAY[]::UUID[]
			) as group_ids
		FROM schedules s
		LEFT JOIN schedule_visibility_groups svg ON svg.schedule_id = s.id
		WHERE s.user_id = $1
		  AND (
			  (s.type = 'one-time' AND s.date = $2)
			  OR
			  (s.type = 'recurring' AND s.day_of_week = EXTRACT(DOW FROM $2::date))
		  )
		GROUP BY s.id, s.start_time, s.end_time, s.is_blocked
		ORDER BY s.start_time
	`, ownerID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type schedule struct {
		id        string
		startTime string
		endTime   string
		isBlocked bool
		groupIDs  []string
	}

	var allSchedules []schedule
	var groupSchedules []schedule
	var generalSchedules []schedule
	
	for rows.Next() {
		var s schedule
		var groupIDs []string
		if err := rows.Scan(&s.id, &s.startTime, &s.endTime, &s.isBlocked, &groupIDs); err != nil {
			return nil, err
		}
		s.groupIDs = filterEmptyUUIDs(groupIDs)
		allSchedules = append(allSchedules, s)
		
		// Separate into group and general schedules
		if len(s.groupIDs) > 0 {
			groupSchedules = append(groupSchedules, s)
		} else {
			generalSchedules = append(generalSchedules, s)
		}
	}
	
	// Build time ranges covered by ALL group schedules (for exclusion of general slots)
	type timeRange struct {
		start time.Time
		end   time.Time
	}
	var groupTimeRanges []timeRange
	for _, s := range groupSchedules {
		if s.isBlocked {
			continue
		}
		start, _ := time.Parse("15:04:05", s.startTime)
		end, _ := time.Parse("15:04:05", s.endTime)
		groupTimeRanges = append(groupTimeRanges, timeRange{start, end})
	}

	// Determine which schedules are visible to user
	var visibleSchedules []schedule
	
	// Add group schedules that user has access to
	for _, s := range groupSchedules {
		if s.isBlocked {
			continue
		}
		// Check if user is in any of this schedule's groups
		hasAccess := false
		for _, gid := range s.groupIDs {
			if memberGroupIDs[gid] {
				hasAccess = true
				break
			}
		}
		if hasAccess {
			visibleSchedules = append(visibleSchedules, s)
		}
	}
	
	// Add general schedules (slots outside group time ranges)
	// General schedules only visible if isPublic=true
	if isOwnerPublic {
		for _, s := range generalSchedules {
			if s.isBlocked {
				continue
			}
			visibleSchedules = append(visibleSchedules, s)
		}
	}

	// Get booked slots - use slot_date to properly track bookings on recurring schedules
	bookedRows, err := h.pool.Query(ctx, `
		SELECT slot_start_time
		FROM bookings
		WHERE owner_id = $1
		  AND slot_date = $2
		  AND status = 'active'
	`, ownerID, date)
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

	for _, s := range visibleSchedules {
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
			
			// For general schedules (no groups), skip slots that fall within any group schedule time range
			// This ensures group schedules have higher priority
			isGeneralSchedule := len(s.groupIDs) == 0
			if isGeneralSchedule {
				skipSlot := false
				for _, tr := range groupTimeRanges {
					if current.Equal(tr.start) || current.After(tr.start) && current.Before(tr.end) {
						skipSlot = true
						break
					}
				}
				if skipSlot {
					continue
				}
			}
			
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
