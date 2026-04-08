package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"call-booking/internal/auth"
	"call-booking/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

// setupTestDB creates a database connection for testing.
// Skips the test if the database is unavailable.
func setupTestDB(t *testing.T) *pgxpool.Pool {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5434/call_booking_test?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("Database not available: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		t.Skipf("Database not available: %v", err)
	}

	return pool
}

// cleanupTestData removes test data from the database.
func cleanupTestData(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	// Delete in order to respect foreign key constraints
	_, _ = pool.Exec(ctx, "DELETE FROM bookings")
	_, _ = pool.Exec(ctx, "DELETE FROM group_members")
	_, _ = pool.Exec(ctx, "DELETE FROM visibility_groups")
	_, _ = pool.Exec(ctx, "DELETE FROM schedules")
	_, _ = pool.Exec(ctx, "DELETE FROM users")
}

// createTestUser creates a test user and returns the user ID.
func createTestUser(t *testing.T, pool *pgxpool.Pool, email, password, name string) string {
	ctx := context.Background()

	// Hash password
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	var userID string
	err = pool.QueryRow(ctx,
		"INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3) RETURNING id",
		email, hash, name).Scan(&userID)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return userID
}

// getAuthToken generates a JWT token for testing.
func getAuthToken(userID, email string) string {
	// Set a test secret for JWT
	auth.SetSecret("test-secret-key-minimum-32-characters-long-for-testing-only")

	token, err := auth.GenerateToken(userID, email)
	if err != nil {
		panic("failed to generate test token: " + err.Error())
	}
	return token
}

// makeRequest creates and executes an HTTP request for testing.
func makeRequest(router http.Handler, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// parseResponse parses the response body into the target struct.
func parseResponse(t *testing.T, rr *httptest.ResponseRecorder, target interface{}) {
	if err := json.NewDecoder(rr.Body).Decode(target); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
}

// parseErrorResponse parses an error response.
func parseErrorResponse(t *testing.T, rr *httptest.ResponseRecorder) models.ErrorResponse {
	var errResp models.ErrorResponse
	parseResponse(t, rr, &errResp)
	return errResp
}

// ptrInt32 returns a pointer to an int32 value.
func ptrInt32(v int32) *int32 {
	return &v
}

// strPtr returns a pointer to a string value.
func strPtr(v string) *string {
	return &v
}
