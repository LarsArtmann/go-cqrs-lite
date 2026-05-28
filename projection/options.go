package projection

import (
	"context"
	"log/slog"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

type runnerOptions struct {
	retryCount      int
	retryDelay      time.Duration
	logger          *slog.Logger
	deadLetter      DeadLetterHandler
}

// DeadLetterHandler is called when a projection handler fails after all retries are exhausted.
type DeadLetterHandler func(ctx context.Context, projectionName string, evt event.Event, err error)

// RunnerOption configures a projection Runner.
type RunnerOption func(*runnerOptions)

// WithRetry enables automatic retry on handler errors.
// count is the maximum number of retry attempts.
// delay is the initial backoff delay between retries.
func WithRetry(count int, delay time.Duration) RunnerOption {
	return func(o *runnerOptions) {
		o.retryCount = count
		o.retryDelay = delay
	}
}

// WithLogger sets the structured logger for the runner.
// Defaults to slog.Default() if not set.
func WithLogger(logger *slog.Logger) RunnerOption {
	return func(o *runnerOptions) {
		o.logger = logger
	}
}

// WithDeadLetterHandler sets a handler that is called when a projection event
// fails after all retry attempts are exhausted.
func WithDeadLetterHandler(h DeadLetterHandler) RunnerOption {
	return func(o *runnerOptions) {
		o.deadLetter = h
	}
}
