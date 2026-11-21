package auth

import (
	"testing"
)

func TestPasswordHasher(t *testing.T) {
	hasher, err := NewPasswordHasher(MinBcryptCost)
	if err != nil {
		t.Fatalf("failed to create hasher: %v", err)
	}

	raw := "supersecret123"
	hash, err := hasher.Hash(raw)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if !hasher.Compare(hash, raw) {
		t.Error("expected password to match hash")
	}

	if hasher.Compare(hash, "wrongpassword") {
		t.Error("expected wrong password not to match")
	}

	if hasher.CompareDummy(raw) {
		t.Error("CompareDummy must always return false")
	}
}

func TestPasswordValidation(t *testing.T) {
	if err := ValidatePassword("short"); err != ErrPasswordTooShort {
		t.Errorf("expected ErrPasswordTooShort, got %v", err)
	}

	if err := ValidatePassword("validlength123"); err != nil {
		t.Errorf("expected valid password, got %v", err)
	}
}
