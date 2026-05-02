package projection

import "time"

type runnerOptions struct {
	retryCount int
	retryDelay time.Duration
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
