package projection

import "time"

type runnerOptions struct {
	batchSize   int
	batchWindow time.Duration
	retryCount  int
	retryDelay  time.Duration
	concurrency int
}

// RunnerOption configures a projection Runner.
type RunnerOption func(*runnerOptions)

// WithBatchSize enables batched processing. Events are collected into
// batches of up to maxSize before being dispatched to handlers.
func WithBatchSize(maxSize int) RunnerOption {
	return func(o *runnerOptions) { o.batchSize = maxSize }
}

// WithBatchWindow enables time-windowed batching. Events are collected
// within the given duration before being dispatched as a batch.
func WithBatchWindow(window time.Duration) RunnerOption {
	return func(o *runnerOptions) { o.batchWindow = window }
}

// WithRetry enables automatic retry on handler errors.
// count is the maximum number of retry attempts.
// delay is the initial backoff delay between retries.
func WithRetry(count int, delay time.Duration) RunnerOption {
	return func(o *runnerOptions) {
		o.retryCount = count
		o.retryDelay = delay
	}
}

// WithConcurrency sets the number of concurrent handler goroutines
// per partition. Default is 1 (sequential).
func WithConcurrency(n int) RunnerOption {
	return func(o *runnerOptions) { o.concurrency = n }
}
