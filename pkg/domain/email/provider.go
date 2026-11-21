package email

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrProviderNotFound = errors.New("email provider not found")
	ErrProviderDisabled = errors.New("email provider is disabled")
	ErrNoActiveProvider = errors.New("no active email provider available")
)

// Message represents an email message to be sent through a provider.
type Message struct {
	To          string
	Subject     string
	HTMLBody    string
	TextBody    string
	Tag         string
	ReplyTo     string
	Attachments []Attachment
}

// Attachment represents a file attached to an email.
type Attachment struct {
	Name        string
	ContentType string
	Content     []byte
}

// Provider defines the interface that all email providers must implement.
type Provider interface {
	// ID returns the unique machine identifier for the provider (e.g. "brevo", "postmark").
	ID() string
	// Name returns the human-readable display name of the provider (e.g. "Brevo").
	Name() string
	// Slug returns the provider name in lower case (e.g. "brevo").
	Slug() string
	// Send delivers an email message.
	Send(ctx context.Context, msg Message) error
}

// ServiceChecker verifies whether a provider or service is enabled in the system.
type ServiceChecker interface {
	IsServiceDisabled(ctx context.Context, identifier string) (bool, error)
}

// Selector defines the interface for selecting and resolving email providers.
type Selector interface {
	// Select chooses an email provider by identifier, or returns the default active provider.
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

// NewRegistry creates a new provider registry with a default provider identifier.
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

// SetServiceChecker attaches a service availability checker (e.g. database-backed) to the registry.
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

// Select resolves an active, non-disabled provider. If identifier is provided, it attempts
// to use that specific provider; otherwise, it returns the configured default provider.
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

// Providers returns a slice of all registered providers.
func (r *Registry) Providers() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		list = append(list, p)
	}
	return list
}
