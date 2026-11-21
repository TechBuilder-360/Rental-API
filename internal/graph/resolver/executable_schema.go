package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"

	"github.com/TechBuilder-360/Rental-API/internal/auth"
	"github.com/TechBuilder-360/Rental-API/internal/graph/generated"
)

// NewExecutableSchema builds the GraphQL schema with resolvers and directives wired up.
func NewExecutableSchema(r *Resolver) graphql.ExecutableSchema {
	return generated.NewExecutableSchema(generated.Config{
		Resolvers: r,
		Directives: generated.DirectiveRoot{
			Auth: authDirective,
		},
	})
}

// authDirective implements @auth: the field resolves only for authenticated callers.
// The principal is injected into ctx by the HTTP auth middleware.
func authDirective(ctx context.Context, _ any, next graphql.Resolver) (any, error) {
	if _, ok := auth.PrincipalFromContext(ctx); !ok {
		return nil, newError(ctx, CodeUnauthenticated, "authentication required")
	}
	return next(ctx)
}
