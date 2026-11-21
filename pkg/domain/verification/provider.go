package verification

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrProviderNotFound = errors.New("verification provider not found")
	ErrProviderDisabled = errors.New("verification provider is disabled")
	ErrNoActiveProvider = errors.New("no active verification provider available")
)

// SessionStatus represents the state of a verification session.
type SessionStatus string

const (
	StatusCreated    SessionStatus = "Created"
	StatusInProgress SessionStatus = "InProgress"
	StatusApproved   SessionStatus = "Approved"
	StatusDeclined   SessionStatus = "Declined"
	StatusReview     SessionStatus = "Review"
	StatusExpired    SessionStatus = "Expired"
	StatusAbandoned  SessionStatus = "Abandoned"
)

// IsTerminal returns true if the verification status is in a final state.
func (s SessionStatus) IsTerminal() bool {
	switch s {
	case StatusApproved, StatusDeclined, StatusExpired, StatusAbandoned:
		return true
	default:
		return false
	}
}

// IsApproved returns true if the status indicates successful verification.
func (s SessionStatus) IsApproved() bool {
	return s == StatusApproved
}

// IsDeclined returns true if the status indicates failed verification.
func (s SessionStatus) IsDeclined() bool {
	return s == StatusDeclined
}

// CreateSessionRequest defines the parameters for creating a new user verification session.
type CreateSessionRequest struct {
	WorkflowID  string `json:"workflow_id,omitempty"`
	VendorData  string `json:"vendor_data,omitempty"`
	CallbackURL string `json:"callback_url,omitempty"`
	Language    string `json:"language,omitempty"`
}

// CreateSessionResponse holds the output of a newly created verification session.
type CreateSessionResponse struct {
	SessionID       string        `json:"session_id"`
	SessionKind     string        `json:"session_kind,omitempty"`
	VerificationURL string        `json:"verification_url"`
	SessionToken    string        `json:"session_token,omitempty"`
	Status          SessionStatus `json:"status"`
	CreatedAt       string        `json:"created_at,omitempty"`
	ExpiresAt       string        `json:"expires_at,omitempty"`
}

// DecisionDetail provides specific decision metadata, risk score, and reason.
type DecisionDetail struct {
	Decision  string  `json:"decision"`
	RiskScore float64 `json:"risk_score,omitempty"`
	Reason    string  `json:"reason,omitempty"`
}

// DecisionResponse represents the full verification decision for a session.
type DecisionResponse struct {
	SessionID   string          `json:"session_id"`
	SessionKind string          `json:"session_kind,omitempty"`
	VendorData  string          `json:"vendor_data,omitempty"`
	Status      SessionStatus   `json:"status"`
	Decision    *DecisionDetail `json:"decision,omitempty"`
	CreatedAt   string          `json:"created_at,omitempty"`
	UpdatedAt   string          `json:"updated_at,omitempty"`
}

// WebhookEvent represents a parsed webhook notification sent by a verification provider.
type WebhookEvent struct {
	EventID    string          `json:"event_id,omitempty"`
	EventType  string          `json:"event_type"`
	SessionID  string          `json:"session_id"`
	VendorData string          `json:"vendor_data,omitempty"`
	Status     SessionStatus   `json:"status"`
	Decision   *DecisionDetail `json:"decision,omitempty"`
	Timestamp  int64           `json:"timestamp,omitempty"`
}

// Provider defines the interface that all user identity/verification providers must implement.
type Provider interface {
	// ID returns the unique machine identifier for the provider (e.g. "didit").
	ID() string
	// Name returns the human-readable display name of the provider (e.g. "Didit").
	Name() string
	// Slug returns the provider name in lower case (e.g. "didit").
	Slug() string

	// CreateSession creates a new hosted verification session for a user.
	CreateSession(ctx context.Context, req CreateSessionRequest) (*CreateSessionResponse, error)

	// GetDecision retrieves the verification decision and status for an existing session.
	GetDecision(ctx context.Context, sessionID string) (*DecisionResponse, error)

	// ParseWebhook verifies and decodes a provider webhook event.
	ParseWebhook(rawBody []byte, signature string, timestamp int64) (*WebhookEvent, error)
}

// ServiceChecker verifies whether a provider or service is enabled in the system.
type ServiceChecker interface {
	IsServiceDisabled(ctx context.Context, identifier string) (bool, error)
}

// Selector defines the interface for selecting and resolving verification providers.
type Selector interface {
	// Select chooses a verification provider by identifier, or returns the default active provider.
	Select(ctx context.Context, identifier ...string) (Provider, error)
	// Register registers a provider.
	Register(provider Provider) error
	// Get retrieves a registered provider by its identifier.
	Get(identifier string) (Provider, error)
	// Providers returns a list of all registered providers.
	Providers() []Provider
}

// Registry is an in-memory implementation of the Selector interface with optional ServiceChecker support.
type Registry struct {
	providers map[string]Provider
	defaultID string
	checker   ServiceChecker
	mu        sync.RWMutex
}

// NewRegistry creates a new verification provider registry with an optional default provider identifier.
func NewRegistry(defaultID string, providers ...Provider) *Registry {
	r := &Registry{
		providers: make(map[string]Provider),
		defaultID: defaultID,
	}
	for _, p := range providers {
		_ = r.Register(p)
	}
	return r
}

// SetServiceChecker attaches a service availability checker to the registry.
func (r *Registry) SetServiceChecker(checker ServiceChecker) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checker = checker
}

// Register adds a provider to the registry.
func (r *Registry) Register(p Provider) error {
	if p == nil {
		return errors.New("cannot register nil provider")
	}
	id := p.ID()
	if id == "" {
		return errors.New("provider ID cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[id] = p
	return nil
}

// Get retrieves a provider by its identifier without status checking.
func (r *Registry) Get(identifier string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, exists := r.providers[identifier]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, identifier)
	}
	return p, nil
}

// Select resolves an active, non-disabled verification provider. If identifier is provided,
// it attempts to use that specific provider; otherwise, it returns the configured default provider.
func (r *Registry) Select(ctx context.Context, identifier ...string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	targetID := r.defaultID
	if len(identifier) > 0 && identifier[0] != "" {
		targetID = identifier[0]
	}

	if targetID == "" {
		return nil, ErrNoActiveProvider
	}

	p, exists := r.providers[targetID]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, targetID)
	}

	if r.checker != nil {
		disabled, err := r.checker.IsServiceDisabled(ctx, targetID)
		if err != nil {
			return nil, fmt.Errorf("check service status for %s: %w", targetID, err)
		}
		if disabled {
			return nil, fmt.Errorf("%w: %s", ErrProviderDisabled, targetID)
		}
	}

	return p, nil
}

// Providers returns a slice of all registered verification providers.
func (r *Registry) Providers() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		list = append(list, p)
	}
	return list
}
