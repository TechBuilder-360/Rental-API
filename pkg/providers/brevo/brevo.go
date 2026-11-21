package brevo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/TechBuilder-360/Rental-API/pkg/domain/email"
	"github.com/TechBuilder-360/Rental-API/pkg/provider"
)

const (
	DefaultBaseURL = "https://api.brevo.com"
	emailEndpoint  = "/v3/smtp/email"
)

// Contact represents an email recipient or sender.
type Contact struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// Attachment represents an email attachment for Brevo.
type Attachment struct {
	Name    string `json:"name"`
	Content []byte `json:"content,omitempty"`
	URL     string `json:"url,omitempty"`
}

// SendEmailRequest contains the payload for sending a transactional email.
type SendEmailRequest struct {
	Sender      Contact           `json:"sender"`
	To          []Contact         `json:"to"`
	Subject     string            `json:"subject"`
	HTMLContent string            `json:"htmlContent,omitempty"`
	TextContent string            `json:"textContent,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Params      map[string]any    `json:"params,omitempty"`
	TemplateID  *int64            `json:"templateId,omitempty"`
	ReplyTo     *Contact          `json:"replyTo,omitempty"`
	Attachment  []Attachment      `json:"attachment,omitempty"`
}

// SendEmailResponse is returned on successful email submission.
type SendEmailResponse struct {
	MessageID string `json:"messageId"`
}

// APIError represents an error returned by the Brevo API.
type APIError struct {
	StatusCode int    `json:"-"`
	Code       string `json:"code"`
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("brevo api error (status %d, code %s): %s", e.StatusCode, e.Code, e.Message)
}

// Client interacts with the Brevo v3 API using resty.
type Client struct {
	apiKey        string
	defaultSender Contact
	baseURL       string
	client        *resty.Client
}

// Option configures the Brevo Client.
type Option func(*Client)

// WithHTTPClient overrides the default HTTP client with a standard http.Client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		if client != nil {
			c.client = resty.NewWithClient(client)
		}
	}
}

// WithRestyClient overrides the default Resty client.
func WithRestyClient(client *resty.Client) Option {
	return func(c *Client) {
		if client != nil {
			c.client = client
		}
	}
}

// WithBaseURL overrides the default Brevo API base URL.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		if url != "" {
			c.baseURL = strings.TrimRight(url, "/")
		}
	}
}

// NewClient initializes a new Brevo API client.
func NewClient(apiKey string, defaultSender Contact, opts ...Option) (*Client, error) {
	if apiKey == "" {
		return nil, errors.New("brevo API key is required")
	}

	c := &Client{
		apiKey:        apiKey,
		defaultSender: defaultSender,
		baseURL:       DefaultBaseURL,
		client:        resty.New().SetTimeout(15 * time.Second),
	}

	for _, opt := range opts {
		opt(c)
	}

	c.client.SetBaseURL(c.baseURL).
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/json").
		SetHeader("api-key", c.apiKey)

	return c, nil
}

// RestyClient returns the underlying resty.Client instance.
func (c *Client) RestyClient() *resty.Client {
	return c.client
}

// SendEmail dispatches a transactional email through Brevo.
func (c *Client) SendEmail(ctx context.Context, req SendEmailRequest) (*SendEmailResponse, error) {
	if len(req.To) == 0 {
		return nil, errors.New("at least one recipient is required")
	}
	if req.Sender.Email == "" {
		req.Sender = c.defaultSender
	}
	if req.Sender.Email == "" {
		return nil, errors.New("sender email is required")
	}
	if req.Subject == "" && req.TemplateID == nil {
		return nil, errors.New("subject is required when not using a template")
	}
	if req.HTMLContent == "" && req.TextContent == "" && req.TemplateID == nil {
		return nil, errors.New("email content (html, text, or templateId) is required")
	}

	resp, err := c.client.R().
		SetContext(ctx).
		SetBody(req).
		Post(emailEndpoint)
	if err != nil {
		return nil, fmt.Errorf("execute brevo request: %w", err)
	}

	if resp.IsSuccess() {
		var successResp SendEmailResponse
		if err := json.Unmarshal(resp.Body(), &successResp); err != nil {
			return nil, fmt.Errorf("decode brevo success response: %w", err)
		}
		return &successResp, nil
	}

	var apiErr APIError
	apiErr.StatusCode = resp.StatusCode()
	if err := json.Unmarshal(resp.Body(), &apiErr); err != nil {
		apiErr.Message = resp.String()
	}

	return nil, &apiErr
}

// Compile-time checks ensuring Client implements provider.Provider and email.Provider.
var (
	_ provider.Provider = (*Client)(nil)
	_ email.Provider    = (*Client)(nil)
)

// ID returns the unique identifier for Brevo.
func (c *Client) ID() string {
	return "brevo"
}

// Name returns the human-readable display name for Brevo.
func (c *Client) Name() string {
	return "Brevo"
}

// DisplayName returns the human-readable display name for Brevo (implements provider.Impl).
func (c *Client) DisplayName() string {
	return c.Name()
}

// Slug returns the provider name in lower case ("brevo").
func (c *Client) Slug() string {
	return strings.ToLower(c.Name())
}

// Send implements the email.Provider interface.
func (c *Client) Send(ctx context.Context, msg email.Message) error {
	req := SendEmailRequest{
		Sender: c.defaultSender,
		To: []Contact{
			{Email: msg.To},
		},
		Subject:     msg.Subject,
		HTMLContent: msg.HTMLBody,
		TextContent: msg.TextBody,
	}
	if msg.Tag != "" {
		req.Tags = []string{msg.Tag}
	}
	if msg.ReplyTo != "" {
		req.ReplyTo = &Contact{Email: msg.ReplyTo}
	}
	for _, att := range msg.Attachments {
		req.Attachment = append(req.Attachment, Attachment{
			Name:    att.Name,
			Content: att.Content,
		})
	}

	_, err := c.SendEmail(ctx, req)
	return err
}
