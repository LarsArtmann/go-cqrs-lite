package projectionhost

import "time"

// MetricsRecorder observes projection host lifecycle events. Implementations
// forward to Prometheus, OTel, Datadog, or any metrics backend.
//
// All methods must be safe for concurrent use. Implementations must not block
// (fire-and-forget; the host does not wait for metrics recording to complete
// before continuing).
//
// The zero-value-friendly approach: pass nil to WithMetrics to disable.
type MetricsRecorder interface {
	// EventProcessed is called after a projection successfully handles an event.
	EventProcessed(projectionName, eventType string, duration time.Duration)

	// EventErrored is called when a projection handler returns an error
	// (before the DLQ threshold is reached — every failed attempt).
	EventErrored(projectionName, eventType string)

	// EventDeadLettered is called when an event exceeds the retry threshold
	// and is moved to the dead-letter queue.
	EventDeadLettered(projectionName, eventType string)

	// WorkerRestarted is called when a worker crashes and is restarted
	// by the host's crash-recovery loop.
	WorkerRestarted(projectionName string)

	// CheckpointAdvanced is called when a worker persists a new checkpoint.
	// lag is the time between the event's creation and its processing.
	CheckpointAdvanced(projectionName string, lag time.Duration)
}

// WithMetrics wires a MetricsRecorder into the host. Pass nil to disable
// metrics (the default). The recorder receives lifecycle events from every
// worker goroutine.
func WithMetrics(r MetricsRecorder) HostOption {
	return func(o *hostOptions) {
		o.metrics = r
	}
}
