package runtime

import (
	"fmt"

	"github.com/rs/zerolog"
	"go.temporal.io/sdk/client"

	"github.com/TechBuilder-360/Rental-API/internal/config"
	"github.com/TechBuilder-360/Rental-API/pkg/logger"
)

// NewTemporalClient dials a Temporal server and returns an initialized Temporal client.
func NewTemporalClient(cfg config.TemporalConfig, log zerolog.Logger) (client.Client, error) {
	c, err := client.Dial(client.Options{
		HostPort:  cfg.HostPort,
		Namespace: cfg.Namespace,
		Logger:    logger.NewTemporalAdapter(log),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create temporal client: %w", err)
	}

	log.Info().
		Str("host_port", cfg.HostPort).
		Str("namespace", cfg.Namespace).
		Str("task_queue", cfg.TaskQueue).
		Msg("connected to Temporal")

	return c, nil
}
