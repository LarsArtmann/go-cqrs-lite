package projection

import (
	"context"
	"log/slog"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

type runnerOptions struct {
	retryCount    int
	retryDelay    time.Duration
	retryMaxDelay time.Duration
	logger        *slog.Logger
	deadLetter    DeadLetterHandler
	parallelism   int
	dedupCapacity int
}

// DeadLetterHandler is called when a projection handler fails after all retries are exhausted.
type DeadLetterHandler func(ctx context.Context, projectionName string, evt event.Event, err error)

// RunnerOption configures a projection Runner.
type RunnerOption func(*runnerOptions)

const defaultRetryMaxDelay = 30 * time.Second

// WithRetry enables automatic retry on handler errors.
// count is the maximum number of retry attempts.
// delay is the initial backoff delay between retries.
func WithRetry(count int, delay time.Duration) RunnerOption {
	return func(o *runnerOptions) {
		o.retryCount = count
		o.retryDelay = delay
		o.retryMaxDelay = defaultRetryMaxDelay
	}
}

// WithRetryMaxDelay caps the exponential backoff at the given maximum.
// Must be called after WithRetry to take effect.
func WithRetryMaxDelay(maxDelay time.Duration) RunnerOption {
	return func(o *runnerOptions) {
		o.retryMaxDelay = maxDelay
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

// WithParallelism sets the maximum number of projections that can process an
// event concurrently. A value of 0 or 1 (default) means sequential processing.
// Use values > 1 when projections are independent and I/O-bound.
func WithParallelism(n int) RunnerOption {
	return func(o *runnerOptions) {
		o.parallelism = n
	}
}

// WithDedupCapacity sets a bounded capacity for the live-stream dedup ring.
// When set (> 0), the Runner uses DistinctByEventIDBoundedWith instead of
// DistinctByEventIDWith, evicting the oldest seen IDs (FIFO) once capacity is
// reached. This bounds memory for long-running (24/7) projections.
//
// A value of 0 (default) uses unbounded dedup (exact, but grows forever).
func WithDedupCapacity(n int) RunnerOption {
	return func(o *runnerOptions) {
		o.dedupCapacity = n
	}
}
