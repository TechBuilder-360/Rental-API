package verification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/TechBuilder-360/Rental-API/ent"
	"github.com/TechBuilder-360/Rental-API/ent/hostedsession"
	"github.com/TechBuilder-360/Rental-API/ent/hostedsessiontemplate"
	"github.com/TechBuilder-360/Rental-API/ent/user"
	"github.com/TechBuilder-360/Rental-API/ent/useridentity"
)

var (
	ErrSessionNotFound  = errors.New("hosted verification session not found")
	ErrTemplateNotFound = errors.New("hosted session template not found")
	ErrIdentityNotFound = errors.New("user identity not found")
	ErrUserNotFound     = errors.New("user not found")
)

// CreateTemplateRequest contains parameters to register a hosted session template.
type CreateTemplateRequest struct {
	Name        string         `json:"name"`
	Provider    string         `json:"provider"`
	TemplateID  string         `json:"template_id"`
	Description string         `json:"description,omitempty"`
	IsDefault   bool           `json:"is_default,omitempty"`
	Active      bool           `json:"active,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// TemplateView represents a hosted session template managing external workflow IDs.
type TemplateView struct {
	ID          uuid.UUID      `json:"id"`
	Name        string         `json:"name"`
	Provider    string         `json:"provider"`
	TemplateID  string         `json:"template_id"`
	Description string         `json:"description,omitempty"`
	IsDefault   bool           `json:"is_default"`
	Active      bool           `json:"active"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// UserIdentityView represents the extracted user identification details.
type UserIdentityView struct {
	ID                 uuid.UUID      `json:"id"`
	UserID             uuid.UUID      `json:"user_id"`
	HostedSessionID    *uuid.UUID     `json:"hosted_session_id,omitempty"`
	Provider           string         `json:"provider"`
	FirstName          *string        `json:"first_name,omitempty"`
	LastName           *string        `json:"last_name,omitempty"`
	DOB                *time.Time     `json:"dob,omitempty"`
	Gender             *string        `json:"gender,omitempty"`
	Nationality        *string        `json:"nationality,omitempty"`
	DocumentType       *string        `json:"document_type,omitempty"`
	DocumentNumber     *string        `json:"document_number,omitempty"`
	DocumentCountry    *string        `json:"document_country,omitempty"`
	DocumentExpiration *time.Time     `json:"document_expiration,omitempty"`
	Address            *string        `json:"address,omitempty"`
	IsVerified         bool           `json:"is_verified"`
	VerifiedAt         *time.Time     `json:"verified_at,omitempty"`
	RawData            map[string]any `json:"raw_data,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

// InitiateSessionRequest contains parameters to create a new hosted verification session.
type InitiateSessionRequest struct {
	UserID      uuid.UUID         `json:"user_id"`
	CallbackURL string            `json:"callback_url,omitempty"`
	TemplateID  *uuid.UUID        `json:"template_id,omitempty"`  // internal template UUID
	WorkflowID  string            `json:"workflow_id,omitempty"` // provider workflow/template ID
	Language    string            `json:"language,omitempty"`
	ProviderID  string            `json:"provider_id,omitempty"` // optional override; defaults to active provider
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// HostedSessionView represents the hosted verification session tracking URL and status.
type HostedSessionView struct {
	ID              uuid.UUID     `json:"id"`
	UserID          uuid.UUID     `json:"user_id"`
	TemplateID      *uuid.UUID    `json:"template_id,omitempty"`
	Provider        string        `json:"provider"`
	SessionID       string        `json:"session_id"`
	SessionToken    string        `json:"session_token,omitempty"`
	VerificationURL string        `json:"verification_url"`
	VendorData      string        `json:"vendor_data,omitempty"`
	Status          SessionStatus `json:"status"`
	Decision        *string       `json:"decision,omitempty"`
	Reason          *string       `json:"reason,omitempty"`
	RiskScore       *float64      `json:"risk_score,omitempty"`
	ExpiresAt       *time.Time    `json:"expires_at,omitempty"`
	VerifiedAt      *time.Time    `json:"verified_at,omitempty"`
	DeclinedAt      *time.Time    `json:"declined_at,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

// UserVerificationStatus represents the verification standing of a user.
type UserVerificationStatus struct {
	UserID             uuid.UUID          `json:"user_id"`
	IsVerified         bool               `json:"is_verified"`
	ActiveIdentity     *UserIdentityView  `json:"active_identity,omitempty"`
	LatestSession      *HostedSessionView `json:"latest_session,omitempty"`
	ActiveSessionCount int                `json:"active_session_count"`
	TotalSessions      int                `json:"total_sessions"`
}

// Service defines the interface to manage access to user identity verification, hosted sessions, and templates.
type Service interface {
	// CreateHostedSession initiates a verification session for a user and records it in the database.
	CreateHostedSession(ctx context.Context, req InitiateSessionRequest) (*HostedSessionView, error)

	// GetSessionBySessionID retrieves a hosted session by external provider session ID.
	GetSessionBySessionID(ctx context.Context, sessionID string) (*HostedSessionView, error)

	// GetSessionByID retrieves a hosted session by internal database UUID.
	GetSessionByID(ctx context.Context, id uuid.UUID) (*HostedSessionView, error)

	// GetUserVerificationStatus returns the verification standing, identity, and latest session for a user.
	GetUserVerificationStatus(ctx context.Context, userID uuid.UUID) (*UserVerificationStatus, error)

	// GetUserIdentity retrieves the verified identity details for a user.
	GetUserIdentity(ctx context.Context, userID uuid.UUID) (*UserIdentityView, error)

	// SyncSessionDecision queries the upstream provider for the latest status and synchronizes the db record.
	SyncSessionDecision(ctx context.Context, sessionID string) (*HostedSessionView, error)

	// ProcessWebhook processes and verifies an incoming webhook event from a verification provider.
	ProcessWebhook(ctx context.Context, providerSlug string, rawBody []byte, signature string, timestamp int64) (*HostedSessionView, error)

	// Template management
	CreateTemplate(ctx context.Context, req CreateTemplateRequest) (*TemplateView, error)
	GetTemplate(ctx context.Context, id uuid.UUID) (*TemplateView, error)
	ListTemplates(ctx context.Context, provider string) ([]*TemplateView, error)
	GetDefaultTemplate(ctx context.Context, provider string) (*TemplateView, error)
}

// Manager is an alias for Service.
type Manager = Service

// IVerification is an alias for Service.
type IVerification = Service

type service struct {
	db       *ent.Client
	selector Selector
}

// NewService initializes a new verification service.
func NewService(db *ent.Client, selector Selector) Service {
	return &service{
		db:       db,
		selector: selector,
	}
}

// CreateHostedSession initiates a new verification session and stores the session URL and status.
func (s *service) CreateHostedSession(ctx context.Context, req InitiateSessionRequest) (*HostedSessionView, error) {
	if req.UserID == uuid.Nil {
		return nil, errors.New("user_id is required")
	}

	exists, err := s.db.User.Query().Where(user.IDEQ(req.UserID)).Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("check user existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrUserNotFound, req.UserID)
	}

	p, err := s.selector.Select(ctx, req.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("resolve verification provider: %w", err)
	}

	// Resolve template / workflow ID
	var (
		templateUUID *uuid.UUID
		workflowID   = req.WorkflowID
	)

	if req.TemplateID != nil {
		tmpl, tmplErr := s.db.HostedSessionTemplate.Get(ctx, *req.TemplateID)
		if tmplErr != nil {
			return nil, fmt.Errorf("%w: %s", ErrTemplateNotFound, *req.TemplateID)
		}
		templateUUID = &tmpl.ID
		if workflowID == "" {
			workflowID = tmpl.TemplateID
		}
	} else if workflowID != "" {
		if tmpl, tmplErr := s.db.HostedSessionTemplate.Query().
			Where(
				hostedsessiontemplate.ProviderEQ(p.Slug()),
				hostedsessiontemplate.TemplateIDEQ(workflowID),
			).First(ctx); tmplErr == nil {
			templateUUID = &tmpl.ID
		}
	} else {
		// Look up active default template for this provider
		if defTmpl, tmplErr := s.db.HostedSessionTemplate.Query().
			Where(
				hostedsessiontemplate.ProviderEQ(p.Slug()),
				hostedsessiontemplate.IsDefault(true),
				hostedsessiontemplate.Active(true),
			).First(ctx); tmplErr == nil {
			templateUUID = &defTmpl.ID
			workflowID = defTmpl.TemplateID
		}
	}

	vendorData := req.UserID.String()
	providerReq := CreateSessionRequest{
		WorkflowID:  workflowID,
		VendorData:  vendorData,
		CallbackURL: req.CallbackURL,
		Language:    req.Language,
	}

	resp, err := p.CreateSession(ctx, providerReq)
	if err != nil {
		return nil, fmt.Errorf("provider create session (%s): %w", p.Slug(), err)
	}

	builder := s.db.HostedSession.Create().
		SetUserID(req.UserID).
		SetProvider(p.Slug()).
		SetSessionID(resp.SessionID).
		SetVerificationURL(resp.VerificationURL).
		SetStatus(string(resp.Status)).
		SetVendorData(vendorData)

	if templateUUID != nil {
		builder.SetTemplateID(*templateUUID)
	}
	if resp.SessionToken != "" {
		builder.SetSessionToken(resp.SessionToken)
	}
	if resp.ExpiresAt != "" {
		if t, parseErr := time.Parse(time.RFC3339, resp.ExpiresAt); parseErr == nil {
			builder.SetExpiresAt(t)
		}
	}

	record, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("save hosted session: %w", err)
	}

	return toHostedSessionView(record), nil
}

// GetSessionBySessionID retrieves a hosted session by external provider session ID.
func (s *service) GetSessionBySessionID(ctx context.Context, sessionID string) (*HostedSessionView, error) {
	record, err := s.db.HostedSession.Query().
		Where(hostedsession.SessionIDEQ(sessionID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("%w: %s", ErrSessionNotFound, sessionID)
		}
		return nil, fmt.Errorf("get hosted session by session_id: %w", err)
	}
	return toHostedSessionView(record), nil
}

// GetSessionByID retrieves a hosted session by database primary key.
func (s *service) GetSessionByID(ctx context.Context, id uuid.UUID) (*HostedSessionView, error) {
	record, err := s.db.HostedSession.Query().
		Where(hostedsession.IDEQ(id)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("%w: %s", ErrSessionNotFound, id)
		}
		return nil, fmt.Errorf("get hosted session by id: %w", err)
	}
	return toHostedSessionView(record), nil
}

// GetUserVerificationStatus checks user standing, fetches the verified identity and latest session.
func (s *service) GetUserVerificationStatus(ctx context.Context, userID uuid.UUID) (*UserVerificationStatus, error) {
	u, err := s.db.User.Query().Where(user.IDEQ(userID)).First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("%w: %s", ErrUserNotFound, userID)
		}
		return nil, fmt.Errorf("query user: %w", err)
	}

	sessions, err := s.db.HostedSession.Query().
		Where(hostedsession.UserIDEQ(userID)).
		Order(ent.Desc(hostedsession.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query user hosted sessions: %w", err)
	}

	status := &UserVerificationStatus{
		UserID:        userID,
		IsVerified:    u.IdentityVerified,
		TotalSessions: len(sessions),
	}

	if len(sessions) > 0 {
		status.LatestSession = toHostedSessionView(sessions[0])
		for _, sess := range sessions {
			sStatus := SessionStatus(sess.Status)
			if !sStatus.IsTerminal() {
				status.ActiveSessionCount++
			}
			if sStatus.IsApproved() {
				status.IsVerified = true
			}
		}
	}

	// Fetch active verified identity if available
	if identity, err := s.db.UserIdentity.Query().
		Where(useridentity.UserIDEQ(userID), useridentity.IsVerified(true)).
		Order(ent.Desc(useridentity.FieldVerifiedAt)).
		First(ctx); err == nil {
		status.ActiveIdentity = toUserIdentityView(identity)
	}

	return status, nil
}

// GetUserIdentity retrieves the verified identity details for a user.
func (s *service) GetUserIdentity(ctx context.Context, userID uuid.UUID) (*UserIdentityView, error) {
	identity, err := s.db.UserIdentity.Query().
		Where(useridentity.UserIDEQ(userID)).
		Order(ent.Desc(useridentity.FieldCreatedAt)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("%w: %s", ErrIdentityNotFound, userID)
		}
		return nil, fmt.Errorf("query user identity: %w", err)
	}
	return toUserIdentityView(identity), nil
}

// SyncSessionDecision fetches the latest outcome from the provider and updates session & user identity.
func (s *service) SyncSessionDecision(ctx context.Context, sessionID string) (*HostedSessionView, error) {
	record, err := s.db.HostedSession.Query().
		Where(hostedsession.SessionIDEQ(sessionID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("%w: %s", ErrSessionNotFound, sessionID)
		}
		return nil, fmt.Errorf("query hosted session: %w", err)
	}

	p, err := s.selector.Select(ctx, record.Provider)
	if err != nil {
		return nil, fmt.Errorf("resolve provider %s: %w", record.Provider, err)
	}

	decision, err := p.GetDecision(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get provider decision: %w", err)
	}

	updater := s.db.HostedSession.UpdateOneID(record.ID).
		SetStatus(string(decision.Status))

	if decision.Decision != nil {
		if decision.Decision.Decision != "" {
			updater.SetDecision(decision.Decision.Decision)
		}
		if decision.Decision.Reason != "" {
			updater.SetReason(decision.Decision.Reason)
		}
		updater.SetRiskScore(decision.Decision.RiskScore)
	}

	if decision.Status.IsApproved() {
		now := time.Now()
		updater.SetVerifiedAt(now)
		_ = s.db.User.UpdateOneID(record.UserID).SetIdentityVerified(true).Exec(ctx)

		// Create/update UserIdentity record
		s.recordVerifiedIdentity(ctx, record.UserID, record.ID, record.Provider, now, nil)
	} else if decision.Status.IsDeclined() {
		updater.SetDeclinedAt(time.Now())
	}

	updated, err := updater.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("update hosted session: %w", err)
	}

	return toHostedSessionView(updated), nil
}

// ProcessWebhook parses a provider webhook callback and updates session & user identity.
func (s *service) ProcessWebhook(ctx context.Context, providerSlug string, rawBody []byte, signature string, timestamp int64) (*HostedSessionView, error) {
	p, err := s.selector.Select(ctx, providerSlug)
	if err != nil {
		return nil, fmt.Errorf("resolve provider %s: %w", providerSlug, err)
	}

	event, err := p.ParseWebhook(rawBody, signature, timestamp)
	if err != nil {
		return nil, fmt.Errorf("parse webhook: %w", err)
	}

	record, err := s.db.HostedSession.Query().
		Where(hostedsession.SessionIDEQ(event.SessionID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("%w: %s", ErrSessionNotFound, event.SessionID)
		}
		return nil, fmt.Errorf("query hosted session: %w", err)
	}

	updater := s.db.HostedSession.UpdateOneID(record.ID).
		SetStatus(string(event.Status))

	if event.Decision != nil {
		if event.Decision.Decision != "" {
			updater.SetDecision(event.Decision.Decision)
		}
		if event.Decision.Reason != "" {
			updater.SetReason(event.Decision.Reason)
		}
		updater.SetRiskScore(event.Decision.RiskScore)
	}

	if event.Status.IsApproved() {
		now := time.Now()
		updater.SetVerifiedAt(now)
		_ = s.db.User.UpdateOneID(record.UserID).SetIdentityVerified(true).Exec(ctx)

		// Create/update UserIdentity record
		s.recordVerifiedIdentity(ctx, record.UserID, record.ID, record.Provider, now, nil)
	} else if event.Status.IsDeclined() {
		updater.SetDeclinedAt(time.Now())
	}

	updated, err := updater.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("update hosted session from webhook: %w", err)
	}

	return toHostedSessionView(updated), nil
}

// recordVerifiedIdentity records or updates the verified identity details in UserIdentity.
func (s *service) recordVerifiedIdentity(ctx context.Context, userID, sessionID uuid.UUID, providerName string, verifiedAt time.Time, rawData map[string]any) {
	_ = s.db.UserIdentity.Create().
		SetUserID(userID).
		SetHostedSessionID(sessionID).
		SetProvider(providerName).
		SetIsVerified(true).
		SetVerifiedAt(verifiedAt).
		SetRawData(rawData).
		Exec(ctx)
}

// CreateTemplate registers a new hosted session template.
func (s *service) CreateTemplate(ctx context.Context, req CreateTemplateRequest) (*TemplateView, error) {
	if req.Name == "" {
		return nil, errors.New("template name is required")
	}
	if req.Provider == "" {
		return nil, errors.New("provider is required")
	}
	if req.TemplateID == "" {
		return nil, errors.New("template_id is required")
	}

	builder := s.db.HostedSessionTemplate.Create().
		SetName(req.Name).
		SetProvider(req.Provider).
		SetTemplateID(req.TemplateID).
		SetIsDefault(req.IsDefault).
		SetActive(req.Active)

	if req.Description != "" {
		builder.SetDescription(req.Description)
	}
	if req.Metadata != nil {
		builder.SetMetadata(req.Metadata)
	}

	tmpl, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("save template: %w", err)
	}

	return toTemplateView(tmpl), nil
}

// GetTemplate retrieves a template by ID.
func (s *service) GetTemplate(ctx context.Context, id uuid.UUID) (*TemplateView, error) {
	tmpl, err := s.db.HostedSessionTemplate.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("%w: %s", ErrTemplateNotFound, id)
		}
		return nil, fmt.Errorf("get template: %w", err)
	}
	return toTemplateView(tmpl), nil
}

// ListTemplates lists active templates for a given provider.
func (s *service) ListTemplates(ctx context.Context, provider string) ([]*TemplateView, error) {
	q := s.db.HostedSessionTemplate.Query()
	if provider != "" {
		q = q.Where(hostedsessiontemplate.ProviderEQ(provider))
	}
	tmpls, err := q.Order(ent.Desc(hostedsessiontemplate.FieldIsDefault)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}

	views := make([]*TemplateView, len(tmpls))
	for i, t := range tmpls {
		views[i] = toTemplateView(t)
	}
	return views, nil
}

// GetDefaultTemplate retrieves the default template for a provider.
func (s *service) GetDefaultTemplate(ctx context.Context, provider string) (*TemplateView, error) {
	tmpl, err := s.db.HostedSessionTemplate.Query().
		Where(
			hostedsessiontemplate.ProviderEQ(provider),
			hostedsessiontemplate.IsDefault(true),
			hostedsessiontemplate.Active(true),
		).First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("default template not found for provider %s: %w", provider, ErrTemplateNotFound)
		}
		return nil, fmt.Errorf("get default template: %w", err)
	}
	return toTemplateView(tmpl), nil
}

// toHostedSessionView maps an Ent HostedSession entity to the domain HostedSessionView.
func toHostedSessionView(s *ent.HostedSession) *HostedSessionView {
	if s == nil {
		return nil
	}

	return &HostedSessionView{
		ID:              s.ID,
		UserID:          s.UserID,
		TemplateID:      s.TemplateID,
		Provider:        s.Provider,
		SessionID:       s.SessionID,
		SessionToken:    s.SessionToken,
		VerificationURL: s.VerificationURL,
		VendorData:      s.VendorData,
		Status:          SessionStatus(s.Status),
		Decision:        s.Decision,
		Reason:          s.Reason,
		RiskScore:       s.RiskScore,
		ExpiresAt:       s.ExpiresAt,
		VerifiedAt:      s.VerifiedAt,
		DeclinedAt:      s.DeclinedAt,
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
	}
}

// toUserIdentityView maps an Ent UserIdentity entity to the domain UserIdentityView.
func toUserIdentityView(u *ent.UserIdentity) *UserIdentityView {
	if u == nil {
		return nil
	}

	return &UserIdentityView{
		ID:                 u.ID,
		UserID:             u.UserID,
		HostedSessionID:    u.HostedSessionID,
		Provider:           u.Provider,
		FirstName:          u.FirstName,
		LastName:           u.LastName,
		DOB:                u.Dob,
		Gender:             u.Gender,
		Nationality:        u.Nationality,
		DocumentType:       u.DocumentType,
		DocumentNumber:     u.DocumentNumber,
		DocumentCountry:    u.DocumentCountry,
		DocumentExpiration: u.DocumentExpiration,
		Address:            u.Address,
		IsVerified:         u.IsVerified,
		VerifiedAt:         u.VerifiedAt,
		RawData:            u.RawData,
		CreatedAt:          u.CreatedAt,
		UpdatedAt:          u.UpdatedAt,
	}
}

// toTemplateView maps an Ent HostedSessionTemplate entity to the domain TemplateView.
func toTemplateView(t *ent.HostedSessionTemplate) *TemplateView {
	if t == nil {
		return nil
	}

	return &TemplateView{
		ID:          t.ID,
		Name:        t.Name,
		Provider:    t.Provider,
		TemplateID:  t.TemplateID,
		Description: t.Description,
		IsDefault:   t.IsDefault,
		Active:      t.Active,
		Metadata:    t.Metadata,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}
