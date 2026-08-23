package jwt

import (
	"testing"
	"time"
)

func TestGenerateAndParseToken(t *testing.T) {
	secret := "test-secret-key-12345"
	userID := "user-uuid-123"
	role := "admin"

	tokenStr, err := GenerateToken(userID, role, secret, time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := ParseToken(tokenStr, secret)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}

	if claims.Subject != userID {
		t.Errorf("expected Subject %q, got %q", userID, claims.Subject)
	}
	if claims.Role != role {
		t.Errorf("expected Role %q, got %q", role, claims.Role)
	}
	if claims.IssuedAt == nil {
		t.Error("expected IssuedAt to be set, got nil")
	}
	if claims.ExpiresAt == nil {
		t.Error("expected ExpiresAt to be set, got nil")
	}
}

func TestInvalidSecret(t *testing.T) {
	tokenStr, err := GenerateToken("user-1", "user", "secret-1", time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	_, err = ParseToken(tokenStr, "wrong-secret")
	if err == nil {
		t.Error("expected error when parsing with wrong secret, got nil")
	}
}
