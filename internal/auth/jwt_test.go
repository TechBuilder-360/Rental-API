package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTokenManager(t *testing.T) {
	secret := "a-very-long-secret-key-that-is-at-least-32-bytes"
	mgr, err := NewTokenManager(secret, "test-issuer", 15*time.Minute)
	if err != nil {
		t.Fatalf("failed to create token manager: %v", err)
	}

	userID := uuid.New()
	email := "user@example.com"
	sessionID := uuid.NewString()

	token, exp, err := mgr.IssueAccessToken(userID, email, sessionID)
	if err != nil {
		t.Fatalf("failed to issue token: %v", err)
	}

	if token == "" {
		t.Fatal("expected token to not be empty")
	}
	if exp.Before(time.Now()) {
		t.Fatal("expected expiry to be in the future")
	}

	claims, err := mgr.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("failed to parse valid token: %v", err)
	}

	parsedID, err := claims.UserID()
	if err != nil {
		t.Fatalf("failed to extract user id: %v", err)
	}

	if parsedID != userID {
		t.Errorf("expected user id %v, got %v", userID, parsedID)
	}
	if claims.Email != email {
		t.Errorf("expected email %s, got %s", email, claims.Email)
	}
	if claims.SessionID != sessionID {
		t.Errorf("expected session id %s, got %s", sessionID, claims.SessionID)
	}

	// Invalid token should fail
	_, err = mgr.ParseAccessToken("invalid.jwt.token")
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}
