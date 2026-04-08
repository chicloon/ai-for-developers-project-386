package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func init() {
	// Set test secret for all tests
	SetSecret("test-secret-key-minimum-32-characters-long-for-testing-only")
}

func TestGenerateToken_Success(t *testing.T) {
	userID := "test-user-id"
	email := "test@example.com"

	token, err := GenerateToken(userID, email)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	if token == "" {
		t.Error("expected token to be non-empty")
	}
}

func TestValidateToken_Success(t *testing.T) {
	userID := "test-user-id"
	email := "test@example.com"

	token, err := GenerateToken(userID, email)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected user ID %s, got %s", userID, claims.UserID)
	}
	if claims.Email != email {
		t.Errorf("expected email %s, got %s", email, claims.Email)
	}
}

func TestValidateToken_InvalidToken(t *testing.T) {
	_, err := ValidateToken("invalid-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestValidateToken_EmptyToken(t *testing.T) {
	_, err := ValidateToken("")
	if err == nil {
		t.Error("expected error for empty token")
	}
}

func TestValidateToken_WrongSignature(t *testing.T) {
	// Create a token with a different secret
	wrongSecret := []byte("wrong-secret-key-minimum-32-characters")
	claims := Claims{
		UserID: "test-user-id",
		Email:  "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(wrongSecret)

	_, err := ValidateToken(tokenString)
	if err == nil {
		t.Error("expected error for token with wrong signature")
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	// Create an expired token
	claims := Claims{
		UserID: "test-user-id",
		Email:  "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // Expired 1 hour ago
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	_, err = ValidateToken(tokenString)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestHashPassword_Success(t *testing.T) {
	password := "password123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if hash == "" {
		t.Error("expected hash to be non-empty")
	}

	if hash == password {
		t.Error("expected hash to be different from password")
	}

	// Verify the hash is valid bcrypt format (starts with $2a$)
	if len(hash) < 7 || hash[:4] != "$2a$" {
		t.Errorf("expected bcrypt hash, got: %s", hash)
	}
}

func TestCheckPassword_Correct(t *testing.T) {
	password := "password123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if !CheckPassword(password, hash) {
		t.Error("expected password to match hash")
	}
}

func TestCheckPassword_Incorrect(t *testing.T) {
	password := "password123"
	wrongPassword := "wrongpassword"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if CheckPassword(wrongPassword, hash) {
		t.Error("expected wrong password to not match hash")
	}
}

func TestCheckPassword_InvalidHash(t *testing.T) {
	if CheckPassword("password", "invalid-hash") {
		t.Error("expected invalid hash to return false")
	}
}

func TestCheckPassword_EmptyHash(t *testing.T) {
	if CheckPassword("password", "") {
		t.Error("expected empty hash to return false")
	}
}

func TestTokenClaims(t *testing.T) {
	userID := "specific-user-id"
	email := "specific@example.com"

	token, err := GenerateToken(userID, email)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	// Check that expiration is set (24 hours from now)
	if claims.ExpiresAt == nil {
		t.Error("expected ExpiresAt to be set")
	}
	if claims.ExpiresAt != nil {
		expectedExpiry := time.Now().Add(24 * time.Hour)
		diff := claims.ExpiresAt.Time.Sub(expectedExpiry)
		if diff < -time.Minute || diff > time.Minute {
			t.Errorf("expected expiry within 1 minute of 24 hours from now, got diff: %v", diff)
		}
	}

	// Check issued at is set
	if claims.IssuedAt == nil {
		t.Error("expected IssuedAt to be set")
	}
}
