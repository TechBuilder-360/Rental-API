package resolver

//go:generate sh -c "cd ../../.. && go tool gqlgen generate"

import (
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"go.temporal.io/sdk/client"

	"github.com/TechBuilder-360/Rental-API/ent"
	"github.com/TechBuilder-360/Rental-API/internal/auth"
)

// Resolver is the root resolver. It holds the dependencies shared by all resolvers.
// This file will not be regenerated automatically.
type Resolver struct {
	DB       *ent.Client
	Redis    *redis.Client
	Temporal client.Client
	Auth     *auth.Service
	Log      zerolog.Logger
}
