package verification

import (
	"context"
	"errors"
	"testing"
)

type mockProvider struct {
	id   string
	name string
}

func (m *mockProvider) ID() string   { return m.id }
func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) Slug() string { return m.id }

func (m *mockProvider) CreateSession(ctx context.Context, req CreateSessionRequest) (*CreateSessionResponse, error) {
	return &CreateSessionResponse{
		SessionID:       "mock-session-123",
		VerificationURL: "https://verify.mock.com/" + req.VendorData,
		Status:          StatusCreated,
	}, nil
}

func (m *mockProvider) GetDecision(ctx context.Context, sessionID string) (*DecisionResponse, error) {
	return &DecisionResponse{
		SessionID: sessionID,
		Status:    StatusApproved,
		Decision: &DecisionDetail{
			Decision:  "Approved",
			RiskScore: 0.01,
		},
	}, nil
}

func (m *mockProvider) ParseWebhook(rawBody []byte, signature string, timestamp int64) (*WebhookEvent, error) {
	return &WebhookEvent{
		SessionID: "mock-session-123",
		Status:    StatusApproved,
	}, nil
}

type mockChecker struct {
	disabled map[string]bool
	err      error
}

func (m *mockChecker) IsServiceDisabled(_ context.Context, identifier string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.disabled[identifier], nil
}

func TestRegistryOperations(t *testing.T) {
	p1 := &mockProvider{id: "didit", name: "Didit"}
	p2 := &mockProvider{id: "veriff", name: "Veriff"}

	reg := NewRegistry("didit", p1)

	// Test Get
	got, err := reg.Get("didit")
	if err != nil {
		t.Fatalf("failed to get didit: %v", err)
	}
	if got.Name() != "Didit" {
		t.Errorf("expected Didit, got %s", got.Name())
	}

	// Test Register
	if err := reg.Register(p2); err != nil {
		t.Fatalf("failed to register veriff: %v", err)
	}

	// Test Select default
	selected, err := reg.Select(context.Background())
	if err != nil {
		t.Fatalf("failed to select default: %v", err)
	}
	if selected.ID() != "didit" {
		t.Errorf("expected didit, got %s", selected.ID())
	}

	// Test Select specific
	selected, err = reg.Select(context.Background(), "veriff")
	if err != nil {
		t.Fatalf("failed to select veriff: %v", err)
	}
	if selected.ID() != "veriff" {
		t.Errorf("expected veriff, got %s", selected.ID())
	}

	// Test Providers list
	all := reg.Providers()
	if len(all) != 2 {
		t.Errorf("expected 2 providers, got %d", len(all))
	}
}

func TestRegistryServiceChecker(t *testing.T) {
	p1 := &mockProvider{id: "didit", name: "Didit"}
	reg := NewRegistry("didit", p1)

	checker := &mockChecker{
		disabled: map[string]bool{
			"didit": true,
		},
	}
	reg.SetServiceChecker(checker)

	_, err := reg.Select(context.Background(), "didit")
	if !errors.Is(err, ErrProviderDisabled) {
		t.Errorf("expected ErrProviderDisabled, got: %v", err)
	}
}

func TestSessionStatusHelpers(t *testing.T) {
	if !StatusApproved.IsTerminal() {
		t.Error("expected StatusApproved to be terminal")
	}
	if !StatusDeclined.IsTerminal() {
		t.Error("expected StatusDeclined to be terminal")
	}
	if !StatusExpired.IsTerminal() {
		t.Error("expected StatusExpired to be terminal")
	}
	if !StatusAbandoned.IsTerminal() {
		t.Error("expected StatusAbandoned to be terminal")
	}
	if StatusCreated.IsTerminal() {
		t.Error("expected StatusCreated to not be terminal")
	}
	if StatusInProgress.IsTerminal() {
		t.Error("expected StatusInProgress to not be terminal")
	}
	if StatusReview.IsTerminal() {
		t.Error("expected StatusReview to not be terminal")
	}

	if !StatusApproved.IsApproved() {
		t.Error("expected StatusApproved.IsApproved() to be true")
	}
	if StatusDeclined.IsApproved() {
		t.Error("expected StatusDeclined.IsApproved() to be false")
	}

	if !StatusDeclined.IsDeclined() {
		t.Error("expected StatusDeclined.IsDeclined() to be true")
	}
	if StatusApproved.IsDeclined() {
		t.Error("expected StatusApproved.IsDeclined() to be false")
	}
}
