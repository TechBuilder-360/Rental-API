package verification

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TechBuilder-360/Rental-API/ent"
)

func TestHostedSessionViewMapping(t *testing.T) {
	now := time.Now()
	userID := uuid.New()
	sessionUUID := uuid.New()
	templateUUID := uuid.New()
	score := 0.05
	decision := "Approved"

	session := &ent.HostedSession{
		ID:              sessionUUID,
		UserID:          userID,
		TemplateID:      &templateUUID,
		Provider:        "didit",
		SessionID:       "sess-abc",
		SessionToken:    "token-xyz",
		VerificationURL: "https://verify.didit.me/sess-abc",
		VendorData:      userID.String(),
		Status:          string(StatusApproved),
		Decision:        &decision,
		RiskScore:       &score,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	view := toHostedSessionView(session)
	if view == nil {
		t.Fatal("expected non-nil view")
	}

	if view.ID != sessionUUID {
		t.Errorf("expected session id %s, got %s", sessionUUID, view.ID)
	}
	if view.UserID != userID {
		t.Errorf("expected user id %s, got %s", userID, view.UserID)
	}
	if view.VerificationURL != "https://verify.didit.me/sess-abc" {
		t.Errorf("expected url https://verify.didit.me/sess-abc, got %s", view.VerificationURL)
	}
	if view.Status != StatusApproved {
		t.Errorf("expected status Approved, got %s", view.Status)
	}
	if view.RiskScore == nil || *view.RiskScore != 0.05 {
		t.Errorf("expected risk score 0.05, got %v", view.RiskScore)
	}
}

func TestUserIdentityViewMapping(t *testing.T) {
	now := time.Now()
	userID := uuid.New()
	sessionID := uuid.New()
	identityID := uuid.New()
	firstName := "Jane"
	lastName := "Doe"
	docType := "passport"
	docNumber := "P987654321"

	identity := &ent.UserIdentity{
		ID:              identityID,
		UserID:          userID,
		HostedSessionID: &sessionID,
		Provider:        "didit",
		FirstName:       &firstName,
		LastName:        &lastName,
		DocumentType:    &docType,
		DocumentNumber:  &docNumber,
		IsVerified:      true,
		VerifiedAt:      &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	view := toUserIdentityView(identity)
	if view == nil {
		t.Fatal("expected non-nil view")
	}

	if view.ID != identityID {
		t.Errorf("expected identity id %s, got %s", identityID, view.ID)
	}
	if view.UserID != userID {
		t.Errorf("expected user id %s, got %s", userID, view.UserID)
	}
	if view.FirstName == nil || *view.FirstName != "Jane" {
		t.Errorf("expected first name Jane, got %v", view.FirstName)
	}
	if view.DocumentNumber == nil || *view.DocumentNumber != "P987654321" {
		t.Errorf("expected doc number P987654321, got %v", view.DocumentNumber)
	}
	if !view.IsVerified {
		t.Error("expected is_verified to be true")
	}
}

func TestTemplateViewMapping(t *testing.T) {
	now := time.Now()
	tmplID := uuid.New()

	tmpl := &ent.HostedSessionTemplate{
		ID:          tmplID,
		Name:        "Tenant KYC",
		Provider:    "didit",
		TemplateID:  "wf-didit-123",
		Description: "Standard KYC flow",
		IsDefault:   true,
		Active:      true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	view := toTemplateView(tmpl)
	if view == nil {
		t.Fatal("expected non-nil view")
	}

	if view.ID != tmplID {
		t.Errorf("expected template id %s, got %s", tmplID, view.ID)
	}
	if view.TemplateID != "wf-didit-123" {
		t.Errorf("expected template_id wf-didit-123, got %s", view.TemplateID)
	}
	if !view.IsDefault {
		t.Error("expected is_default to be true")
	}
}

func TestInitiateSessionValidation(t *testing.T) {
	reg := NewRegistry("didit", &mockProvider{id: "didit", name: "Didit"})
	svc := NewService(nil, reg)

	// Missing UserID
	_, err := svc.CreateHostedSession(context.Background(), InitiateSessionRequest{})
	if err == nil {
		t.Fatal("expected error for empty user_id, got nil")
	}
	if err.Error() != "user_id is required" {
		t.Errorf("unexpected error: %v", err)
	}
}
