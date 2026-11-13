package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/TechBuilder-360/Rental-API/internal/config"
)

// NewRedisClient initializes a Redis client and verifies connectivity.
func NewRedisClient(ctx context.Context, cfg config.RedisConfig, log zerolog.Logger) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolSize:     cfg.PoolSize,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("failed to ping redis server at %s: %w", cfg.Addr, err)
	}

	log.Info().
		Str("addr", cfg.Addr).
		Int("db", cfg.DB).
		Msg("connected to Redis")

	return client, nil
}
