package api

import (
	"context"
	"net/http"
	"testing"

	"call-booking/internal/models"
)

func TestSchedulesList_Empty(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	// Create user
	email := "schedules@example.com"
	userID := createTestUser(t, pool, email, "password123", "Schedules Test User")
	token := getAuthToken(userID, email)

	rr := makeRequest(router, "GET", "/api/my/schedules", nil, token)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string][]models.Schedule
	parseResponse(t, rr, &resp)

	if resp["schedules"] == nil {
		t.Error("expected empty array, not nil")
	}
	if len(resp["schedules"]) != 0 {
		t.Errorf("expected 0 schedules, got %d", len(resp["schedules"]))
	}
}

func TestSchedulesList_WithSchedules(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create user
	email := "schedules2@example.com"
	userID := createTestUser(t, pool, email, "password123", "Schedules Test User")
	token := getAuthToken(userID, email)

	// Insert test schedules
	_, err := pool.Exec(ctx, "INSERT INTO schedules (user_id, type, day_of_week, start_time, end_time, is_blocked) VALUES ($1, 'recurring', 1, '09:00:00', '17:00:00', false)", userID)
	if err != nil {
		t.Fatalf("failed to insert schedule: %v", err)
	}
	_, err = pool.Exec(ctx, "INSERT INTO schedules (user_id, type, date, start_time, end_time, is_blocked) VALUES ($1, 'one-time', '2026-04-15', '10:00:00', '14:00:00', false)", userID)
	if err != nil {
		t.Fatalf("failed to insert schedule: %v", err)
	}

	rr := makeRequest(router, "GET", "/api/my/schedules", nil, token)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string][]models.Schedule
	parseResponse(t, rr, &resp)

	if len(resp["schedules"]) != 2 {
		t.Errorf("expected 2 schedules, got %d", len(resp["schedules"]))
	}

	// Check types
	recurringFound := false
	oneTimeFound := false
	for _, s := range resp["schedules"] {
		if s.Type == "recurring" {
			recurringFound = true
			if s.DayOfWeek == nil || *s.DayOfWeek != 1 {
				t.Error("expected day_of_week to be 1")
			}
		}
		if s.Type == "one-time" {
			oneTimeFound = true
			if s.Date == nil || *s.Date != "2026-04-15" {
				t.Errorf("expected date to be 2026-04-15, got %v", *s.Date)
			}
		}
	}
	if !recurringFound {
		t.Error("expected to find recurring schedule")
	}
	if !oneTimeFound {
		t.Error("expected to find one-time schedule")
	}
}

func TestSchedulesList_Unauthorized(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	router := NewRouter(pool)

	rr := makeRequest(router, "GET", "/api/my/schedules", nil, "")

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestSchedulesCreate_Recurring(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	// Create user
	email := "schedules3@example.com"
	userID := createTestUser(t, pool, email, "password123", "Schedules Test User")
	token := getAuthToken(userID, email)

	req := models.CreateScheduleRequest{
		Type:      "recurring",
		DayOfWeek: ptrInt32(1), // Monday
		StartTime: "09:00",
		EndTime:   "17:00",
		IsBlocked: false,
	}

	rr := makeRequest(router, "POST", "/api/my/schedules", req, token)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp models.Schedule
	parseResponse(t, rr, &resp)

	if resp.UserID != userID {
		t.Errorf("expected user ID %s, got %s", userID, resp.UserID)
	}
	if resp.Type != "recurring" {
		t.Errorf("expected type recurring, got %s", resp.Type)
	}
	if resp.DayOfWeek == nil || *resp.DayOfWeek != 1 {
		t.Error("expected day_of_week to be 1")
	}
	if resp.StartTime != "09:00:00" {
		t.Errorf("expected start_time 09:00:00, got %s", resp.StartTime)
	}
}

func TestSchedulesCreate_OneTime(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	// Create user
	email := "schedules4@example.com"
	userID := createTestUser(t, pool, email, "password123", "Schedules Test User")
	token := getAuthToken(userID, email)

	date := "2026-04-20"
	req := models.CreateScheduleRequest{
		Type:      "one-time",
		Date:      &date,
		StartTime: "10:00",
		EndTime:   "14:00",
		IsBlocked: false,
	}

	rr := makeRequest(router, "POST", "/api/my/schedules", req, token)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp models.Schedule
	parseResponse(t, rr, &resp)

	if resp.Type != "one-time" {
		t.Errorf("expected type one-time, got %s", resp.Type)
	}
	if resp.Date == nil || *resp.Date != date {
		t.Errorf("expected date %s, got %v", date, resp.Date)
	}
}

func TestSchedulesCreate_Blocked(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	// Create user
	email := "schedules5@example.com"
	userID := createTestUser(t, pool, email, "password123", "Schedules Test User")
	token := getAuthToken(userID, email)

	req := models.CreateScheduleRequest{
		Type:      "recurring",
		DayOfWeek: ptrInt32(6), // Saturday
		StartTime: "00:00",
		EndTime:   "23:59",
		IsBlocked: true,
	}

	rr := makeRequest(router, "POST", "/api/my/schedules", req, token)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp models.Schedule
	parseResponse(t, rr, &resp)

	if !resp.IsBlocked {
		t.Error("expected is_blocked to be true")
	}
}

func TestSchedulesCreate_InvalidType(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	// Create user
	email := "schedules6@example.com"
	userID := createTestUser(t, pool, email, "password123", "Schedules Test User")
	token := getAuthToken(userID, email)

	req := models.CreateScheduleRequest{
		Type:      "invalid-type",
		DayOfWeek: ptrInt32(1),
		StartTime: "09:00",
		EndTime:   "17:00",
	}

	rr := makeRequest(router, "POST", "/api/my/schedules", req, token)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "type must be 'recurring' or 'one-time'" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}

func TestSchedulesCreate_MissingTimes(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	// Create user
	email := "schedules7@example.com"
	userID := createTestUser(t, pool, email, "password123", "Schedules Test User")
	token := getAuthToken(userID, email)

	testCases := []struct {
		name      string
		startTime string
		endTime   string
	}{
		{name: "missing start", startTime: "", endTime: "17:00"},
		{name: "missing end", startTime: "09:00", endTime: ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := models.CreateScheduleRequest{
				Type:      "recurring",
				DayOfWeek: ptrInt32(1),
				StartTime: tc.startTime,
				EndTime:   tc.endTime,
			}

			rr := makeRequest(router, "POST", "/api/my/schedules", req, token)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", rr.Code)
			}

			errResp := parseErrorResponse(t, rr)
			if errResp.Error != "start_time and end_time are required" {
				t.Errorf("unexpected error message: %s", errResp.Error)
			}
		})
	}
}

func TestSchedulesCreate_Unauthorized(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	router := NewRouter(pool)

	req := models.CreateScheduleRequest{
		Type:      "recurring",
		DayOfWeek: ptrInt32(1),
		StartTime: "09:00",
		EndTime:   "17:00",
	}

	rr := makeRequest(router, "POST", "/api/my/schedules", req, "")

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestSchedulesUpdate_Success(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create user
	email := "schedules8@example.com"
	userID := createTestUser(t, pool, email, "password123", "Schedules Test User")
	token := getAuthToken(userID, email)

	// Insert a schedule
	var scheduleID string
	dayOfWeek := int32(1)
	err := pool.QueryRow(ctx, "INSERT INTO schedules (user_id, type, day_of_week, start_time, end_time, is_blocked) VALUES ($1, 'recurring', $2, '09:00:00', '17:00:00', false) RETURNING id", userID, dayOfWeek).Scan(&scheduleID)
	if err != nil {
		t.Fatalf("failed to insert schedule: %v", err)
	}

	req := models.CreateScheduleRequest{
		Type:      "recurring",
		DayOfWeek: ptrInt32(2), // Tuesday
		StartTime: "08:00",
		EndTime:   "16:00",
		IsBlocked: false,
	}

	rr := makeRequest(router, "PUT", "/api/my/schedules/"+scheduleID, req, token)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp models.Schedule
	parseResponse(t, rr, &resp)

	if resp.ID != scheduleID {
		t.Errorf("expected schedule ID %s, got %s", scheduleID, resp.ID)
	}
	if resp.DayOfWeek == nil || *resp.DayOfWeek != 2 {
		t.Error("expected day_of_week to be updated to 2")
	}
	if resp.StartTime != "08:00:00" {
		t.Errorf("expected start_time 08:00:00, got %s", resp.StartTime)
	}
}

func TestSchedulesUpdate_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	// Create user
	email := "schedules9@example.com"
	userID := createTestUser(t, pool, email, "password123", "Schedules Test User")
	token := getAuthToken(userID, email)

	req := models.CreateScheduleRequest{
		Type:      "recurring",
		DayOfWeek: ptrInt32(1),
		StartTime: "09:00",
		EndTime:   "17:00",
	}

	rr := makeRequest(router, "PUT", "/api/my/schedules/nonexistent-id", req, token)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "schedule not found" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}

func TestSchedulesUpdate_NotOwner(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create two users
	ownerEmail := "owner@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Owner User")

	otherEmail := "other@example.com"
	otherID := createTestUser(t, pool, otherEmail, "password123", "Other User")
	otherToken := getAuthToken(otherID, otherEmail)

	// Insert a schedule for owner
	var scheduleID string
	dayOfWeek := int32(1)
	err := pool.QueryRow(ctx, "INSERT INTO schedules (user_id, type, day_of_week, start_time, end_time, is_blocked) VALUES ($1, 'recurring', $2, '09:00:00', '17:00:00', false) RETURNING id", ownerID, dayOfWeek).Scan(&scheduleID)
	if err != nil {
		t.Fatalf("failed to insert schedule: %v", err)
	}

	req := models.CreateScheduleRequest{
		Type:      "recurring",
		DayOfWeek: ptrInt32(1),
		StartTime: "09:00",
		EndTime:   "17:00",
	}

	rr := makeRequest(router, "PUT", "/api/my/schedules/"+scheduleID, req, otherToken)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404 (not visible to other user), got %d", rr.Code)
	}
}

func TestSchedulesDelete_Success(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create user
	email := "schedules10@example.com"
	userID := createTestUser(t, pool, email, "password123", "Schedules Test User")
	token := getAuthToken(userID, email)

	// Insert a schedule
	var scheduleID string
	dayOfWeek := int32(1)
	err := pool.QueryRow(ctx, "INSERT INTO schedules (user_id, type, day_of_week, start_time, end_time, is_blocked) VALUES ($1, 'recurring', $2, '09:00:00', '17:00:00', false) RETURNING id", userID, dayOfWeek).Scan(&scheduleID)
	if err != nil {
		t.Fatalf("failed to insert schedule: %v", err)
	}

	rr := makeRequest(router, "DELETE", "/api/my/schedules/"+scheduleID, nil, token)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestSchedulesDelete_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	// Create user
	email := "schedules11@example.com"
	userID := createTestUser(t, pool, email, "password123", "Schedules Test User")
	token := getAuthToken(userID, email)

	rr := makeRequest(router, "DELETE", "/api/my/schedules/nonexistent-id", nil, token)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "schedule not found" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}

func TestSchedulesDelete_NotOwner(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create two users
	ownerEmail := "owner2@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Owner User")

	otherEmail := "other2@example.com"
	otherID := createTestUser(t, pool, otherEmail, "password123", "Other User")
	otherToken := getAuthToken(otherID, otherEmail)

	// Insert a schedule for owner
	var scheduleID string
	dayOfWeek := int32(1)
	err := pool.QueryRow(ctx, "INSERT INTO schedules (user_id, type, day_of_week, start_time, end_time, is_blocked) VALUES ($1, 'recurring', $2, '09:00:00', '17:00:00', false) RETURNING id", ownerID, dayOfWeek).Scan(&scheduleID)
	if err != nil {
		t.Fatalf("failed to insert schedule: %v", err)
	}

	rr := makeRequest(router, "DELETE", "/api/my/schedules/"+scheduleID, nil, otherToken)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404 (not visible to other user), got %d", rr.Code)
	}
}

func TestSchedulesDelete_Unauthorized(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	router := NewRouter(pool)

	rr := makeRequest(router, "DELETE", "/api/my/schedules/some-id", nil, "")

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}
