package api

import (
	"context"
	"net/http"
	"testing"
	"time"

	"call-booking/internal/auth"
	"call-booking/internal/models"
)

func TestUsersList_VisibleUsers(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create current user
	currentEmail := "current@example.com"
	currentUserID := createTestUser(t, pool, currentEmail, "password123", "Current User")
	currentToken := getAuthToken(currentUserID, currentEmail)

	// Create public user (visible via public group)
	publicUserEmail := "public@example.com"
	publicUserID := createTestUser(t, pool, publicUserEmail, "password123", "Public User")
	_, _ = pool.Exec(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Public', 'public')", publicUserID)

	// Create private user (not visible)
	privateUserEmail := "private@example.com"
	_ = createTestUser(t, pool, privateUserEmail, "password123", "Private User")
	// No group created, so not visible

	// Create user with member group (visible via membership)
	memberUserEmail := "member@example.com"
	memberUserID := createTestUser(t, pool, memberUserEmail, "password123", "Member User")
	var groupID string
	_ = pool.QueryRow(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Work', 'work') RETURNING id", memberUserID).Scan(&groupID)
	_, _ = pool.Exec(ctx, "INSERT INTO group_members (group_id, member_id, added_by) VALUES ($1, $2, $1)", groupID, currentUserID)

	rr := makeRequest(router, "GET", "/api/users", nil, currentToken)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string][]models.User
	parseResponse(t, rr, &resp)

	// Should see public user and member user, but not private user or self
	foundPublic := false
	foundMember := false
	foundPrivate := false
	foundSelf := false

	for _, u := range resp["users"] {
		if u.ID == publicUserID {
			foundPublic = true
		}
		if u.ID == memberUserID {
			foundMember = true
		}
		if u.ID == currentUserID {
			foundSelf = true
		}
		if u.Email == privateUserEmail {
			foundPrivate = true
		}
	}

	if !foundPublic {
		t.Error("expected to find public user")
	}
	if !foundMember {
		t.Error("expected to find member user")
	}
	if foundPrivate {
		t.Error("should not find private user")
	}
	if foundSelf {
		t.Error("should not see self in list")
	}
}

func TestUsersList_EmptyWhenNoVisibleUsers(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	// Create current user
	currentEmail := "current2@example.com"
	currentUserID := createTestUser(t, pool, currentEmail, "password123", "Current User")
	currentToken := getAuthToken(currentUserID, currentEmail)

	rr := makeRequest(router, "GET", "/api/users", nil, currentToken)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string][]models.User
	parseResponse(t, rr, &resp)

	if resp["users"] == nil {
		t.Error("expected empty array, not nil")
	}
	if len(resp["users"]) != 0 {
		t.Errorf("expected 0 users, got %d", len(resp["users"]))
	}
}

func TestUsersList_Unauthorized(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	router := NewRouter(pool)

	rr := makeRequest(router, "GET", "/api/users", nil, "")

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestUsersGet_Success(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create current user
	currentEmail := "current3@example.com"
	currentUserID := createTestUser(t, pool, currentEmail, "password123", "Current User")
	currentToken := getAuthToken(currentUserID, currentEmail)

	// Create public user
	publicUserEmail := "public2@example.com"
	publicUserID := createTestUser(t, pool, publicUserEmail, "password123", "Public User")
	_, _ = pool.Exec(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Public', 'public')", publicUserID)

	rr := makeRequest(router, "GET", "/api/users/"+publicUserID, nil, currentToken)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp models.User
	parseResponse(t, rr, &resp)

	if resp.ID != publicUserID {
		t.Errorf("expected user ID %s, got %s", publicUserID, resp.ID)
	}
	if resp.Email != publicUserEmail {
		t.Errorf("expected email %s, got %s", publicUserEmail, resp.Email)
	}
}

func TestUsersGet_NotVisible(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	// Create current user
	currentEmail := "current4@example.com"
	currentUserID := createTestUser(t, pool, currentEmail, "password123", "Current User")
	currentToken := getAuthToken(currentUserID, currentEmail)

	// Create private user (no visibility)
	privateUserEmail := "private2@example.com"
	privateUserID := createTestUser(t, pool, privateUserEmail, "password123", "Private User")

	rr := makeRequest(router, "GET", "/api/users/"+privateUserID, nil, currentToken)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "you don't have access to this user" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}

func TestUsersGet_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	router := NewRouter(pool)

	// Create current user
	currentEmail := "current5@example.com"
	currentUserID := createTestUser(t, pool, currentEmail, "password123", "Current User")
	currentToken := getAuthToken(currentUserID, currentEmail)

	rr := makeRequest(router, "GET", "/api/users/nonexistent-id", nil, currentToken)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestUsersGet_Unauthorized(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	router := NewRouter(pool)

	rr := makeRequest(router, "GET", "/api/users/some-id", nil, "")

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestUsersSlots_Success(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create current user
	currentEmail := "current6@example.com"
	currentUserID := createTestUser(t, pool, currentEmail, "password123", "Current User")
	currentToken := getAuthToken(currentUserID, currentEmail)

	// Create owner user with public visibility
	ownerEmail := "owner@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Owner User")
	_, _ = pool.Exec(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Public', 'public')", ownerID)

	// Create a recurring schedule for the owner
	tomorrow := time.Now().Add(24 * time.Hour).Format("2006-01-02")
	dayOfWeek := int32(time.Now().Add(24 * time.Hour).Weekday())

	_, err := pool.Exec(ctx,
		"INSERT INTO schedules (user_id, type, day_of_week, start_time, end_time, is_blocked) VALUES ($1, 'recurring', $2, '09:00:00', '17:00:00', false)",
		ownerID, dayOfWeek)
	if err != nil {
		t.Fatalf("failed to create schedule: %v", err)
	}

	rr := makeRequest(router, "GET", "/api/users/"+ownerID+"/slots?date="+tomorrow, nil, currentToken)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string][]models.Slot
	parseResponse(t, rr, &resp)

	// Should have 16 slots (9:00-17:00 = 8 hours = 16 x 30-min slots)
	if len(resp["slots"]) != 16 {
		t.Errorf("expected 16 slots, got %d", len(resp["slots"]))
	}

	// Verify first slot
	if len(resp["slots"]) > 0 {
		firstSlot := resp["slots"][0]
		if firstSlot.StartTime != "09:00" {
			t.Errorf("expected first slot at 09:00, got %s", firstSlot.StartTime)
		}
		if firstSlot.EndTime != "09:30" {
			t.Errorf("expected first slot to end at 09:30, got %s", firstSlot.EndTime)
		}
	}
}

func TestUsersSlots_MissingDate(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create current user
	currentEmail := "current7@example.com"
	currentUserID := createTestUser(t, pool, currentEmail, "password123", "Current User")
	currentToken := getAuthToken(currentUserID, currentEmail)

	// Create owner user with public visibility
	ownerEmail := "owner2@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Owner User")
	_, _ = pool.Exec(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Public', 'public')", ownerID)

	rr := makeRequest(router, "GET", "/api/users/"+ownerID+"/slots", nil, currentToken)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "date parameter is required" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}

func TestUsersSlots_NotVisible(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	// Create current user
	currentEmail := "current8@example.com"
	currentUserID := createTestUser(t, pool, currentEmail, "password123", "Current User")
	currentToken := getAuthToken(currentUserID, currentEmail)

	// Create private user
	privateEmail := "private3@example.com"
	privateID := createTestUser(t, pool, privateEmail, "password123", "Private User")

	tomorrow := time.Now().Add(24 * time.Hour).Format("2006-01-02")
	rr := makeRequest(router, "GET", "/api/users/"+privateID+"/slots?date="+tomorrow, nil, currentToken)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", rr.Code)
	}
}

func TestUsersSlots_Unauthorized(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	router := NewRouter(pool)

	rr := makeRequest(router, "GET", "/api/users/some-id/slots?date=2024-01-01", nil, "")

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestUsersSlots_BlockedSchedule(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create current user
	currentEmail := "current9@example.com"
	currentUserID := createTestUser(t, pool, currentEmail, "password123", "Current User")
	currentToken := getAuthToken(currentUserID, currentEmail)

	// Create owner user with public visibility
	ownerEmail := "owner3@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Owner User")
	_, _ = pool.Exec(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Public', 'public')", ownerID)

	// Create a blocked schedule
	tomorrow := time.Now().Add(24 * time.Hour).Format("2006-01-02")
	dayOfWeek := int32(time.Now().Add(24 * time.Hour).Weekday())

	_, err := pool.Exec(ctx,
		"INSERT INTO schedules (user_id, type, day_of_week, start_time, end_time, is_blocked) VALUES ($1, 'recurring', $2, '00:00:00', '23:59:00', true)",
		ownerID, dayOfWeek)
	if err != nil {
		t.Fatalf("failed to create blocked schedule: %v", err)
	}

	rr := makeRequest(router, "GET", "/api/users/"+ownerID+"/slots?date="+tomorrow, nil, currentToken)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string][]models.Slot
	parseResponse(t, rr, &resp)

	// Blocked schedule should result in no slots
	if len(resp["slots"]) != 0 {
		t.Errorf("expected 0 slots for blocked day, got %d", len(resp["slots"]))
	}
}

func TestUsersSlots_BookedSlot(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create current user (as booker)
	currentEmail := "booker@example.com"
	currentUserID := createTestUser(t, pool, currentEmail, "password123", "Booker User")
	currentToken := getAuthToken(currentUserID, currentEmail)

	// Create owner user with public visibility
	ownerEmail := "owner4@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Owner User")
	_, _ = pool.Exec(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Public', 'public')", ownerID)

	// Create a schedule and a booking
	tomorrow := time.Now().Add(24 * time.Hour).Format("2006-01-02")
	dayOfWeek := int32(time.Now().Add(24 * time.Hour).Weekday())

	var scheduleID string
	err := pool.QueryRow(ctx,
		"INSERT INTO schedules (user_id, type, day_of_week, start_time, end_time, is_blocked) VALUES ($1, 'recurring', $2, '09:00:00', '10:00:00', false) RETURNING id",
		ownerID, dayOfWeek).Scan(&scheduleID)
	if err != nil {
		t.Fatalf("failed to create schedule: %v", err)
	}

	// Create a booking
	_, err = pool.Exec(ctx,
		"INSERT INTO bookings (schedule_id, booker_id, owner_id, status, slot_start_time) VALUES ($1, $2, $3, 'active', '09:00')",
		scheduleID, currentUserID, ownerID)
	if err != nil {
		t.Fatalf("failed to create booking: %v", err)
	}

	rr := makeRequest(router, "GET", "/api/users/"+ownerID+"/slots?date="+tomorrow, nil, currentToken)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string][]models.Slot
	parseResponse(t, rr, &resp)

	// Find the 09:00 slot and check it's marked as booked
	for _, slot := range resp["slots"] {
		if slot.StartTime == "09:00" {
			if !slot.IsBooked {
				t.Error("expected 09:00 slot to be marked as booked")
			}
			break
		}
	}
}

// SetSecret is needed for tests
func init() {
	auth.SetSecret("test-secret-key-minimum-32-characters-long-for-testing-only")
}
