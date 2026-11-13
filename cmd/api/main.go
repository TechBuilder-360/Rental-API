package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/TechBuilder-360/Rental-API/cmd/api/runtime"
	"github.com/TechBuilder-360/Rental-API/internal/cache"
	"github.com/TechBuilder-360/Rental-API/internal/config"
	"github.com/TechBuilder-360/Rental-API/internal/database"
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
	temporalRuntime, err := runtime.SetupTemporal(cfg.Temporal, log)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize Temporal runtime")
	}
	defer temporalRuntime.Stop()

	log.Info().Msg("all services initialized; awaiting shutdown signal")
	<-ctx.Done()
	log.Info().Msg("shutting down gracefully")
}
