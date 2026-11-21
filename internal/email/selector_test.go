package email

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
)

type mockSender struct {
	sent []Message
}

func (m *mockSender) Send(ctx context.Context, msg Message) error {
	m.sent = append(m.sent, msg)
	return nil
}

func TestDynamicSenderFallback(t *testing.T) {
	brevoMock := &mockSender{}
	postmarkMock := &mockSender{}

	senders := map[string]Sender{
		"brevo":    brevoMock,
		"postmark": postmarkMock,
	}

	// Without DB connection, falls back to default provider
	dynamic := NewDynamicSender(nil, senders, "brevo", zerolog.Nop())

	resolved, slug, err := dynamic.ResolveProvider(context.Background())
	if err != nil {
		t.Fatalf("ResolveProvider failed: %v", err)
	}

	if slug != "brevo" {
		t.Errorf("expected slug brevo, got %s", slug)
	}
	if resolved != brevoMock {
		t.Error("expected brevoMock sender")
	}

	msg := Message{
		To:       "user@example.com",
		Subject:  "Test",
		TextBody: "Hello",
	}

	if err := dynamic.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if len(brevoMock.sent) != 1 {
		t.Errorf("expected 1 message sent via brevo, got %d", len(brevoMock.sent))
	}
}

func TestDynamicSenderNotFound(t *testing.T) {
	senders := map[string]Sender{}
	dynamic := NewDynamicSender(nil, senders, "non-existent", zerolog.Nop())

	_, _, err := dynamic.ResolveProvider(context.Background())
	if err == nil {
		t.Fatal("expected error for non-existent provider, got nil")
	}
}
