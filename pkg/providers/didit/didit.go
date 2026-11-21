package didit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/TechBuilder-360/Rental-API/pkg/provider"
	"github.com/TechBuilder-360/Rental-API/pkg/domain/verification"
)

const (
	DefaultBaseURL          = "https://verification.didit.me"
	DefaultWebhookTolerance = 5 * time.Minute
)

// Type aliases linking Didit types directly to the verification.Provider interface.
type (
	SessionStatus         = verification.SessionStatus
	CreateSessionRequest  = verification.CreateSessionRequest
	CreateSessionResponse = verification.CreateSessionResponse
	DecisionDetail        = verification.DecisionDetail
	DecisionResponse      = verification.DecisionResponse
	WebhookEvent          = verification.WebhookEvent
)

const (
	StatusCreated    = verification.StatusCreated
	StatusInProgress = verification.StatusInProgress
	StatusApproved   = verification.StatusApproved
	StatusDeclined   = verification.StatusDeclined
	StatusReview     = verification.StatusReview
	StatusExpired    = verification.StatusExpired
	StatusAbandoned  = verification.StatusAbandoned
)

// APIError represents an error returned by the Didit API.
type APIError struct {
	StatusCode int    `json:"-"`
	Detail     string `json:"detail,omitempty"`
	Message    string `json:"message,omitempty"`
}

func (e *APIError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = e.Detail
	}
	return fmt.Sprintf("didit api error (status %d): %s", e.StatusCode, msg)
}

// Client interacts with the Didit Verification API using resty.
type Client struct {
	apiKey           string
	defaultWorkflow  string
	webhookSecret    string
	baseURL          string
	webhookTolerance time.Duration
	client           *resty.Client
}

// Option configures the Didit Client.
type Option func(*Client)

// WithBaseURL overrides the default Didit API base URL.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		if url != "" {
			c.baseURL = strings.TrimRight(url, "/")
		}
	}
}

// WithHTTPClient overrides the default HTTP transport with a standard http.Client.
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

// WithWebhookSecret configures the secret for verifying webhook signatures.
func WithWebhookSecret(secret string) Option {
	return func(c *Client) {
		c.webhookSecret = secret
	}
}

// WithDefaultWorkflow sets a default workflow ID used when none is specified.
func WithDefaultWorkflow(workflowID string) Option {
	return func(c *Client) {
		c.defaultWorkflow = workflowID
	}
}

// WithWebhookTolerance sets the allowed clock skew when verifying webhook timestamps.
func WithWebhookTolerance(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			c.webhookTolerance = d
		}
	}
}

// NewClient initializes a Didit verification client.
func NewClient(apiKey string, opts ...Option) (*Client, error) {
	if apiKey == "" {
		return nil, errors.New("didit API key is required")
	}

	c := &Client{
		apiKey:           apiKey,
		baseURL:          DefaultBaseURL,
		webhookTolerance: DefaultWebhookTolerance,
		client:           resty.New().SetTimeout(15 * time.Second),
	}

	for _, opt := range opts {
		opt(c)
	}

	c.client.SetBaseURL(c.baseURL).
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/json").
		SetHeader("x-api-key", c.apiKey)

	return c, nil
}

// RestyClient returns the underlying resty.Client instance.
func (c *Client) RestyClient() *resty.Client {
	return c.client
}

// CreateSession creates a new hosted verification session for a user.
func (c *Client) CreateSession(ctx context.Context, req CreateSessionRequest) (*CreateSessionResponse, error) {
	if req.WorkflowID == "" {
		req.WorkflowID = c.defaultWorkflow
	}
	if req.WorkflowID == "" {
		return nil, errors.New("workflow_id is required")
	}

	resp, err := c.client.R().
		SetContext(ctx).
		SetBody(req).
		Post("/v3/session/")
	if err != nil {
		return nil, fmt.Errorf("execute didit session request: %w", err)
	}

	if resp.IsSuccess() {
		var successResp CreateSessionResponse
		if err := json.Unmarshal(resp.Body(), &successResp); err != nil {
			return nil, fmt.Errorf("decode didit session response: %w", err)
		}
		return &successResp, nil
	}

	var apiErr APIError
	apiErr.StatusCode = resp.StatusCode()
	if err := json.Unmarshal(resp.Body(), &apiErr); err != nil || (apiErr.Message == "" && apiErr.Detail == "") {
		apiErr.Message = resp.String()
	}

	return nil, &apiErr
}

// GetDecision retrieves the verification decision and status for an existing session.
func (c *Client) GetDecision(ctx context.Context, sessionID string) (*DecisionResponse, error) {
	if sessionID == "" {
		return nil, errors.New("session_id is required")
	}

	path := fmt.Sprintf("/v3/session/%s/decision/", sessionID)
	resp, err := c.client.R().
		SetContext(ctx).
		Get(path)
	if err != nil {
		return nil, fmt.Errorf("execute didit decision request: %w", err)
	}

	if resp.IsSuccess() {
		var decision DecisionResponse
		if err := json.Unmarshal(resp.Body(), &decision); err != nil {
			return nil, fmt.Errorf("decode didit decision response: %w", err)
		}
		return &decision, nil
	}

	var apiErr APIError
	apiErr.StatusCode = resp.StatusCode()
	if err := json.Unmarshal(resp.Body(), &apiErr); err != nil || (apiErr.Message == "" && apiErr.Detail == "") {
		apiErr.Message = resp.String()
	}

	return nil, &apiErr
}

// VerifyWebhookSignature verifies the authenticity of a Didit webhook request.
// It checks timestamp freshness (anti-replay) and validates HMAC-SHA256 signature using constant-time comparison.
func (c *Client) VerifyWebhookSignature(rawBody []byte, signature string, timestamp int64) error {
	if c.webhookSecret == "" {
		return errors.New("didit webhook secret is not configured")
	}
	if signature == "" {
		return errors.New("missing webhook signature")
	}

	// Validate timestamp freshness
	if timestamp > 0 {
		now := time.Now().Unix()
		diff := math.Abs(float64(now - timestamp))
		if diff > c.webhookTolerance.Seconds() {
			return fmt.Errorf("webhook timestamp outside tolerance window (diff: %.0fs)", diff)
		}
	}

	// Compute expected HMAC-SHA256 signature
	mac := hmac.New(sha256.New, []byte(c.webhookSecret))
	mac.Write(rawBody)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	if subtle.ConstantTimeCompare([]byte(expectedSignature), []byte(signature)) != 1 {
		return errors.New("invalid webhook signature")
	}

	return nil
}

// ParseWebhook verifies and decodes a Didit webhook notification.
func (c *Client) ParseWebhook(rawBody []byte, signature string, timestamp int64) (*WebhookEvent, error) {
	if err := c.VerifyWebhookSignature(rawBody, signature, timestamp); err != nil {
		return nil, err
	}

	var event WebhookEvent
	if err := json.Unmarshal(rawBody, &event); err != nil {
		return nil, fmt.Errorf("decode didit webhook payload: %w", err)
	}

	return &event, nil
}

// Compile-time checks ensuring Client implements provider.Provider and verification.Provider.
var (
	_ provider.Provider     = (*Client)(nil)
	_ verification.Provider = (*Client)(nil)
)

// ID returns the unique identifier for Didit.
func (c *Client) ID() string {
	return "didit"
}

// Name returns the human-readable display name for Didit.
func (c *Client) Name() string {
	return "Didit"
}

// DisplayName returns the human-readable display name for Didit (implements provider.Impl).
func (c *Client) DisplayName() string {
	return c.Name()
}

// Slug returns the provider name in lower case ("didit").
func (c *Client) Slug() string {
	return strings.ToLower(c.Name())
}
