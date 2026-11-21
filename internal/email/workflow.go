package email

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const (
	SendEmailWorkflowName = "SendEmailWorkflow"
	SendEmailActivityName = "SendEmailActivity"
)

// Activities holds the email activity implementations.
type Activities struct {
	Sender Sender
}

// SendEmail delivers one message. Permanent failures are marked non-retryable.
func (a *Activities) SendEmail(ctx context.Context, msg Message) error {
	if err := a.Sender.Send(ctx, msg); err != nil {
		if IsPermanent(err) {
			return temporal.NewNonRetryableApplicationError(err.Error(), "EmailPermanentError", err)
		}
		return err
	}
	return nil
}

// SendEmailWorkflow sends an email via the SendEmail activity with retries.
func SendEmailWorkflow(ctx workflow.Context, msg Message) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		// Give up after 10 minutes; OTP codes expire long before that anyway.
		ScheduleToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    10,
		},
	})
	return workflow.ExecuteActivity(ctx, SendEmailActivityName, msg).Get(ctx, nil)
}

// Register registers the email workflow and activities on a Temporal worker.
func Register(w worker.Registry, sender Sender) {
	w.RegisterWorkflowWithOptions(SendEmailWorkflow, workflow.RegisterOptions{Name: SendEmailWorkflowName})
	acts := &Activities{Sender: sender}
	w.RegisterActivityWithOptions(acts.SendEmail, activity.RegisterOptions{Name: SendEmailActivityName})
}

// Dispatcher enqueues emails for asynchronous delivery by the Temporal worker.
type Dispatcher interface {
	Dispatch(ctx context.Context, msg Message) error
}

// TemporalDispatcher starts a SendEmailWorkflow per message.
type TemporalDispatcher struct {
	client    client.Client
	taskQueue string
}

func NewTemporalDispatcher(c client.Client, taskQueue string) *TemporalDispatcher {
	return &TemporalDispatcher{client: c, taskQueue: taskQueue}
}

func (d *TemporalDispatcher) Dispatch(ctx context.Context, msg Message) error {
	_, err := d.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:        "email-" + uuid.NewString(),
		TaskQueue: d.taskQueue,
	}, SendEmailWorkflowName, msg)
	if err != nil {
		return fmt.Errorf("start email workflow: %w", err)
	}
	return nil
}
