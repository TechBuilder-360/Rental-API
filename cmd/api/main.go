package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/TechBuilder-360/Rental-API/cmd/api/runtime/temporal"
	"github.com/TechBuilder-360/Rental-API/cmd/migration/seeders"
	"github.com/TechBuilder-360/Rental-API/internal/auth"
	"github.com/TechBuilder-360/Rental-API/internal/cache"
	"github.com/TechBuilder-360/Rental-API/internal/config"
	"github.com/TechBuilder-360/Rental-API/internal/database"
	"github.com/TechBuilder-360/Rental-API/internal/email"
	"github.com/TechBuilder-360/Rental-API/internal/graph/resolver"
	"github.com/TechBuilder-360/Rental-API/internal/server"
	"github.com/TechBuilder-360/Rental-API/pkg/logger"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.Logger.Level, cfg.Logger.Pretty, cfg.App.Name)

	log.Info().
		Str("env", cfg.App.Env).
		Int("port", cfg.App.Port).
		Msg("starting API server")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 1. Initialize PostgreSQL (Ent)
	dbClient, err := database.NewPostgresClient(ctx, cfg.Postgres, log)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize PostgreSQL")
	}
	defer func() {
		if err := dbClient.Close(); err != nil {
			log.Error().Err(err).Msg("error closing PostgreSQL client")
		}
	}()

	// Run seeders if auto-migrate is enabled
	if cfg.Postgres.AutoMigrate {
		if err := seeders.Seed(ctx, dbClient); err != nil {
			log.Warn().Err(err).Msg("database seeding encountered error")
		} else {
			log.Info().Msg("database default services and providers seeded")
		}
	}

	// 2. Initialize Redis
	redisClient, err := cache.NewRedisClient(ctx, cfg.Redis, log)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize Redis")
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Error().Err(err).Msg("error closing Redis client")
		}
	}()

	// 3. Initialize Temporal Runtime
	temporalRuntime, err := temporal.SetupTemporal(cfg.Temporal, log)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize Temporal runtime")
	}
	defer temporalRuntime.Stop()

	// 4. Initialize Auth and Email services
	hasher, err := auth.NewPasswordHasher(cfg.Auth.BcryptCost)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize password hasher")
	}
	tokenMgr, err := auth.NewTokenManager(cfg.Auth.JWTSecret, cfg.Auth.JWTIssuer, cfg.Auth.AccessTokenTTL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize token manager")
	}
	sessionStore := auth.NewSessionStore(redisClient, cfg.Auth.RefreshTokenTTL)
	otpStore := auth.NewOTPStore(redisClient, cfg.Auth.OTPTTL, cfg.Auth.OTPMaxAttempts, cfg.Auth.OTPResendDelay)

	emailSender, err := email.NewDynamicSenderWithConfig(dbClient, cfg.Email, log)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize email sender")
	}
	email.Register(temporalRuntime.Worker, emailSender)
	emailDispatcher := email.NewTemporalDispatcher(temporalRuntime.Client, cfg.Temporal.TaskQueue)

	authService := auth.NewService(
		auth.NewEntUserStore(dbClient),
		hasher,
		tokenMgr,
		sessionStore,
		otpStore,
		emailDispatcher,
		log,
	)

	// 5. Initialize HTTP server (Gin + gqlgen)
	res := &resolver.Resolver{
		DB:       dbClient,
		Redis:    redisClient,
		Temporal: temporalRuntime.Client,
		Auth:     authService,
		Log:      log,
	}
	srv := server.New(cfg.App, res, authService, log)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.Start()
	}()

	log.Info().Msg("all services initialized; awaiting shutdown signal")
	select {
	case <-ctx.Done():
	case err := <-serverErr:
		if err != nil {
			log.Error().Err(err).Msg("HTTP server stopped unexpectedly")
		}
	}

	log.Info().Msg("shutting down gracefully")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("error shutting down HTTP server")
	}
}
