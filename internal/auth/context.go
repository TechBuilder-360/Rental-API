package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// ErrUnauthenticated is returned when an operation requires an authenticated user.
var ErrUnauthenticated = errors.New("unauthenticated")

type ctxKey struct{}
type metaCtxKey struct{}

// WithRequestMeta stores client metadata (user agent, IP) for session records.
func WithRequestMeta(ctx context.Context, m SessionMeta) context.Context {
	return context.WithValue(ctx, metaCtxKey{}, m)
}

// RequestMetaFromContext returns the client metadata, or zero values if absent.
func RequestMetaFromContext(ctx context.Context) SessionMeta {
	m, _ := ctx.Value(metaCtxKey{}).(SessionMeta)
	return m
}

// Principal is the authenticated caller, injected into the request context by the auth middleware.
type Principal struct {
	UserID uuid.UUID
	Claims *Claims
}

// WithPrincipal returns a copy of ctx carrying the principal.
func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

// PrincipalFromContext returns the authenticated principal, if any.
func PrincipalFromContext(ctx context.Context) (*Principal, bool) {
	p, ok := ctx.Value(ctxKey{}).(*Principal)
	return p, ok && p != nil
}

// UserIDFromContext returns the authenticated user's ID, or ErrUnauthenticated.
func UserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	p, ok := PrincipalFromContext(ctx)
	if !ok {
		return uuid.Nil, ErrUnauthenticated
	}
	return p.UserID, nil
}
