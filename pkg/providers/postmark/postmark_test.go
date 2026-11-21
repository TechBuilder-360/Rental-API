package postmark

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/TechBuilder-360/Rental-API/pkg/domain/email"
	"github.com/TechBuilder-360/Rental-API/pkg/provider"
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
		if r.Header.Get("X-Postmark-Server-Token") != "test-server-token" {
			t.Errorf("expected X-Postmark-Server-Token 'test-server-token', got '%s'", r.Header.Get("X-Postmark-Server-Token"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json Content-Type, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("expected application/json Accept header, got %s", r.Header.Get("Accept"))
		}

		var req SendEmailRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if req.To != "tenant@example.com" {
			t.Errorf("unexpected recipient: %s", req.To)
		}
		if req.From != "no-reply@rental.com" {
			t.Errorf("unexpected sender: %s", req.From)
		}
		if req.Subject != "Verification Code" {
			t.Errorf("unexpected subject: %s", req.Subject)
		}
		if req.TextBody != "Your OTP is 123456" {
			t.Errorf("unexpected text body: %s", req.TextBody)
		}
		if req.MessageStream != "outbound" {
			t.Errorf("unexpected message stream: %s", req.MessageStream)
		}

		respBody, _ := json.Marshal(SendEmailResponse{
			To:          "tenant@example.com",
			SubmittedAt: time.Now(),
			MessageID:   "b7bc954a-e233-4136-9e63-7186d6342898",
			ErrorCode:   0,
			Message:     "OK",
		})

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(respBody)),
			Header:     make(http.Header),
		}, nil
	})

	httpClient := &http.Client{Transport: mockTransport}
	client, err := NewClient("test-server-token", "no-reply@rental.com", WithHTTPClient(httpClient))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	resp, err := client.SendEmail(context.Background(), SendEmailRequest{
		To:       "tenant@example.com",
		Subject:  "Verification Code",
		TextBody: "Your OTP is 123456",
	})
	if err != nil {
		t.Fatalf("SendEmail failed: %v", err)
	}

	if resp.MessageID != "b7bc954a-e233-4136-9e63-7186d6342898" {
		t.Errorf("unexpected message ID: %s", resp.MessageID)
	}
	if resp.ErrorCode != 0 {
		t.Errorf("expected ErrorCode 0, got %d", resp.ErrorCode)
	}
}

func TestSendEmailAPIError(t *testing.T) {
	mockTransport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		errResp, _ := json.Marshal(SendEmailResponse{
			ErrorCode: 300,
			Message:   "Zero recipients specified",
		})
		return &http.Response{
			StatusCode: http.StatusUnprocessableEntity,
			Body:       io.NopCloser(bytes.NewReader(errResp)),
			Header:     make(http.Header),
		}, nil
	})

	httpClient := &http.Client{Transport: mockTransport}
	client, err := NewClient("test-server-token", "no-reply@rental.com", WithHTTPClient(httpClient))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.SendEmail(context.Background(), SendEmailRequest{
		To:       "invalid-recipient",
		Subject:  "Test",
		TextBody: "Hello",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, apiErr.StatusCode)
	}
	if apiErr.ErrorCode != 300 {
		t.Errorf("expected error code 300, got %d", apiErr.ErrorCode)
	}
	if apiErr.Message != "Zero recipients specified" {
		t.Errorf("unexpected error message: %s", apiErr.Message)
	}
}

func TestSendEmailValidation(t *testing.T) {
	client, err := NewClient("test-server-token", "no-reply@rental.com")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	tests := []struct {
		name    string
		req     SendEmailRequest
		wantErr string
	}{
		{
			name:    "missing to",
			req:     SendEmailRequest{Subject: "Test", TextBody: "Hello"},
			wantErr: "recipient email (To) is required",
		},
		{
			name:    "missing subject",
			req:     SendEmailRequest{To: "a@b.com", TextBody: "Hello"},
			wantErr: "subject is required",
		},
		{
			name:    "missing content",
			req:     SendEmailRequest{To: "a@b.com", Subject: "Test"},
			wantErr: "email content (HtmlBody or TextBody) is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.SendEmail(context.Background(), tt.req)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if err.Error() != tt.wantErr {
				t.Errorf("expected %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestNewClientValidation(t *testing.T) {
	_, err := NewClient("", "no-reply@rental.com")
	if err == nil {
		t.Fatal("expected error for empty server token, got nil")
	}
	if err.Error() != "postmark server token is required" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestProviderInterface(t *testing.T) {
	client, err := NewClient("token", "sender@domain.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var p provider.Provider = client
	if p.DisplayName() != "Postmark" {
		t.Errorf("expected DisplayName 'Postmark', got '%s'", p.DisplayName())
	}
	if p.Slug() != "postmark" {
		t.Errorf("expected Slug 'postmark', got '%s'", p.Slug())
	}

	var ep email.Provider = client
	if ep.ID() != "postmark" {
		t.Errorf("expected ID 'postmark', got '%s'", ep.ID())
	}
	if ep.Slug() != "postmark" {
		t.Errorf("expected Slug 'postmark', got '%s'", ep.Slug())
	}
}

func TestSendViaEmailProvider(t *testing.T) {
	var capturedReq SendEmailRequest
	mockTransport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		_ = json.NewDecoder(r.Body).Decode(&capturedReq)

		respBody, _ := json.Marshal(SendEmailResponse{
			To:        capturedReq.To,
			MessageID: "pm-msg-999",
			ErrorCode: 0,
			Message:   "OK",
		})
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(respBody)),
			Header:     make(http.Header),
		}, nil
	})

	httpClient := &http.Client{Transport: mockTransport}
	client, err := NewClient("token-xyz", "support@rental.com",
		WithHTTPClient(httpClient),
		WithMessageStream("broadcast"),
		WithBaseURL("https://api.postmarkapp.com/"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	attachmentContent := []byte("hello attachment")
	msg := email.Message{
		To:       "user@example.com",
		Subject:  "Invoice",
		HTMLBody: "<h1>Your Invoice</h1>",
		TextBody: "Your Invoice",
		Tag:      "billing",
		ReplyTo:  "billing@rental.com",
		Attachments: []email.Attachment{
			{
				Name:        "invoice.pdf",
				ContentType: "application/pdf",
				Content:     attachmentContent,
			},
		},
	}

	if err := client.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if capturedReq.To != "user@example.com" {
		t.Errorf("expected To 'user@example.com', got '%s'", capturedReq.To)
	}
	if capturedReq.From != "support@rental.com" {
		t.Errorf("expected From 'support@rental.com', got '%s'", capturedReq.From)
	}
	if capturedReq.Tag != "billing" {
		t.Errorf("expected Tag 'billing', got '%s'", capturedReq.Tag)
	}
	if capturedReq.ReplyTo != "billing@rental.com" {
		t.Errorf("expected ReplyTo 'billing@rental.com', got '%s'", capturedReq.ReplyTo)
	}
	if capturedReq.MessageStream != "broadcast" {
		t.Errorf("expected MessageStream 'broadcast', got '%s'", capturedReq.MessageStream)
	}
	if len(capturedReq.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(capturedReq.Attachments))
	}
	expectedB64 := base64.StdEncoding.EncodeToString(attachmentContent)
	if capturedReq.Attachments[0].Content != expectedB64 {
		t.Errorf("expected base64 content %s, got %s", expectedB64, capturedReq.Attachments[0].Content)
	}
}
