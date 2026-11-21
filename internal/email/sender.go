package email

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/TechBuilder-360/Rental-API/ent"
	"github.com/TechBuilder-360/Rental-API/internal/config"
)

// Message is a transactional email.
type Message struct {
	To       string
	Subject  string
	TextBody string
	HTMLBody string
	// Tag is an optional provider tag for analytics (e.g. "otp").
	Tag string
}

// Sender delivers a single email.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

// PermanentError marks a failure that will not succeed on retry
// (e.g. invalid recipient, bad credentials).
type PermanentError struct{ Err error }

func (e *PermanentError) Error() string { return e.Err.Error() }
func (e *PermanentError) Unwrap() error { return e.Err }

// IsPermanent reports whether err should not be retried.
func IsPermanent(err error) bool {
	var p *PermanentError
	return errors.As(err, &p)
}

// NewSender builds the Sender for the configured provider.
func NewSender(cfg config.EmailConfig, log zerolog.Logger) (Sender, error) {
	switch cfg.Provider {
	case "brevo":
		if cfg.BrevoAPIKey == "" {
			return nil, errors.New("BREVO_API_KEY is required when EMAIL_PROVIDER=brevo")
		}
		return NewBrevoSender(cfg.BrevoAPIKey, cfg.From, cfg.FromName)
	case "postmark":
		if cfg.PostmarkServerToken == "" {
			return nil, errors.New("POSTMARK_SERVER_TOKEN is required when EMAIL_PROVIDER=postmark")
		}
		return NewPostmarkSender(cfg.PostmarkServerToken, cfg.From, cfg.PostmarkMessageStream)
	case "log":
		log.Warn().Msg("EMAIL_PROVIDER=log: emails are written to the log, not delivered")
		return NewLogSender(log), nil
	default:
		return nil, fmt.Errorf("unknown EMAIL_PROVIDER %q (expected brevo, postmark or log)", cfg.Provider)
	}
}

// NewDynamicSenderWithConfig builds a DynamicSender containing all configured email providers.
func NewDynamicSenderWithConfig(db *ent.Client, cfg config.EmailConfig, log zerolog.Logger) (Sender, error) {
	senders := make(map[string]Sender)

	senders["log"] = NewLogSender(log)

	if cfg.BrevoAPIKey != "" {
		brevoSender, err := NewBrevoSender(cfg.BrevoAPIKey, cfg.From, cfg.FromName)
		if err == nil {
			senders["brevo"] = brevoSender
		}
	}

	if cfg.PostmarkServerToken != "" {
		postmarkSender, err := NewPostmarkSender(cfg.PostmarkServerToken, cfg.From, cfg.PostmarkMessageStream)
		if err == nil {
			senders["postmark"] = postmarkSender
		}
	}

	defaultID := cfg.Provider
	if defaultID == "" {
		defaultID = "log"
	}

	return NewDynamicSender(db, senders, defaultID, log), nil
}

// LogSender writes emails to the log instead of sending them. For local development only.
type LogSender struct{ log zerolog.Logger }

func NewLogSender(log zerolog.Logger) *LogSender {
	return &LogSender{log: log.With().Str("component", "email").Logger()}
}

func (s *LogSender) Send(_ context.Context, msg Message) error {
	s.log.Info().
		Str("to", msg.To).
		Str("subject", msg.Subject).
		Str("tag", msg.Tag).
		Str("body", msg.TextBody).
		Msg("email (not sent: log provider)")
	return nil
}
