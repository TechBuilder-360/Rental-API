package resolver

import (
	"context"
	"errors"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/TechBuilder-360/Rental-API/internal/auth"
)

// Error codes exposed in GraphQL `extensions.code`.
const (
	CodeBadUserInput    = "BAD_USER_INPUT"
	CodeUnauthenticated = "UNAUTHENTICATED"
	CodeForbidden       = "FORBIDDEN"
	CodeConflict        = "CONFLICT"
	CodeRateLimited     = "RATE_LIMITED"
	CodeInternal        = "INTERNAL_SERVER_ERROR"
)

// newError builds a GraphQL error with a machine-readable code.
func newError(ctx context.Context, code, message string) *gqlerror.Error {
	return &gqlerror.Error{
		Message:    message,
		Path:       graphql.GetPath(ctx),
		Extensions: map[string]any{"code": code},
	}
}

// toGQLError converts service errors into client-safe GraphQL errors.
// Unknown errors are logged and replaced with a generic message.
func (r *Resolver) toGQLError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	var verr *auth.ValidationError
	switch {
	case errors.As(err, &verr):
		e := newError(ctx, CodeBadUserInput, verr.Error())
		e.Extensions["field"] = verr.Field
		return e
	case errors.Is(err, auth.ErrInvalidCredentials),
		errors.Is(err, auth.ErrOTPInvalid),
		errors.Is(err, auth.ErrInvalidRefreshToken),
		errors.Is(err, auth.ErrInvalidToken),
		errors.Is(err, auth.ErrUnauthenticated):
		return newError(ctx, CodeUnauthenticated, err.Error())
	case errors.Is(err, auth.ErrAccountDisabled):
		return newError(ctx, CodeForbidden, err.Error())
	case errors.Is(err, auth.ErrAccountExists):
		return newError(ctx, CodeConflict, err.Error())
	case errors.Is(err, auth.ErrOTPRateLimited):
		return newError(ctx, CodeRateLimited, err.Error())
	}

	r.Log.Error().Err(err).Str("path", graphql.GetPath(ctx).String()).Msg("resolver error")
	return newError(ctx, CodeInternal, "internal server error")
}
