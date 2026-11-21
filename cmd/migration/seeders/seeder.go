package seeders

import (
	"context"
	"fmt"

	"github.com/TechBuilder-360/Rental-API/ent"
	"github.com/TechBuilder-360/Rental-API/ent/provider"
	"github.com/TechBuilder-360/Rental-API/ent/service"
)

const (
	// ServiceEmailProvider is the service identifier for the email provider.
	ServiceEmailProvider = "email-provider"

	// ServiceUserIdentityVerification is the service identifier for user identity verification.
	ServiceUserIdentityVerification = "user-identity-verification"
)

// ServiceSeedData defines the structure for initial service entries.
type ServiceSeedData struct {
	Identifier string
	Name       string
	Provider   string
	Disabled   bool
}

// ProviderSeedData defines the structure for initial provider entries.
type ProviderSeedData struct {
	Name   string
	Slug   string
	Active bool
}

// DefaultServices returns the default service configuration records.
func DefaultServices() []ServiceSeedData {
	return []ServiceSeedData{
		{
			Identifier: ServiceEmailProvider,
			Name:       "Email Provider",
			Provider:   "brevo",
			Disabled:   false,
		},
		{
			Identifier: ServiceUserIdentityVerification,
			Name:       "User Identity Verification",
			Provider:   "didit",
			Disabled:   false,
		},
	}
}

// DefaultProviders returns the default provider configuration records.
func DefaultProviders() []ProviderSeedData {
	return []ProviderSeedData{
		{
			Name:   "Brevo",
			Slug:   "brevo",
			Active: true,
		},
		{
			Name:   "Didit",
			Slug:   "didit",
			Active: true,
		},
		{
			Name:   "Postmark",
			Slug:   "postmark",
			Active: true,
		},
	}
}

// Seed populates the database with initial providers and services.
// It is idempotent and safe to run multiple times.
func Seed(ctx context.Context, client *ent.Client) error {
	if client == nil {
		return fmt.Errorf("ent client cannot be nil")
	}

	// 1. Seed Providers
	for _, p := range DefaultProviders() {
		exists, err := client.Provider.Query().
			Where(provider.SlugEQ(p.Slug)).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("check provider %s: %w", p.Slug, err)
		}

		if !exists {
			if _, err := client.Provider.Create().
				SetName(p.Name).
				SetSlug(p.Slug).
				SetActive(p.Active).
				Save(ctx); err != nil {
				return fmt.Errorf("create provider %s: %w", p.Slug, err)
			}
		}
	}

	// 2. Seed Services
	for _, s := range DefaultServices() {
		existing, err := client.Service.Query().
			Where(service.IdentifierEQ(s.Identifier)).
			Only(ctx)
		if err != nil && !ent.IsNotFound(err) {
			return fmt.Errorf("check service %s: %w", s.Identifier, err)
		}

		if ent.IsNotFound(err) {
			if _, err := client.Service.Create().
				SetIdentifier(s.Identifier).
				SetName(s.Name).
				SetProvider(s.Provider).
				SetDisabled(s.Disabled).
				Save(ctx); err != nil {
				return fmt.Errorf("create service %s: %w", s.Identifier, err)
			}
		} else if existing.Provider == "" {
			// Backfill provider column if empty on existing record
			if err := client.Service.UpdateOne(existing).
				SetProvider(s.Provider).
				Exec(ctx); err != nil {
				return fmt.Errorf("update service %s provider: %w", s.Identifier, err)
			}
		}
	}

	return nil
}
