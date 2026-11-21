package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const minJWTSecretBytes = 32

var ErrInvalidToken = errors.New("invalid or expired token")

// Claims are the JWT claims carried by an access token.
type Claims struct {
	// SessionID links the token to a Redis session, so sessions can be revoked.
	SessionID string `json:"sid"`
	Email     string `json:"email"`
	jwt.RegisteredClaims
}

// UserID returns the subject as a UUID.
func (c *Claims) UserID() (uuid.UUID, error) {
	return uuid.Parse(c.Subject)
}

// TokenManager issues and verifies HS256 access tokens.
type TokenManager struct {
	secret    []byte
	issuer    string
	accessTTL time.Duration
	now       func() time.Time
}

func NewTokenManager(secret, issuer string, accessTTL time.Duration) (*TokenManager, error) {
	if len(secret) < minJWTSecretBytes {
		return nil, fmt.Errorf("JWT secret must be at least %d bytes (set JWT_SECRET)", minJWTSecretBytes)
	}
	if accessTTL <= 0 {
		return nil, errors.New("access token TTL must be positive")
	}
	return &TokenManager{
		secret:    []byte(secret),
		issuer:    issuer,
		accessTTL: accessTTL,
		now:       time.Now,
	}, nil
}

// AccessTTL returns the configured access token lifetime.
func (m *TokenManager) AccessTTL() time.Duration { return m.accessTTL }

// IssueAccessToken signs an access token for the user and session.
func (m *TokenManager) IssueAccessToken(userID uuid.UUID, email, sessionID string) (string, time.Time, error) {
	now := m.now()
	exp := now.Add(m.accessTTL)
	claims := Claims{
		SessionID: sessionID,
		Email:     email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			ID:        uuid.NewString(),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return signed, exp, nil
}

// ParseAccessToken verifies the signature, algorithm, issuer and expiry of a token.
func (m *TokenManager) ParseAccessToken(token string) (*Claims, error) {
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(token, claims,
		func(t *jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(m.issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(m.now),
	)
	if err != nil || !parsed.Valid {
		return nil, ErrInvalidToken
	}
	if _, err := claims.UserID(); err != nil || claims.SessionID == "" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
