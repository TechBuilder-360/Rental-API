package main

import (
	"context"
	"os"

	"github.com/TechBuilder-360/Rental-API/cmd/migration/seeders"
	"github.com/TechBuilder-360/Rental-API/internal/config"
	"github.com/TechBuilder-360/Rental-API/internal/database"
	"github.com/TechBuilder-360/Rental-API/pkg/logger"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.Logger.Level, cfg.Logger.Pretty, "migration-seeder")

	ctx := context.Background()

	dbClient, err := database.NewPostgresClient(ctx, cfg.Postgres, log)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to postgres for seeding")
	}
	defer dbClient.Close()

	log.Info().Msg("starting database seeder...")
	if err := seeders.Seed(ctx, dbClient); err != nil {
		log.Fatal().Err(err).Msg("database seeding failed")
		os.Exit(1)
	}

	log.Info().Msg("database seeding completed successfully")
}
