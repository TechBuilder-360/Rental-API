package postmark

import (
	"context"
	"encoding/base64"
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
	// DefaultBaseURL is the official Postmark API endpoint.
	DefaultBaseURL = "https://api.postmarkapp.com"
	emailEndpoint  = "/email"
)

// Header represents a custom header in a Postmark email.
type Header struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

// Attachment represents an attachment in a Postmark email.
type Attachment struct {
	Name        string `json:"Name"`
	Content     string `json:"Content"` // Base64 encoded content
	ContentType string `json:"ContentType"`
	ContentID   string `json:"ContentID,omitempty"`
}

// SendEmailRequest contains the payload for sending a single email via Postmark.
type SendEmailRequest struct {
	From          string            `json:"From"`
	To            string            `json:"To"`
	Cc            string            `json:"Cc,omitempty"`
	Bcc           string            `json:"Bcc,omitempty"`
	Subject       string            `json:"Subject"`
	Tag           string            `json:"Tag,omitempty"`
	HTMLBody      string            `json:"HtmlBody,omitempty"`
	TextBody      string            `json:"TextBody,omitempty"`
	ReplyTo       string            `json:"ReplyTo,omitempty"`
	Headers       []Header          `json:"Headers,omitempty"`
	TrackOpens    *bool             `json:"TrackOpens,omitempty"`
	TrackLinks    string            `json:"TrackLinks,omitempty"`
	Metadata      map[string]string `json:"Metadata,omitempty"`
	Attachments   []Attachment      `json:"Attachments,omitempty"`
	MessageStream string            `json:"MessageStream,omitempty"`
}

// SendEmailResponse contains the response returned by Postmark on message submission.
type SendEmailResponse struct {
	To          string    `json:"To"`
	SubmittedAt time.Time `json:"SubmittedAt"`
	MessageID   string    `json:"MessageID"`
	ErrorCode   int       `json:"ErrorCode"`
	Message     string    `json:"Message"`
}

// APIError represents an error returned by the Postmark API.
type APIError struct {
	StatusCode int    `json:"-"`
	ErrorCode  int    `json:"ErrorCode"`
	Message    string `json:"Message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("postmark api error (status %d, error code %d): %s", e.StatusCode, e.ErrorCode, e.Message)
}

// Client interacts with the Postmark API using resty.
type Client struct {
	serverToken   string
	defaultSender string
	messageStream string
	baseURL       string
	client        *resty.Client
}

// Option configures the Postmark Client.
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

// WithBaseURL overrides the default Postmark API base URL.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		if url != "" {
			c.baseURL = strings.TrimRight(url, "/")
		}
	}
}

// WithMessageStream overrides the default outbound message stream.
func WithMessageStream(stream string) Option {
	return func(c *Client) {
		if stream != "" {
			c.messageStream = stream
		}
	}
}

// NewClient initializes a new Postmark API client.
func NewClient(serverToken string, defaultSender string, opts ...Option) (*Client, error) {
	if serverToken == "" {
		return nil, errors.New("postmark server token is required")
	}

	c := &Client{
		serverToken:   serverToken,
		defaultSender: defaultSender,
		messageStream: "outbound",
		baseURL:       DefaultBaseURL,
		client:        resty.New().SetTimeout(15 * time.Second),
	}

	for _, opt := range opts {
		opt(c)
	}

	c.client.SetBaseURL(c.baseURL).
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/json").
		SetHeader("X-Postmark-Server-Token", c.serverToken)

	return c, nil
}

// RestyClient returns the underlying resty.Client instance.
func (c *Client) RestyClient() *resty.Client {
	return c.client
}

// SendEmail delivers an email via Postmark API.
func (c *Client) SendEmail(ctx context.Context, req SendEmailRequest) (*SendEmailResponse, error) {
	if req.To == "" {
		return nil, errors.New("recipient email (To) is required")
	}
	if req.From == "" {
		req.From = c.defaultSender
	}
	if req.From == "" {
		return nil, errors.New("sender email (From) is required")
	}
	if req.Subject == "" {
		return nil, errors.New("subject is required")
	}
	if req.HTMLBody == "" && req.TextBody == "" {
		return nil, errors.New("email content (HtmlBody or TextBody) is required")
	}
	if req.MessageStream == "" && c.messageStream != "" {
		req.MessageStream = c.messageStream
	}

	resp, err := c.client.R().
		SetContext(ctx).
		SetBody(req).
		Post(emailEndpoint)
	if err != nil {
		return nil, fmt.Errorf("execute postmark request: %w", err)
	}

	var postmarkResp SendEmailResponse
	_ = json.Unmarshal(resp.Body(), &postmarkResp)

	if resp.IsSuccess() && postmarkResp.ErrorCode == 0 {
		return &postmarkResp, nil
	}

	apiErr := &APIError{
		StatusCode: resp.StatusCode(),
		ErrorCode:  postmarkResp.ErrorCode,
		Message:    postmarkResp.Message,
	}
	if apiErr.Message == "" {
		apiErr.Message = resp.String()
	}

	return nil, apiErr
}

// Compile-time checks ensuring Client implements provider.Provider and email.Provider.
var (
	_ provider.Provider = (*Client)(nil)
	_ email.Provider    = (*Client)(nil)
)

// ID returns the unique identifier for Postmark ("postmark").
func (c *Client) ID() string {
	return "postmark"
}

// Name returns the human-readable display name for Postmark ("Postmark").
func (c *Client) Name() string {
	return "Postmark"
}

// DisplayName returns the human-readable display name for Postmark (implements provider.Impl).
func (c *Client) DisplayName() string {
	return c.Name()
}

// Slug returns the provider name in lower case ("postmark").
func (c *Client) Slug() string {
	return strings.ToLower(c.Name())
}

// Send delivers an email message conforming to the email.Provider interface.
func (c *Client) Send(ctx context.Context, msg email.Message) error {
	req := SendEmailRequest{
		From:          c.defaultSender,
		To:            msg.To,
		Subject:       msg.Subject,
		HTMLBody:      msg.HTMLBody,
		TextBody:      msg.TextBody,
		Tag:           msg.Tag,
		ReplyTo:       msg.ReplyTo,
		MessageStream: c.messageStream,
	}

	if len(msg.Attachments) > 0 {
		req.Attachments = make([]Attachment, len(msg.Attachments))
		for i, att := range msg.Attachments {
			req.Attachments[i] = Attachment{
				Name:        att.Name,
				ContentType: att.ContentType,
				Content:     base64.StdEncoding.EncodeToString(att.Content),
			}
		}
	}

	_, err := c.SendEmail(ctx, req)
	return err
}
