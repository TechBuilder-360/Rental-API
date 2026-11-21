package server

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/TechBuilder-360/Rental-API/internal/auth"
)

// requestLogger logs each HTTP request using zerolog.
func requestLogger(log zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" {
			path += "?" + raw
		}

		c.Next()

		status := c.Writer.Status()
		var event *zerolog.Event
		switch {
		case status >= 500:
			event = log.Error()
		case status >= 400:
			event = log.Warn()
		default:
			event = log.Info()
		}

		if len(c.Errors) > 0 {
			event = event.Str("errors", c.Errors.String())
		}

		event.
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", status).
			Dur("latency", time.Since(start)).
			Str("client_ip", c.ClientIP()).
			Int("size", c.Writer.Size()).
			Msg("http request")
	}
}

// recovery recovers from panics and logs them using zerolog.
func recovery(log zerolog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, err any) {
		log.Error().
			Interface("panic", err).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Msg("recovered from panic")
		c.AbortWithStatusJSON(500, gin.H{"error": "internal server error"})
	})
}

// authMiddleware extracts Bearer tokens, validates sessions, and injects Principal and client metadata into context.
func authMiddleware(authSvc *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := auth.WithRequestMeta(c.Request.Context(), auth.SessionMeta{
			UserAgent: c.Request.UserAgent(),
			IP:        c.ClientIP(),
		})

		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			if token != "" && authSvc != nil {
				principal, err := authSvc.Authenticate(ctx, token)
				if err == nil {
					ctx = auth.WithPrincipal(ctx, principal)
				}
			}
		}

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
