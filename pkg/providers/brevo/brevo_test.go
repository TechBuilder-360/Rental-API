package brevo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/TechBuilder-360/Rental-API/pkg/domain/email"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestSendEmailSuccess(t *testing.T) {
	mockTransport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != emailEndpoint {
			t.Errorf("expected path %s, got %s", emailEndpoint, r.URL.Path)
		}
		if r.Header.Get("api-key") != "test-api-key" {
			t.Errorf("expected api-key 'test-api-key', got '%s'", r.Header.Get("api-key"))
		}
		if r.Header.Get("content-type") != "application/json" {
			t.Errorf("expected application/json content-type")
		}

		var req SendEmailRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if len(req.To) != 1 || req.To[0].Email != "tenant@example.com" {
			t.Errorf("unexpected recipient: %+v", req.To)
		}
		if req.Subject != "Verification Code" {
			t.Errorf("unexpected subject: %s", req.Subject)
		}

		respBody, _ := json.Marshal(SendEmailResponse{MessageID: "<msg-123@smtp-relay.brevo.com>"})
		return &http.Response{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(bytes.NewReader(respBody)),
			Header:     make(http.Header),
		}, nil
	})

	httpClient := &http.Client{Transport: mockTransport}
	client, err := NewClient("test-api-key", Contact{Email: "no-reply@rental.com", Name: "Rental App"}, WithHTTPClient(httpClient))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	resp, err := client.SendEmail(context.Background(), SendEmailRequest{
		To:          []Contact{{Email: "tenant@example.com", Name: "Tenant"}},
		Subject:     "Verification Code",
		HTMLContent: "<p>Your code is 123456</p>",
		TextContent: "Your code is 123456",
	})
	if err != nil {
		t.Fatalf("SendEmail failed: %v", err)
	}

	if resp.MessageID != "<msg-123@smtp-relay.brevo.com>" {
		t.Errorf("unexpected message ID: %s", resp.MessageID)
	}
}

func TestSendEmailError(t *testing.T) {
	mockTransport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		errResp, _ := json.Marshal(APIError{
			Code:    "invalid_parameter",
			Message: "invalid email address",
		})
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(bytes.NewReader(errResp)),
			Header:     make(http.Header),
		}, nil
	})

	httpClient := &http.Client{Transport: mockTransport}
	client, err := NewClient("test-api-key", Contact{Email: "no-reply@rental.com"}, WithHTTPClient(httpClient))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.SendEmail(context.Background(), SendEmailRequest{
		To:          []Contact{{Email: "invalid-email"}},
		Subject:     "Test",
		TextContent: "Hello",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.StatusCode)
	}
	if apiErr.Code != "invalid_parameter" {
		t.Errorf("expected code invalid_parameter, got %s", apiErr.Code)
	}
}

func TestClientValidation(t *testing.T) {
	_, err := NewClient("", Contact{})
	if err == nil {
		t.Error("expected error for empty API key")
	}

	client, _ := NewClient("key", Contact{})
	_, err = client.SendEmail(context.Background(), SendEmailRequest{})
	if err == nil {
		t.Error("expected error for empty recipients")
	}
}

func TestSendViaEmailProviderInterface(t *testing.T) {
	mockTransport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		respBody, _ := json.Marshal(SendEmailResponse{MessageID: "<msg-456@smtp-relay.brevo.com>"})
		return &http.Response{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(bytes.NewReader(respBody)),
			Header:     make(http.Header),
		}, nil
	})

	httpClient := &http.Client{Transport: mockTransport}
	client, err := NewClient("test-api-key", Contact{Email: "no-reply@rental.com", Name: "Rental App"}, WithHTTPClient(httpClient))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	var emailProv email.Provider = client
	if emailProv.ID() != "brevo" {
		t.Errorf("expected ID 'brevo', got '%s'", emailProv.ID())
	}
	if emailProv.Name() != "Brevo" {
		t.Errorf("expected Name 'Brevo', got '%s'", emailProv.Name())
	}
	if emailProv.Slug() != "brevo" {
		t.Errorf("expected Slug 'brevo', got '%s'", emailProv.Slug())
	}

	err = emailProv.Send(context.Background(), email.Message{
		To:       "user@example.com",
		Subject:  "Hello from Provider",
		HTMLBody: "<p>Hello</p>",
		TextBody: "Hello",
	})
	if err != nil {
		t.Fatalf("Send via Provider interface failed: %v", err)
	}
}
