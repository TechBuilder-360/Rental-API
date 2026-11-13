package runtime

import (
	"fmt"

	"github.com/rs/zerolog"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/TechBuilder-360/Rental-API/internal/config"
)

// TemporalRuntime manages the Temporal client and worker lifecycle.
type TemporalRuntime struct {
	Client client.Client
	Worker worker.Worker
	log    zerolog.Logger
}

// SetupTemporal initializes the Temporal client and starts the worker.
func SetupTemporal(cfg config.TemporalConfig, log zerolog.Logger) (*TemporalRuntime, error) {
	c, err := NewTemporalClient(cfg, log)
	if err != nil {
		return nil, err
	}

	w := NewTemporalWorker(c, cfg.TaskQueue, worker.Options{})
	if err := w.Start(); err != nil {
		c.Close()
		return nil, fmt.Errorf("failed to start temporal worker: %w", err)
	}

	return &TemporalRuntime{
		Client: c,
		Worker: w,
		log:    log,
	}, nil
}

// Stop gracefully stops the Temporal worker and closes the client connection.
func (tr *TemporalRuntime) Stop() {
	if tr.Worker != nil {
		tr.Worker.Stop()
	}
	if tr.Client != nil {
		tr.Client.Close()
	}
	tr.log.Info().Msg("Temporal runtime stopped")
}
