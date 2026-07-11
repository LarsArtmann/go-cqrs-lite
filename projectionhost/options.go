package projectionhost

import (
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
)

// HostOption configures a Host.
type HostOption func(*hostOptions)

type hostOptions struct {
	maxRestarts     int
	backoffInitial  time.Duration
	backoffMax      time.Duration
	batchSize       int
	dlq             DeadLetterStore
	dlqThreshold    int
	logger          *slog.Logger
	metrics         MetricsRecorder
	subscriber      event.Subscriber
	shutdownTimeout time.Duration
	onFailed        func(projectionName, lastError string)
}

func defaultOptions() hostOptions {
	return hostOptions{ //nolint:exhaustruct // option fields default to zero
		maxRestarts:     5,
		backoffInitial:  1 * time.Second,
		backoffMax:      30 * time.Second,
		batchSize:       100,
		dlqThreshold:    3,
		logger:          slog.Default(),
		shutdownTimeout: 30 * time.Second,
	}
}

// tracer returns the OTel tracer for projectionhost. Lazily fetched from the
// global provider so consumers can set up their provider after calling New.
// Returns a no-op tracer when no provider is configured.
func tracer() trace.Tracer {
	return otel.GetTracerProvider().Tracer(cqrsotel.ComponentTracer("projectionhost"))
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

// WithLogger sets the structured logger for worker lifecycle events (crashes,
// restarts, dead-letter captures). Default: slog.Default().
func WithLogger(l *slog.Logger) HostOption {
	return func(o *hostOptions) {
		if l != nil {
			o.logger = l
		}
	}
}

// WithSubscriber enables live event processing after journal drain. When set,
// each worker drains the journal (replay), then transitions to live subscription
// via subscriber.SubscribeAll. Events seen during replay are deduped in the live
// phase to prevent double-processing at the replay→live boundary.
//
// Without this option, the host is a batch-drainer: workers exit after catching up.
func WithSubscriber(subscriber event.Subscriber) HostOption {
	return func(o *hostOptions) {
		o.subscriber = subscriber
	}
}

// WithShutdownTimeout sets the maximum duration Stop waits for in-flight events
// to complete before returning a timeout error. Default: 30s. Increase for
// projections with slow handlers; decrease for fast-fail shutdown requirements.
func WithShutdownTimeout(d time.Duration) HostOption {
	return func(o *hostOptions) { o.shutdownTimeout = d }
}

// WithOnFailed registers a callback invoked when a worker exhausts its restart
// budget and transitions to WorkerFailed. The callback receives the projection
// name and the last error message. Use this for alerting (e.g. send a Slack
// notification, increment an external error counter, trigger a page).
//
// The callback is invoked synchronously from the worker goroutine — keep it
// fast and non-blocking. For expensive operations, dispatch to a separate
// goroutine from within the callback.
func WithOnFailed(fn func(projectionName, lastError string)) HostOption {
	return func(o *hostOptions) { o.onFailed = fn }
}
