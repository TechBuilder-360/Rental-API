package email

import (
	"context"
	"fmt"

	"github.com/TechBuilder-360/Rental-API/pkg/providers/brevo"
)

// BrevoSender adapts the brevo.Client to the email.Sender interface.
type BrevoSender struct {
	client *brevo.Client
	from   string
	name   string
}

func NewBrevoSender(apiKey, fromEmail, fromName string) (*BrevoSender, error) {
	client, err := brevo.NewClient(apiKey, brevo.Contact{
		Email: fromEmail,
		Name:  fromName,
	})
	if err != nil {
		return nil, fmt.Errorf("init brevo client: %w", err)
	}
	return &BrevoSender{
		client: client,
		from:   fromEmail,
		name:   fromName,
	}, nil
}

func (s *BrevoSender) Send(ctx context.Context, msg Message) error {
	req := brevo.SendEmailRequest{
		Sender: brevo.Contact{
			Email: s.from,
			Name:  s.name,
		},
		To: []brevo.Contact{
			{Email: msg.To},
		},
		Subject:     msg.Subject,
		HTMLContent: msg.HTMLBody,
		TextContent: msg.TextBody,
	}
	if msg.Tag != "" {
		req.Tags = []string{msg.Tag}
	}

	_, err := s.client.SendEmail(ctx, req)
	if err != nil {
		if apiErr, ok := err.(*brevo.APIError); ok {
			if apiErr.StatusCode < 500 && apiErr.StatusCode != 429 {
				return &PermanentError{Err: err}
			}
		}
		return err
	}
	return nil
}
