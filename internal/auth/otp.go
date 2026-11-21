package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrOTPInvalid     = errors.New("invalid or expired code")
	ErrOTPRateLimited = errors.New("a code was sent recently; please wait before requesting another")
)

const (
	otpKeyPrefix      = "auth:otp:email:"
	otpCooldownPrefix = "auth:otp:cooldown:"
	otpDigits         = 6
)

// OTPStore issues and verifies one-time codes stored in Redis.
// Only a SHA-256 hash of each code is stored, under a key with a TTL.
type OTPStore struct {
	rdb         *redis.Client
	ttl         time.Duration
	maxAttempts int
	cooldown    time.Duration
}

func NewOTPStore(rdb *redis.Client, ttl time.Duration, maxAttempts int, cooldown time.Duration) *OTPStore {
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	return &OTPStore{rdb: rdb, ttl: ttl, maxAttempts: maxAttempts, cooldown: cooldown}
}

// TTL returns how long an issued code stays valid.
func (s *OTPStore) TTL() time.Duration { return s.ttl }

func otpKey(email string) string      { return otpKeyPrefix + email }
func cooldownKey(email string) string { return otpCooldownPrefix + email }

// GenerateCode returns a cryptographically secure, uniformly distributed 6-digit code.
func GenerateCode() (string, error) {
	max := big.NewInt(1_000_000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("generate otp: %w", err)
	}
	return fmt.Sprintf("%0*d", otpDigits, n.Int64()), nil
}

// AcquireCooldown enforces a minimum delay between code requests for an email.
// It returns ErrOTPRateLimited if a code was requested within the cooldown window.
func (s *OTPStore) AcquireCooldown(ctx context.Context, email string) error {
	if s.cooldown <= 0 {
		return nil
	}
	ok, err := s.rdb.SetNX(ctx, cooldownKey(email), 1, s.cooldown).Result()
	if err != nil {
		return fmt.Errorf("otp cooldown: %w", err)
	}
	if !ok {
		return ErrOTPRateLimited
	}
	return nil
}

// Issue generates a new code for the email, replacing any previous one.
// The code's hash is stored with an expiry of s.ttl (5 minutes by default).
func (s *OTPStore) Issue(ctx context.Context, email string) (string, error) {
	code, err := GenerateCode()
	if err != nil {
		return "", err
	}
	key := otpKey(email)
	pipe := s.rdb.TxPipeline()
	pipe.Del(ctx, key)
	pipe.HSet(ctx, key, "hash", hashSecret(code), "attempts", 0)
	pipe.Expire(ctx, key, s.ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", fmt.Errorf("store otp: %w", err)
	}
	return code, nil
}

// verifyScript atomically checks a code. Codes are single use: a correct code is
// deleted, and too many wrong attempts delete the code too.
// Returns: 1 = valid, 0 = missing/expired, -1 = wrong code, -2 = wrong and now locked out.
var verifyScript = redis.NewScript(`
local hash = redis.call("HGET", KEYS[1], "hash")
if not hash then
  return 0
end
if hash == ARGV[1] then
  redis.call("DEL", KEYS[1])
  return 1
end
local attempts = redis.call("HINCRBY", KEYS[1], "attempts", 1)
if attempts >= tonumber(ARGV[2]) then
  redis.call("DEL", KEYS[1])
  return -2
end
return -1
`)

// Verify checks a submitted code. It returns ErrOTPInvalid for any failure.
func (s *OTPStore) Verify(ctx context.Context, email, code string) error {
	if len(code) != otpDigits {
		return ErrOTPInvalid
	}
	res, err := verifyScript.Run(ctx, s.rdb, []string{otpKey(email)}, hashSecret(code), s.maxAttempts).Int()
	if err != nil {
		return fmt.Errorf("verify otp: %w", err)
	}
	if res != 1 {
		return ErrOTPInvalid
	}
	return nil
}
