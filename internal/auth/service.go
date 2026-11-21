package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode"

	"github.com/rs/zerolog"

	"github.com/TechBuilder-360/Rental-API/ent"
	"github.com/TechBuilder-360/Rental-API/internal/email"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrAccountDisabled    = errors.New("account is disabled")
)

// ValidationError is a user-input error safe to show to clients.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string { return e.Field + ": " + e.Message }

// Result is returned on successful authentication.
type Result struct {
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
	User                  *ent.User
}

// SignUpInput is the data required to register with email and password.
type SignUpInput struct {
	Email     string
	Password  string
	FirstName string
	LastName  string
}

// Service implements email/password and email OTP authentication.
type Service struct {
	users    UserStore
	hasher   *PasswordHasher
	tokens   *TokenManager
	sessions *SessionStore
	otps     *OTPStore
	mailer   email.Dispatcher
	log      zerolog.Logger
}

func NewService(
	users UserStore,
	hasher *PasswordHasher,
	tokens *TokenManager,
	sessions *SessionStore,
	otps *OTPStore,
	mailer email.Dispatcher,
	log zerolog.Logger,
) *Service {
	return &Service{
		users: users, hasher: hasher, tokens: tokens, sessions: sessions,
		otps: otps, mailer: mailer,
		log: log.With().Str("component", "auth").Logger(),
	}
}

// SignUp registers a new user and starts a session.
func (s *Service) SignUp(ctx context.Context, in SignUpInput) (*Result, error) {
	addr, err := NormalizeEmail(in.Email)
	if err != nil {
		return nil, err
	}
	firstName := strings.TrimSpace(in.FirstName)
	lastName := strings.TrimSpace(in.LastName)
	if firstName == "" {
		return nil, &ValidationError{Field: "first_name", Message: "is required"}
	}
	if lastName == "" {
		return nil, &ValidationError{Field: "last_name", Message: "is required"}
	}
	if err := ValidatePassword(in.Password); err != nil {
		return nil, &ValidationError{Field: "password", Message: err.Error()}
	}

	hash, err := s.hasher.Hash(in.Password)
	if err != nil {
		return nil, err
	}
	u, err := s.users.Create(ctx, NewUser{
		Email:        addr,
		PasswordHash: hash,
		FirstName:    firstName,
		LastName:     lastName,
	})
	if err != nil {
		return nil, err
	}

	s.log.Info().Str("user_id", u.ID.String()).Msg("user signed up")
	return s.startSession(ctx, u)
}

// LoginResult is returned when credentials are valid and an OTP is dispatched.
type LoginResult struct {
	Message          string
	ExpiresInSeconds int
}

// Login verifies email and password, then generates and sends an OTP to the user's email.
func (s *Service) Login(ctx context.Context, emailAddr, password string) (*LoginResult, error) {
	addr, err := NormalizeEmail(emailAddr)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	u, err := s.users.GetByEmail(ctx, addr)
	if errors.Is(err, ErrUserNotFound) {
		s.hasher.CompareDummy(password) // equalize timing with the found-user path
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if !s.hasher.Compare(u.Password, password) {
		return nil, ErrInvalidCredentials
	}
	if !u.Active {
		return nil, ErrAccountDisabled
	}

	if err := s.otps.AcquireCooldown(ctx, addr); err != nil {
		return nil, err
	}

	code, err := s.otps.Issue(ctx, addr)
	if err != nil {
		return nil, err
	}
	msg, err := email.OTPMessage(addr, code, s.otps.TTL())
	if err != nil {
		return nil, err
	}
	if err := s.mailer.Dispatch(ctx, msg); err != nil {
		return nil, fmt.Errorf("dispatch otp email: %w", err)
	}

	s.log.Info().Str("user_id", u.ID.String()).Msg("login credentials verified; OTP sent")
	return &LoginResult{
		Message:          "Verification code sent to your email",
		ExpiresInSeconds: int(s.otps.TTL().Seconds()),
	}, nil
}

// OTPTTL returns how long issued codes are valid.
func (s *Service) OTPTTL() time.Duration { return s.otps.TTL() }

// VerifyEmailOTP validates a code and starts a session. A valid code also marks the email as verified.
func (s *Service) VerifyEmailOTP(ctx context.Context, emailAddr, code string) (*Result, error) {
	addr, err := NormalizeEmail(emailAddr)
	if err != nil {
		return nil, ErrOTPInvalid
	}
	if err := s.otps.Verify(ctx, addr, strings.TrimSpace(code)); err != nil {
		return nil, err
	}

	u, err := s.users.GetByEmail(ctx, addr)
	if errors.Is(err, ErrUserNotFound) {
		return nil, ErrOTPInvalid
	}
	if err != nil {
		return nil, err
	}
	if !u.Active {
		return nil, ErrAccountDisabled
	}
	if !u.EmailVerified {
		if err := s.users.MarkEmailVerified(ctx, u.ID); err != nil {
			return nil, err
		}
		u.EmailVerified = true
	}

	return s.startSession(ctx, u)
}

// Refresh exchanges a refresh token for a new access token and a new refresh token.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*Result, error) {
	sess, newRefresh, err := s.sessions.Rotate(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	u, err := s.users.GetByID(ctx, sess.UserID)
	if errors.Is(err, ErrUserNotFound) || (err == nil && !u.Active) {
		_ = s.sessions.Revoke(ctx, sess.ID, sess.UserID)
		return nil, ErrInvalidRefreshToken
	}
	if err != nil {
		return nil, err
	}

	access, accessExp, err := s.tokens.IssueAccessToken(u.ID, u.Email, sess.ID)
	if err != nil {
		return nil, err
	}
	return &Result{
		AccessToken: access, AccessTokenExpiresAt: accessExp,
		RefreshToken: newRefresh, RefreshTokenExpiresAt: time.Now().Add(s.sessions.TTL()),
		User: u,
	}, nil
}

// Logout revokes the caller's current session.
func (s *Service) Logout(ctx context.Context) error {
	p, ok := PrincipalFromContext(ctx)
	if !ok {
		return ErrUnauthenticated
	}
	return s.sessions.Revoke(ctx, p.Claims.SessionID, p.UserID)
}

// CurrentUser loads the authenticated user.
func (s *Service) CurrentUser(ctx context.Context) (*ent.User, error) {
	uid, err := UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	u, err := s.users.GetByID(ctx, uid)
	if errors.Is(err, ErrUserNotFound) {
		return nil, ErrUnauthenticated
	}
	return u, err
}

// Authenticate verifies an access token and that its session is still active.
func (s *Service) Authenticate(ctx context.Context, token string) (*Principal, error) {
	claims, err := s.tokens.ParseAccessToken(token)
	if err != nil {
		return nil, err
	}
	uid, _ := claims.UserID() // validated in ParseAccessToken
	if err := s.sessions.Validate(ctx, claims.SessionID, uid); err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}
	return &Principal{UserID: uid, Claims: claims}, nil
}

func (s *Service) startSession(ctx context.Context, u *ent.User) (*Result, error) {
	sess, refresh, err := s.sessions.Create(ctx, u.ID, u.Email, RequestMetaFromContext(ctx))
	if err != nil {
		return nil, err
	}
	access, accessExp, err := s.tokens.IssueAccessToken(u.ID, u.Email, sess.ID)
	if err != nil {
		return nil, err
	}
	return &Result{
		AccessToken: access, AccessTokenExpiresAt: accessExp,
		RefreshToken: refresh, RefreshTokenExpiresAt: sess.CreatedAt.Add(s.sessions.TTL()),
		User: u,
	}, nil
}

// NormalizeEmail trims and lower-cases an email and validates its format.
func NormalizeEmail(raw string) (string, error) {
	addr := strings.ToLower(strings.TrimSpace(raw))
	parsed, err := mail.ParseAddress(addr)
	if err != nil || parsed.Address != addr || len(addr) > 254 {
		return "", &ValidationError{Field: "email", Message: "is not a valid email address"}
	}
	return addr, nil
}

func normalizePhone(raw string) (string, error) {
	var b strings.Builder
	for i, r := range strings.TrimSpace(raw) {
		switch {
		case unicode.IsDigit(r):
			b.WriteRune(r)
		case r == '+' && i == 0:
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '(' || r == ')':
			// formatting characters are dropped
		default:
			return "", &ValidationError{Field: "phoneNumber", Message: "contains invalid characters"}
		}
	}
	phone := b.String()
	digits := strings.TrimPrefix(phone, "+")
	if len(digits) < 7 || len(digits) > 15 {
		return "", &ValidationError{Field: "phoneNumber", Message: "must have 7 to 15 digits"}
	}
	return phone, nil
}
