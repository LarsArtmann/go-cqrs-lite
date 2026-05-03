package projection

import (
	"log/slog"
	"time"
)

type runnerOptions struct {
	retryCount int
	retryDelay time.Duration
	logger     *slog.Logger
}

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
