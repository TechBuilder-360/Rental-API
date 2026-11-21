package didit

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/TechBuilder-360/Rental-API/pkg/domain/verification"
	"github.com/TechBuilder-360/Rental-API/pkg/provider"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCreateSessionSuccess(t *testing.T) {
	mockTransport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v3/session/" {
			t.Errorf("expected /v3/session/, got %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "didit-test-key" {
			t.Errorf("expected didit-test-key, got %s", r.Header.Get("x-api-key"))
		}

		var req CreateSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if req.WorkflowID != "wf-123" {
			t.Errorf("expected wf-123, got %s", req.WorkflowID)
		}
		if req.VendorData != "user-abc" {
			t.Errorf("expected user-abc, got %s", req.VendorData)
		}

		respBody, _ := json.Marshal(CreateSessionResponse{
			SessionID:       "sess-999",
			SessionKind:     "user",
			VerificationURL: "https://verify.didit.me/session/sess-999",
			Status:          StatusCreated,
		})
		return &http.Response{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(bytes.NewReader(respBody)),
			Header:     make(http.Header),
		}, nil
	})

	httpClient := &http.Client{Transport: mockTransport}
	client, err := NewClient("didit-test-key", WithHTTPClient(httpClient), WithDefaultWorkflow("wf-123"))
	if err != nil {
		t.Fatalf("failed to create didit client: %v", err)
	}

	resp, err := client.CreateSession(context.Background(), CreateSessionRequest{
		VendorData:  "user-abc",
		CallbackURL: "https://rental.local/verified",
	})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if resp.SessionID != "sess-999" {
		t.Errorf("expected sess-999, got %s", resp.SessionID)
	}
	if resp.VerificationURL != "https://verify.didit.me/session/sess-999" {
		t.Errorf("unexpected verification url: %s", resp.VerificationURL)
	}
}

func TestGetDecision(t *testing.T) {
	mockTransport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v3/session/sess-123/decision/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		respBody, _ := json.Marshal(DecisionResponse{
			SessionID:  "sess-123",
			Status:     StatusApproved,
			VendorData: "user-abc",
			Decision: &DecisionDetail{
				Decision:  "Approved",
				RiskScore: 0.02,
			},
		})
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(respBody)),
			Header:     make(http.Header),
		}, nil
	})

	httpClient := &http.Client{Transport: mockTransport}
	client, err := NewClient("didit-test-key", WithHTTPClient(httpClient))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	decision, err := client.GetDecision(context.Background(), "sess-123")
	if err != nil {
		t.Fatalf("GetDecision failed: %v", err)
	}

	if decision.Status != StatusApproved {
		t.Errorf("expected Approved, got %s", decision.Status)
	}
	if decision.Decision == nil || decision.Decision.RiskScore != 0.02 {
		t.Errorf("unexpected decision detail: %+v", decision.Decision)
	}
}

func TestWebhookSignatureVerification(t *testing.T) {
	secret := "webhook-secret-key"
	client, err := NewClient("api-key", WithWebhookSecret(secret))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	payload := []byte(`{"event_type":"session.completed","session_id":"sess-123","status":"Approved"}`)
	now := time.Now().Unix()

	// Compute HMAC
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	validSignature := hex.EncodeToString(mac.Sum(nil))

	// 1. Valid webhook
	event, err := client.ParseWebhook(payload, validSignature, now)
	if err != nil {
		t.Fatalf("expected webhook verification to succeed: %v", err)
	}
	if event.SessionID != "sess-123" {
		t.Errorf("expected session_id sess-123, got %s", event.SessionID)
	}
	if event.Status != StatusApproved {
		t.Errorf("expected status Approved, got %s", event.Status)
	}

	// 2. Tampered payload
	tampered := []byte(`{"event_type":"session.completed","session_id":"sess-999"}`)
	_, err = client.ParseWebhook(tampered, validSignature, now)
	if err == nil {
		t.Error("expected signature error for tampered payload")
	}

	// 3. Expired timestamp (replay attack)
	oldTimestamp := now - 600 // 10 minutes ago
	_, err = client.ParseWebhook(payload, validSignature, oldTimestamp)
	if err == nil {
		t.Error("expected error for stale timestamp")
	}
}

func TestDiditImplementsVerificationProvider(t *testing.T) {
	client, err := NewClient("test-key")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	var p provider.Provider = client
	if p.DisplayName() != "Didit" {
		t.Errorf("expected DisplayName Didit, got %s", p.DisplayName())
	}
	if p.Slug() != "didit" {
		t.Errorf("expected Slug didit, got %s", p.Slug())
	}

	var vp verification.Provider = client
	if vp.ID() != "didit" {
		t.Errorf("expected ID didit, got %s", vp.ID())
	}
	if vp.Name() != "Didit" {
		t.Errorf("expected Name Didit, got %s", vp.Name())
	}
	if vp.Slug() != "didit" {
		t.Errorf("expected Slug didit, got %s", vp.Slug())
	}
}
