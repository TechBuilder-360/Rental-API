package seeders

import (
	"testing"
)

func TestDefaultSeedData(t *testing.T) {
	services := DefaultServices()
	if len(services) < 2 {
		t.Fatalf("expected at least 2 default services, got %d", len(services))
	}

	foundEmail := false
	foundIdentity := false

	for _, s := range services {
		if s.Identifier == ServiceEmailProvider {
			foundEmail = true
			if s.Provider != "brevo" {
				t.Errorf("expected email provider to default to brevo, got %s", s.Provider)
			}
			if s.Disabled {
				t.Errorf("expected email provider to be enabled by default")
			}
		}
		if s.Identifier == ServiceUserIdentityVerification {
			foundIdentity = true
			if s.Provider != "didit" {
				t.Errorf("expected user identity verification provider to be didit, got %s", s.Provider)
			}
			if s.Disabled {
				t.Errorf("expected user identity verification to be enabled by default")
			}
		}
	}

	if !foundEmail {
		t.Errorf("missing %s in default services", ServiceEmailProvider)
	}
	if !foundIdentity {
		t.Errorf("missing %s in default services", ServiceUserIdentityVerification)
	}

	providers := DefaultProviders()
	if len(providers) < 2 {
		t.Fatalf("expected default providers, got %d", len(providers))
	}

	foundBrevo := false
	foundDidit := false
	for _, p := range providers {
		if p.Slug == "brevo" {
			foundBrevo = true
			if !p.Active {
				t.Error("expected brevo to be active")
			}
		}
		if p.Slug == "didit" {
			foundDidit = true
			if !p.Active {
				t.Error("expected didit to be active")
			}
		}
	}

	if !foundBrevo {
		t.Error("missing brevo in default providers")
	}
	if !foundDidit {
		t.Error("missing didit in default providers")
	}
}
