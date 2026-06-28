package projectionhost

import (
	"time"
)

// HostOption configures a Host.
type HostOption func(*hostOptions)

type hostOptions struct {
	maxRestarts    int
	backoffInitial time.Duration
	backoffMax     time.Duration
	batchSize      int
	dlq            DeadLetterStore
	dlqThreshold   int
}

func defaultOptions() hostOptions {
	return hostOptions{
		maxRestarts:    5,
		backoffInitial: 1 * time.Second,
		backoffMax:     30 * time.Second,
		batchSize:      100,
		dlqThreshold:   3,
	}
}

// WithMaxRestarts sets the maximum number of restarts per worker before it
// transitions to WorkerFailed. Default: 5. Set to -1 for unlimited.
func WithMaxRestarts(n int) HostOption {
	return func(o *hostOptions) { o.maxRestarts = n }
}

// WithBackoff sets the initial and maximum exponential backoff duration
// between restarts. Default: 1s initial, 30s max.
func WithBackoff(initial, max time.Duration) HostOption {
	return func(o *hostOptions) {
		o.backoffInitial = initial
		o.backoffMax = max
	}
}

// WithBatchSize sets the number of events read per journal batch. Default: 100.
func WithBatchSize(n int) HostOption {
	return func(o *hostOptions) { o.batchSize = n }
}

// WithDeadLetterStore enables poison-message capture. Events that fail more
// than threshold times are stored in the DLQ and the checkpoint advances.
func WithDeadLetterStore(store DeadLetterStore, threshold int) HostOption {
	return func(o *hostOptions) {
		o.dlq = store
		if threshold > 0 {
			o.dlqThreshold = threshold
		}
	}
}
