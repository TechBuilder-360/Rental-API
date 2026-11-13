package runtime

import (
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// NewTemporalWorker creates a Temporal worker listening on the given task queue.
// Register workflows and activities on the returned worker before starting it.
func NewTemporalWorker(c client.Client, taskQueue string, opts worker.Options) worker.Worker {
	return worker.New(c, taskQueue, opts)
}
