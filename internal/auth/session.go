package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	ErrSessionNotFound     = errors.New("session not found or expired")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
)

const (
	sessionKeyPrefix      = "auth:session:"
	userSessionsKeyPrefix = "auth:user_sessions:"
	refreshSecretBytes    = 32
)

// SessionMeta is client metadata stored with a session.
type SessionMeta struct {
	UserAgent string
	IP        string
}

// Session is the server-side record of a login, stored in Redis.
type Session struct {
	ID         string
	UserID     uuid.UUID
	Email      string
	UserAgent  string
	IP         string
	CreatedAt  time.Time
	LastUsedAt time.Time
}

// SessionStore persists sessions in Redis. Each session holds a hash of the
// current refresh token; the raw refresh token is never stored.
type SessionStore struct {
	rdb *redis.Client
	ttl time.Duration
	now func() time.Time
}

func NewSessionStore(rdb *redis.Client, refreshTTL time.Duration) *SessionStore {
	return &SessionStore{rdb: rdb, ttl: refreshTTL, now: time.Now}
}

// TTL returns the session (refresh token) lifetime.
func (s *SessionStore) TTL() time.Duration { return s.ttl }

func sessionKey(id string) string           { return sessionKeyPrefix + id }
func userSessionsKey(uid uuid.UUID) string { return userSessionsKeyPrefix + uid.String() }

// Create stores a new session and returns it with its refresh token ("<sessionID>.<secret>").
func (s *SessionStore) Create(ctx context.Context, userID uuid.UUID, email string, meta SessionMeta) (*Session, string, error) {
	sid := uuid.NewString()
	secret, err := randomToken(refreshSecretBytes)
	if err != nil {
		return nil, "", err
	}
	now := s.now().UTC()

	pipe := s.rdb.TxPipeline()
	pipe.HSet(ctx, sessionKey(sid), map[string]any{
		"user_id":      userID.String(),
		"email":        email,
		"refresh_hash": hashSecret(secret),
		"user_agent":   meta.UserAgent,
		"ip":           meta.IP,
		"created_at":   now.Format(time.RFC3339),
		"last_used_at": now.Format(time.RFC3339),
	})
	pipe.Expire(ctx, sessionKey(sid), s.ttl)
	pipe.SAdd(ctx, userSessionsKey(userID), sid)
	pipe.Expire(ctx, userSessionsKey(userID), s.ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, "", fmt.Errorf("create session: %w", err)
	}

	return &Session{
		ID: sid, UserID: userID, Email: email,
		UserAgent: meta.UserAgent, IP: meta.IP,
		CreatedAt: now, LastUsedAt: now,
	}, sid + "." + secret, nil
}

// Validate checks that the session exists and belongs to the user.
func (s *SessionStore) Validate(ctx context.Context, sessionID string, userID uuid.UUID) error {
	uid, err := s.rdb.HGet(ctx, sessionKey(sessionID), "user_id").Result()
	if errors.Is(err, redis.Nil) {
		return ErrSessionNotFound
	}
	if err != nil {
		return fmt.Errorf("validate session: %w", err)
	}
	if uid != userID.String() {
		return ErrSessionNotFound
	}
	return nil
}

// rotateScript atomically swaps the refresh hash if the presented one matches.
// If it does not match, the token was already rotated (possible theft), so the
// session is revoked.
// Returns: 1 = rotated, 0 = session missing, -1 = mismatch (session revoked).
var rotateScript = redis.NewScript(`
local current = redis.call("HGET", KEYS[1], "refresh_hash")
if not current then
  return 0
end
if current ~= ARGV[1] then
  redis.call("DEL", KEYS[1])
  return -1
end
redis.call("HSET", KEYS[1], "refresh_hash", ARGV[2], "last_used_at", ARGV[3])
redis.call("PEXPIRE", KEYS[1], ARGV[4])
return 1
`)

// Rotate validates a refresh token and replaces it with a new one (single use).
func (s *SessionStore) Rotate(ctx context.Context, refreshToken string) (*Session, string, error) {
	sid, secret, ok := splitRefreshToken(refreshToken)
	if !ok {
		return nil, "", ErrInvalidRefreshToken
	}
	newSecret, err := randomToken(refreshSecretBytes)
	if err != nil {
		return nil, "", err
	}
	now := s.now().UTC()

	res, err := rotateScript.Run(ctx, s.rdb, []string{sessionKey(sid)},
		hashSecret(secret), hashSecret(newSecret), now.Format(time.RFC3339), s.ttl.Milliseconds(),
	).Int()
	if err != nil {
		return nil, "", fmt.Errorf("rotate refresh token: %w", err)
	}
	if res != 1 {
		return nil, "", ErrInvalidRefreshToken
	}

	sess, err := s.Get(ctx, sid)
	if err != nil {
		return nil, "", err
	}
	s.rdb.Expire(ctx, userSessionsKey(sess.UserID), s.ttl)
	return sess, sid + "." + newSecret, nil
}

// Get loads a session by ID.
func (s *SessionStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	vals, err := s.rdb.HGetAll(ctx, sessionKey(sessionID)).Result()
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	if len(vals) == 0 {
		return nil, ErrSessionNotFound
	}
	uid, err := uuid.Parse(vals["user_id"])
	if err != nil {
		return nil, ErrSessionNotFound
	}
	created, _ := time.Parse(time.RFC3339, vals["created_at"])
	lastUsed, _ := time.Parse(time.RFC3339, vals["last_used_at"])
	return &Session{
		ID: sessionID, UserID: uid, Email: vals["email"],
		UserAgent: vals["user_agent"], IP: vals["ip"],
		CreatedAt: created, LastUsedAt: lastUsed,
	}, nil
}

// Revoke deletes a session. Access tokens tied to it stop working immediately.
func (s *SessionStore) Revoke(ctx context.Context, sessionID string, userID uuid.UUID) error {
	pipe := s.rdb.TxPipeline()
	pipe.Del(ctx, sessionKey(sessionID))
	pipe.SRem(ctx, userSessionsKey(userID), sessionID)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

func splitRefreshToken(token string) (sid, secret string, ok bool) {
	sid, secret, found := strings.Cut(token, ".")
	if !found || secret == "" {
		return "", "", false
	}
	if _, err := uuid.Parse(sid); err != nil {
		return "", "", false
	}
	return sid, secret, true
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}
