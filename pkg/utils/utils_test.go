package utils

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "secret"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !CheckPassword(password, hash) {
		t.Errorf("Expected password check to pass")
	}

	if CheckPassword("wrongpassword", hash) {
		t.Errorf("Expected password check to fail")
	}
}

func TestJWT(t *testing.T) {
	secret := "supersecret"
	userID := "123"
	role := "user"

	token, err := GenerateToken(userID, role, secret)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	claims, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Expected UserID %s, got %s", userID, claims.UserID)
	}

	if claims.Role != role {
		t.Errorf("Expected Role %s, got %s", role, claims.Role)
	}

	_, err = ValidateToken(token, "wrongsecret")
	if err == nil {
		t.Errorf("Expected error with wrong secret")
	}
}
