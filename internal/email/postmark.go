package email

import (
	"context"
	"fmt"
	"net/http"

	"github.com/TechBuilder-360/Rental-API/pkg/providers/postmark"
)

// PostmarkSender adapts the postmark.Client to the email.Sender interface.
type PostmarkSender struct {
	client *postmark.Client
	from   string
	stream string
}

func NewPostmarkSender(serverToken, from, messageStream string) (*PostmarkSender, error) {
	var opts []postmark.Option
	if messageStream != "" {
		opts = append(opts, postmark.WithMessageStream(messageStream))
	}

	client, err := postmark.NewClient(serverToken, from, opts...)
	if err != nil {
		return nil, fmt.Errorf("init postmark client: %w", err)
	}

	return &PostmarkSender{
		client: client,
		from:   from,
		stream: messageStream,
	}, nil
}

func (s *PostmarkSender) Send(ctx context.Context, msg Message) error {
	req := postmark.SendEmailRequest{
		From:          s.from,
		To:            msg.To,
		Subject:       msg.Subject,
		HTMLBody:      msg.HTMLBody,
		TextBody:      msg.TextBody,
		Tag:           msg.Tag,
		MessageStream: s.stream,
	}

	_, err := s.client.SendEmail(ctx, req)
	if err != nil {
		if apiErr, ok := err.(*postmark.APIError); ok {
			// 429 and 5xx are transient; other 4xx (401 bad token, 422 invalid request) are permanent errors
			if apiErr.StatusCode == http.StatusTooManyRequests || apiErr.StatusCode >= 500 {
				return err
			}
			return &PermanentError{Err: err}
		}
		return err
	}
	return nil
}
