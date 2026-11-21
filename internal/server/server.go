package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/TechBuilder-360/Rental-API/internal/auth"
	"github.com/TechBuilder-360/Rental-API/internal/config"
	"github.com/TechBuilder-360/Rental-API/internal/graph/resolver"
)

const (
	graphqlPath    = "/graphql"
	playgroundPath = "/playground"
)

// Server wraps the Gin engine and the underlying HTTP server.
type Server struct {
	http *http.Server
	log  zerolog.Logger
}

// New builds the Gin router with GraphQL, playground, and health endpoints.
func New(cfg config.AppConfig, res *resolver.Resolver, authSvc *auth.Service, log zerolog.Logger) *Server {
	if cfg.Env != "development" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(recovery(log), requestLogger(log), authMiddleware(authSvc))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	gql := newGraphQLHandler(res)
	router.POST(graphqlPath, gin.WrapH(gql))
	router.GET(graphqlPath, gin.WrapH(gql))

	if cfg.Env == "development" {
		router.GET(playgroundPath, gin.WrapH(playground.Handler("Rental API", graphqlPath)))
	}

	return &Server{
		http: &http.Server{
			Addr:              fmt.Sprintf(":%d", cfg.Port),
			Handler:           router,
			ReadHeaderTimeout: 10 * time.Second,
		},
		log: log,
	}
}

func newGraphQLHandler(res *resolver.Resolver) *handler.Server {
	srv := handler.New(resolver.NewExecutableSchema(res))

	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	return srv
}

// Start runs the HTTP server. It blocks until the server stops.
// http.ErrServerClosed (from Shutdown) is not treated as an error.
func (s *Server) Start() error {
	s.log.Info().Str("addr", s.http.Addr).Msg("HTTP server listening")
	if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}
