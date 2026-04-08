package api

import (
	"net/http"
	"testing"

	"call-booking/internal/models"
)

func TestAuthRegister_Success(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	req := models.RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}

	rr := makeRequest(router, "POST", "/api/auth/register", req, "")

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp models.AuthResponse
	parseResponse(t, rr, &resp)

	if resp.User.Email != req.Email {
		t.Errorf("expected email %s, got %s", req.Email, resp.User.Email)
	}
	if resp.User.Name != req.Name {
		t.Errorf("expected name %s, got %s", req.Name, resp.User.Name)
	}
	if resp.Token == "" {
		t.Error("expected token to be present")
	}
}

func TestAuthRegister_DuplicateEmail(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	// Create first user
	req := models.RegisterRequest{
		Email:    "duplicate@example.com",
		Password: "password123",
		Name:     "First User",
	}
	rr := makeRequest(router, "POST", "/api/auth/register", req, "")
	if rr.Code != http.StatusCreated {
		t.Fatalf("failed to create first user: %d", rr.Code)
	}

	// Try to create second user with same email
	req2 := models.RegisterRequest{
		Email:    "duplicate@example.com",
		Password: "password456",
		Name:     "Second User",
	}
	rr = makeRequest(router, "POST", "/api/auth/register", req2, "")

	if rr.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "user with this email already exists" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}

func TestAuthRegister_MissingFields(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	testCases := []struct {
		name    string
		req     models.RegisterRequest
		wantErr string
	}{
		{
			name:    "missing email",
			req:     models.RegisterRequest{Password: "pass", Name: "Test"},
			wantErr: "email, password and name are required",
		},
		{
			name:    "missing password",
			req:     models.RegisterRequest{Email: "test@test.com", Name: "Test"},
			wantErr: "email, password and name are required",
		},
		{
			name:    "missing name",
			req:     models.RegisterRequest{Email: "test@test.com", Password: "pass"},
			wantErr: "email, password and name are required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rr := makeRequest(router, "POST", "/api/auth/register", tc.req, "")
			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", rr.Code)
			}
			errResp := parseErrorResponse(t, rr)
			if errResp.Error != tc.wantErr {
				t.Errorf("expected error %q, got %q", tc.wantErr, errResp.Error)
			}
		})
	}
}

func TestAuthLogin_Success(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	// Create user
	email := "login@example.com"
	password := "password123"
	createTestUser(t, pool, email, password, "Login Test User")

	// Login
	req := models.LoginRequest{
		Email:    email,
		Password: password,
	}
	rr := makeRequest(router, "POST", "/api/auth/login", req, "")

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp models.AuthResponse
	parseResponse(t, rr, &resp)

	if resp.User.Email != email {
		t.Errorf("expected email %s, got %s", email, resp.User.Email)
	}
	if resp.Token == "" {
		t.Error("expected token to be present")
	}
}

func TestAuthLogin_InvalidCredentials(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	// Create user
	email := "login2@example.com"
	createTestUser(t, pool, email, "correctpassword", "Login Test User")

	// Try to login with wrong password
	req := models.LoginRequest{
		Email:    email,
		Password: "wrongpassword",
	}
	rr := makeRequest(router, "POST", "/api/auth/login", req, "")

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "invalid email or password" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}

func TestAuthLogin_NonExistentUser(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	req := models.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "password123",
	}
	rr := makeRequest(router, "POST", "/api/auth/login", req, "")

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "invalid email or password" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}

func TestAuthMe_Success(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	// Create user and get token
	email := "me@example.com"
	userID := createTestUser(t, pool, email, "password123", "Me Test User")
	token := getAuthToken(userID, email)

	rr := makeRequest(router, "GET", "/api/auth/me", nil, token)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp models.User
	parseResponse(t, rr, &resp)

	if resp.ID != userID {
		t.Errorf("expected user ID %s, got %s", userID, resp.ID)
	}
	if resp.Email != email {
		t.Errorf("expected email %s, got %s", email, resp.Email)
	}
}

func TestAuthMe_MissingToken(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	rr := makeRequest(router, "GET", "/api/auth/me", nil, "")

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "missing authorization header" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}

func TestAuthMe_InvalidToken(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	rr := makeRequest(router, "GET", "/api/auth/me", nil, "invalid-token")

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "invalid authorization header format" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}
