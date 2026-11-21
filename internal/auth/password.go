package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	// MinBcryptCost is the minimum accepted bcrypt cost factor.
	MinBcryptCost = 12

	MinPasswordLength = 8
	// MaxPasswordBytes is bcrypt's input limit; longer inputs are rejected
	// rather than silently truncated.
	MaxPasswordBytes = 72
)

var (
	ErrPasswordTooShort = fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	ErrPasswordTooLong  = fmt.Errorf("password must be at most %d bytes", MaxPasswordBytes)
)

// PasswordHasher hashes and verifies passwords with bcrypt.
type PasswordHasher struct {
	cost int
	// dummyHash is compared against when a user does not exist, so that
	// login latency does not reveal whether an account exists.
	dummyHash []byte
}

// NewPasswordHasher returns a hasher using the given cost (raised to MinBcryptCost if lower).
func NewPasswordHasher(cost int) (*PasswordHasher, error) {
	if cost < MinBcryptCost {
		cost = MinBcryptCost
	}
	if cost > bcrypt.MaxCost {
		return nil, fmt.Errorf("bcrypt cost %d exceeds max %d", cost, bcrypt.MaxCost)
	}
	dummy, err := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), cost)
	if err != nil {
		return nil, fmt.Errorf("generate dummy hash: %w", err)
	}
	return &PasswordHasher{cost: cost, dummyHash: dummy}, nil
}

// ValidatePassword enforces the password policy.
func ValidatePassword(password string) error {
	if len([]rune(password)) < MinPasswordLength {
		return ErrPasswordTooShort
	}
	if len(password) > MaxPasswordBytes {
		return ErrPasswordTooLong
	}
	return nil
}

// Hash validates and hashes a password.
func (h *PasswordHasher) Hash(password string) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// Compare reports whether password matches hash.
func (h *PasswordHasher) Compare(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// CompareDummy performs a bcrypt comparison against a dummy hash to equalize timing
// when the user does not exist. It always returns false.
func (h *PasswordHasher) CompareDummy(password string) bool {
	_ = bcrypt.CompareHashAndPassword(h.dummyHash, []byte(password))
	return false
}

// IsPolicyError reports whether err is a password policy violation.
func IsPolicyError(err error) bool {
	return errors.Is(err, ErrPasswordTooShort) || errors.Is(err, ErrPasswordTooLong)
}
