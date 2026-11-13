package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog"

	"github.com/TechBuilder-360/Rental-API/ent"
	"github.com/TechBuilder-360/Rental-API/internal/config"
)

// NewPostgresClient opens a PostgreSQL connection via pgx stdlib and returns an initialized Ent client.
func NewPostgresClient(ctx context.Context, cfg config.PostgresConfig, log zerolog.Logger) (*ent.Client, error) {
	db, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping postgres database: %w", err)
	}

	drv := entsql.OpenDB(dialect.Postgres, db)
	client := ent.NewClient(
		ent.Driver(drv),
		ent.Log(func(a ... any) {
			log.Debug().Str("component", "ent").Msg(fmt.Sprint(a...))
		}),
	)

	if cfg.AutoMigrate {
		if err := client.Schema.Create(ctx); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("failed running ent auto-migration: %w", err)
		}
		log.Info().Msg("database schema auto-migration completed")
	}

	log.Info().
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Str("db", cfg.DBName).
		Msg("connected to PostgreSQL via Ent")

	return client, nil
}
