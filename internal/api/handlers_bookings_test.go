package api

import (
	"context"
	"net/http"
	"testing"
	"time"

	"call-booking/internal/models"
)

func TestBookingsList_Empty(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	// Create user
	email := "bookings@example.com"
	userID := createTestUser(t, pool, email, "password123", "Bookings Test User")
	token := getAuthToken(userID, email)

	rr := makeRequest(router, "GET", "/api/my/bookings", nil, token)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string][]models.Booking
	parseResponse(t, rr, &resp)

	if resp["bookings"] == nil {
		t.Error("expected empty array, not nil")
	}
	if len(resp["bookings"]) != 0 {
		t.Errorf("expected 0 bookings, got %d", len(resp["bookings"]))
	}
}

func TestBookingsList_AsBooker(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create booker user
	bookerEmail := "booker@example.com"
	bookerID := createTestUser(t, pool, bookerEmail, "password123", "Booker User")
	bookerToken := getAuthToken(bookerID, bookerEmail)

	// Create owner user with public visibility
	ownerEmail := "ownerbooked@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Owner User")
	_, _ = pool.Exec(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Public', 'public')", ownerID)

	// Create a schedule for owner
	var scheduleID string
	dayOfWeek := int32(1)
	err := pool.QueryRow(ctx, "INSERT INTO schedules (user_id, type, day_of_week, start_time, end_time, is_blocked) VALUES ($1, 'recurring', $2, '09:00:00', '17:00:00', false) RETURNING id", ownerID, dayOfWeek).Scan(&scheduleID)
	if err != nil {
		t.Fatalf("failed to create schedule: %v", err)
	}

	// Create a booking
	_, err = pool.Exec(ctx, "INSERT INTO bookings (schedule_id, booker_id, owner_id, status) VALUES ($1, $2, $3, 'active')", scheduleID, bookerID, ownerID)
	if err != nil {
		t.Fatalf("failed to create booking: %v", err)
	}

	rr := makeRequest(router, "GET", "/api/my/bookings", nil, bookerToken)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string][]models.Booking
	parseResponse(t, rr, &resp)

	if len(resp["bookings"]) != 1 {
		t.Errorf("expected 1 booking, got %d", len(resp["bookings"]))
	}

	if len(resp["bookings"]) > 0 {
		booking := resp["bookings"][0]
		if booking.Booker.ID != bookerID {
			t.Errorf("expected booker ID %s, got %s", bookerID, booking.Booker.ID)
		}
		if booking.Owner.ID != ownerID {
			t.Errorf("expected owner ID %s, got %s", ownerID, booking.Owner.ID)
		}
	}
}

func TestBookingsList_AsOwner(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create booker user
	bookerEmail := "booker2@example.com"
	bookerID := createTestUser(t, pool, bookerEmail, "password123", "Booker User")

	// Create owner user
	ownerEmail := "ownerbooked2@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Owner User")
	ownerToken := getAuthToken(ownerID, ownerEmail)
	_, _ = pool.Exec(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Public', 'public')", ownerID)

	// Create a schedule for owner
	var scheduleID string
	dayOfWeek := int32(1)
	err := pool.QueryRow(ctx, "INSERT INTO schedules (user_id, type, day_of_week, start_time, end_time, is_blocked) VALUES ($1, 'recurring', $2, '09:00:00', '17:00:00', false) RETURNING id", ownerID, dayOfWeek).Scan(&scheduleID)
	if err != nil {
		t.Fatalf("failed to create schedule: %v", err)
	}

	// Create a booking
	_, err = pool.Exec(ctx, "INSERT INTO bookings (schedule_id, booker_id, owner_id, status) VALUES ($1, $2, $3, 'active')", scheduleID, bookerID, ownerID)
	if err != nil {
		t.Fatalf("failed to create booking: %v", err)
	}

	rr := makeRequest(router, "GET", "/api/my/bookings", nil, ownerToken)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string][]models.Booking
	parseResponse(t, rr, &resp)

	if len(resp["bookings"]) != 1 {
		t.Errorf("expected 1 booking, got %d", len(resp["bookings"]))
	}

	if len(resp["bookings"]) > 0 {
		booking := resp["bookings"][0]
		if booking.Owner.ID != ownerID {
			t.Errorf("expected owner ID %s, got %s", ownerID, booking.Owner.ID)
		}
	}
}

func TestBookingsList_Unauthorized(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	router := NewRouter(pool)

	rr := makeRequest(router, "GET", "/api/my/bookings", nil, "")

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestBookingsCreate_Success(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create booker user
	bookerEmail := "booker3@example.com"
	bookerID := createTestUser(t, pool, bookerEmail, "password123", "Booker User")
	bookerToken := getAuthToken(bookerID, bookerEmail)

	// Create owner user with public visibility
	ownerEmail := "ownerbooked3@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Owner User")
	_, _ = pool.Exec(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Public', 'public')", ownerID)

	// Create a schedule for owner
	var scheduleID string
	dayOfWeek := int32(1)
	err := pool.QueryRow(ctx, "INSERT INTO schedules (user_id, type, day_of_week, start_time, end_time, is_blocked) VALUES ($1, 'recurring', $2, '09:00:00', '17:00:00', false) RETURNING id", ownerID, dayOfWeek).Scan(&scheduleID)
	if err != nil {
		t.Fatalf("failed to create schedule: %v", err)
	}

	req := models.CreateBookingRequest{
		OwnerID:       ownerID,
		ScheduleID:    scheduleID,
		SlotStartTime: "10:00",
	}

	rr := makeRequest(router, "POST", "/api/my/bookings", req, bookerToken)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp models.Booking
	parseResponse(t, rr, &resp)

	if resp.Booker.ID != bookerID {
		t.Errorf("expected booker ID %s, got %s", bookerID, resp.Booker.ID)
	}
	if resp.Owner.ID != ownerID {
		t.Errorf("expected owner ID %s, got %s", ownerID, resp.Owner.ID)
	}
	if resp.Status != "active" {
		t.Errorf("expected status active, got %s", resp.Status)
	}
}

func TestBookingsCreate_MissingOwnerID(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	// Create booker user
	bookerEmail := "booker4@example.com"
	bookerID := createTestUser(t, pool, bookerEmail, "password123", "Booker User")
	bookerToken := getAuthToken(bookerID, bookerEmail)

	req := models.CreateBookingRequest{
		ScheduleID: "some-id",
	}

	rr := makeRequest(router, "POST", "/api/my/bookings", req, bookerToken)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "owner_id and schedule_id are required" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}

func TestBookingsCreate_NotVisibleOwner(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create booker user
	bookerEmail := "booker5@example.com"
	bookerID := createTestUser(t, pool, bookerEmail, "password123", "Booker User")
	bookerToken := getAuthToken(bookerID, bookerEmail)

	// Create owner user without public visibility
	ownerEmail := "privateowner@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Private Owner")
	// No visibility group created, so not visible

	// Create a schedule for owner
	var scheduleID string
	dayOfWeek := int32(1)
	err := pool.QueryRow(ctx, "INSERT INTO schedules (user_id, type, day_of_week, start_time, end_time, is_blocked) VALUES ($1, 'recurring', $2, '09:00:00', '17:00:00', false) RETURNING id", ownerID, dayOfWeek).Scan(&scheduleID)
	if err != nil {
		t.Fatalf("failed to create schedule: %v", err)
	}

	req := models.CreateBookingRequest{
		OwnerID:    ownerID,
		ScheduleID: scheduleID,
	}

	rr := makeRequest(router, "POST", "/api/my/bookings", req, bookerToken)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "you don't have access to this user" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}

func TestBookingsCreate_DuplicateBooking(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create booker user
	bookerEmail := "booker6@example.com"
	bookerID := createTestUser(t, pool, bookerEmail, "password123", "Booker User")
	bookerToken := getAuthToken(bookerID, bookerEmail)

	// Create owner user with public visibility
	ownerEmail := "ownerdup@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Owner User")
	_, _ = pool.Exec(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Public', 'public')", ownerID)

	// Create a schedule for owner
	var scheduleID string
	dayOfWeek := int32(1)
	err := pool.QueryRow(ctx, "INSERT INTO schedules (user_id, type, day_of_week, start_time, end_time, is_blocked) VALUES ($1, 'recurring', $2, '09:00:00', '17:00:00', false) RETURNING id", ownerID, dayOfWeek).Scan(&scheduleID)
	if err != nil {
		t.Fatalf("failed to create schedule: %v", err)
	}

	// Create first booking
	_, err = pool.Exec(ctx, "INSERT INTO bookings (schedule_id, booker_id, owner_id, status, slot_start_time) VALUES ($1, $2, $3, 'active', '10:00')", scheduleID, bookerID, ownerID)
	if err != nil {
		t.Fatalf("failed to create booking: %v", err)
	}

	// Try to create duplicate booking
	req := models.CreateBookingRequest{
		OwnerID:       ownerID,
		ScheduleID:    scheduleID,
		SlotStartTime: "10:00",
	}

	rr := makeRequest(router, "POST", "/api/my/bookings", req, bookerToken)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "this slot is already booked" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}

func TestBookingsCreate_Unauthorized(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	router := NewRouter(pool)

	req := models.CreateBookingRequest{
		OwnerID:    "owner-id",
		ScheduleID: "schedule-id",
	}

	rr := makeRequest(router, "POST", "/api/my/bookings", req, "")

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestBookingsCancel_AsBooker(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create booker user
	bookerEmail := "booker7@example.com"
	bookerID := createTestUser(t, pool, bookerEmail, "password123", "Booker User")
	bookerToken := getAuthToken(bookerID, bookerEmail)

	// Create owner user with public visibility
	ownerEmail := "ownercancel@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Owner User")
	_, _ = pool.Exec(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Public', 'public')", ownerID)

	// Create a schedule for owner
	var scheduleID string
	dayOfWeek := int32(1)
	err := pool.QueryRow(ctx, "INSERT INTO schedules (user_id, type, day_of_week, start_time, end_time, is_blocked) VALUES ($1, 'recurring', $2, '09:00:00', '17:00:00', false) RETURNING id", ownerID, dayOfWeek).Scan(&scheduleID)
	if err != nil {
		t.Fatalf("failed to create schedule: %v", err)
	}

	// Create a booking
	var bookingID string
	err = pool.QueryRow(ctx, "INSERT INTO bookings (schedule_id, booker_id, owner_id, status) VALUES ($1, $2, $3, 'active') RETURNING id", scheduleID, bookerID, ownerID).Scan(&bookingID)
	if err != nil {
		t.Fatalf("failed to create booking: %v", err)
	}

	rr := makeRequest(router, "DELETE", "/api/my/bookings/"+bookingID, nil, bookerToken)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestBookingsCancel_AsOwner(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create booker user
	bookerEmail := "booker8@example.com"
	bookerID := createTestUser(t, pool, bookerEmail, "password123", "Booker User")

	// Create owner user
	ownerEmail := "ownercancel2@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Owner User")
	ownerToken := getAuthToken(ownerID, ownerEmail)
	_, _ = pool.Exec(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Public', 'public')", ownerID)

	// Create a schedule for owner
	var scheduleID string
	dayOfWeek := int32(1)
	err := pool.QueryRow(ctx, "INSERT INTO schedules (user_id, type, day_of_week, start_time, end_time, is_blocked) VALUES ($1, 'recurring', $2, '09:00:00', '17:00:00', false) RETURNING id", ownerID, dayOfWeek).Scan(&scheduleID)
	if err != nil {
		t.Fatalf("failed to create schedule: %v", err)
	}

	// Create a booking
	var bookingID string
	err = pool.QueryRow(ctx, "INSERT INTO bookings (schedule_id, booker_id, owner_id, status) VALUES ($1, $2, $3, 'active') RETURNING id", scheduleID, bookerID, ownerID).Scan(&bookingID)
	if err != nil {
		t.Fatalf("failed to create booking: %v", err)
	}

	rr := makeRequest(router, "DELETE", "/api/my/bookings/"+bookingID, nil, ownerToken)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestBookingsCancel_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	// Create user
	email := "booker9@example.com"
	userID := createTestUser(t, pool, email, "password123", "User")
	token := getAuthToken(userID, email)

	rr := makeRequest(router, "DELETE", "/api/my/bookings/nonexistent-id", nil, token)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "booking not found" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}

func TestBookingsCancel_NotAuthorized(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create booker user
	bookerEmail := "booker10@example.com"
	bookerID := createTestUser(t, pool, bookerEmail, "password123", "Booker User")

	// Create owner user
	ownerEmail := "ownercancel3@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Owner User")
	_, _ = pool.Exec(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Public', 'public')", ownerID)

	// Create third user (unauthorized)
	otherEmail := "otherbooker@example.com"
	otherID := createTestUser(t, pool, otherEmail, "password123", "Other User")
	otherToken := getAuthToken(otherID, otherEmail)

	// Create a schedule for owner
	var scheduleID string
	dayOfWeek := int32(1)
	err := pool.QueryRow(ctx, "INSERT INTO schedules (user_id, type, day_of_week, start_time, end_time, is_blocked) VALUES ($1, 'recurring', $2, '09:00:00', '17:00:00', false) RETURNING id", ownerID, dayOfWeek).Scan(&scheduleID)
	if err != nil {
		t.Fatalf("failed to create schedule: %v", err)
	}

	// Create a booking
	var bookingID string
	err = pool.QueryRow(ctx, "INSERT INTO bookings (schedule_id, booker_id, owner_id, status) VALUES ($1, $2, $3, 'active') RETURNING id", scheduleID, bookerID, ownerID).Scan(&bookingID)
	if err != nil {
		t.Fatalf("failed to create booking: %v", err)
	}

	rr := makeRequest(router, "DELETE", "/api/my/bookings/"+bookingID, nil, otherToken)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "you don't have permission to cancel this booking" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}

func TestBookingsCancel_Unauthorized(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	router := NewRouter(pool)

	rr := makeRequest(router, "DELETE", "/api/my/bookings/some-id", nil, "")

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestBookingsList_WithCancelled(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create booker user
	bookerEmail := "booker11@example.com"
	bookerID := createTestUser(t, pool, bookerEmail, "password123", "Booker User")
	bookerToken := getAuthToken(bookerID, bookerEmail)

	// Create owner user
	ownerEmail := "ownercancelled@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Owner User")
	_, _ = pool.Exec(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Public', 'public')", ownerID)

	// Create a schedule for owner
	var scheduleID string
	dayOfWeek := int32(1)
	err := pool.QueryRow(ctx, "INSERT INTO schedules (user_id, type, day_of_week, start_time, end_time, is_blocked) VALUES ($1, 'recurring', $2, '09:00:00', '17:00:00', false) RETURNING id", ownerID, dayOfWeek).Scan(&scheduleID)
	if err != nil {
		t.Fatalf("failed to create schedule: %v", err)
	}

	// Create a cancelled booking
	cancelledAt := time.Now()
	_, err = pool.Exec(ctx, "INSERT INTO bookings (schedule_id, booker_id, owner_id, status, cancelled_at, cancelled_by) VALUES ($1, $2, $3, 'cancelled', $4, $2)", scheduleID, bookerID, ownerID, cancelledAt)
	if err != nil {
		t.Fatalf("failed to create cancelled booking: %v", err)
	}

	rr := makeRequest(router, "GET", "/api/my/bookings", nil, bookerToken)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string][]models.Booking
	parseResponse(t, rr, &resp)

	if len(resp["bookings"]) != 1 {
		t.Errorf("expected 1 booking, got %d", len(resp["bookings"]))
	}

	if len(resp["bookings"]) > 0 {
		booking := resp["bookings"][0]
		if booking.Status != "cancelled" {
			t.Errorf("expected status cancelled, got %s", booking.Status)
		}
		if booking.CancelledAt == nil {
			t.Error("expected cancelled_at to be set")
		}
	}
}
