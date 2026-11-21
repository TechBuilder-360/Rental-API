package email

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog"

	"github.com/TechBuilder-360/Rental-API/cmd/migration/seeders"
	"github.com/TechBuilder-360/Rental-API/ent"
	"github.com/TechBuilder-360/Rental-API/ent/provider"
	"github.com/TechBuilder-360/Rental-API/ent/service"
)

var (
	ErrServiceNotFound  = errors.New("service configuration not found")
	ErrServiceDisabled  = errors.New("service is disabled")
	ErrProviderNotFound = errors.New("provider not found")
	ErrProviderInactive = errors.New("provider is inactive")
)

// DynamicSender dynamically selects the active email provider using the 'email-provider'
// record in the services table and verifies that the provider is marked active in the providers table.
type DynamicSender struct {
	db        *ent.Client
	providers map[string]Sender
	defaultID string
	log       zerolog.Logger
}

// NewDynamicSender creates a new dynamically resolving email sender.
func NewDynamicSender(db *ent.Client, providers map[string]Sender, defaultID string, log zerolog.Logger) *DynamicSender {
	return &DynamicSender{
		db:        db,
		providers: providers,
		defaultID: defaultID,
		log:       log.With().Str("component", "dynamic-email-sender").Logger(),
	}
}

// ResolveProvider inspects the database 'email-provider' service and provider record
// to determine which email provider implementation to use.
func (s *DynamicSender) ResolveProvider(ctx context.Context) (Sender, string, error) {
	targetSlug := s.defaultID

	if s.db != nil {
		// 1. Check services table for 'email-provider'
		svc, err := s.db.Service.Query().
			Where(service.IdentifierEQ(seeders.ServiceEmailProvider)).
			Only(ctx)
		if err != nil && !ent.IsNotFound(err) {
			return nil, "", fmt.Errorf("query email service: %w", err)
		}

		if svc != nil {
			if svc.Disabled {
				return nil, "", fmt.Errorf("%w: %s", ErrServiceDisabled, seeders.ServiceEmailProvider)
			}
			if svc.Provider != "" {
				targetSlug = strings.ToLower(strings.TrimSpace(svc.Provider))
			}
		}

		// 2. Check providers table to verify target provider is active
		prov, err := s.db.Provider.Query().
			Where(provider.SlugEQ(targetSlug)).
			Only(ctx)
		if err != nil && !ent.IsNotFound(err) {
			return nil, "", fmt.Errorf("query provider %s: %w", targetSlug, err)
		}

		if prov != nil && !prov.Active {
			return nil, "", fmt.Errorf("%w: provider %s is inactive", ErrProviderInactive, targetSlug)
		}
	}

	// 3. Retrieve matching implementation
	sender, exists := s.providers[targetSlug]
	if !exists {
		// Fallback to default if target is not registered
		sender, exists = s.providers[s.defaultID]
		if !exists {
			return nil, "", fmt.Errorf("%w: %s", ErrProviderNotFound, targetSlug)
		}
		targetSlug = s.defaultID
	}

	return sender, targetSlug, nil
}

// Send transmits the email message through the dynamically determined active provider.
func (s *DynamicSender) Send(ctx context.Context, msg Message) error {
	sender, slug, err := s.ResolveProvider(ctx)
	if err != nil {
		return err
	}

	s.log.Debug().Str("provider", slug).Str("to", msg.To).Msg("sending email through resolved provider")
	return sender.Send(ctx, msg)
}
