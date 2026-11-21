package email

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type mockProvider struct {
	id   string
	name string
	sent []Message
}

func (m *mockProvider) ID() string   { return m.id }
func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) Slug() string { return strings.ToLower(m.name) }
func (m *mockProvider) Send(ctx context.Context, msg Message) error {
	m.sent = append(m.sent, msg)
	return nil
}

type mockChecker struct {
	disabled map[string]bool
}

func (m *mockChecker) IsServiceDisabled(ctx context.Context, identifier string) (bool, error) {
	return m.disabled[identifier], nil
}

func TestRegistrySelect(t *testing.T) {
	p1 := &mockProvider{id: "brevo", name: "Brevo"}
	p2 := &mockProvider{id: "postmark", name: "Postmark"}

	reg := NewRegistry("brevo", p1, p2)

	// 1. Select default
	selected, err := reg.Select(context.Background())
	if err != nil {
		t.Fatalf("expected default provider, got error: %v", err)
	}
	if selected.ID() != "brevo" {
		t.Errorf("expected brevo, got %s", selected.ID())
	}

	// 2. Select by ID
	selected, err = reg.Select(context.Background(), "postmark")
	if err != nil {
		t.Fatalf("expected postmark, got error: %v", err)
	}
	if selected.ID() != "postmark" {
		t.Errorf("expected postmark, got %s", selected.ID())
	}

	// 3. Select unknown ID
	_, err = reg.Select(context.Background(), "unknown")
	if !errors.Is(err, ErrProviderNotFound) {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestRegistryServiceCheckerDisabled(t *testing.T) {
	p1 := &mockProvider{id: "brevo", name: "Brevo"}
	reg := NewRegistry("brevo", p1)

	checker := &mockChecker{
		disabled: map[string]bool{
			"brevo": true,
		},
	}
	reg.SetServiceChecker(checker)

	_, err := reg.Select(context.Background(), "brevo")
	if !errors.Is(err, ErrProviderDisabled) {
		t.Errorf("expected ErrProviderDisabled, got %v", err)
	}
}
