// Package projectionhost provides a managed lifecycle for projection workers.
//
// A Host reads events from a [event.SeekableJournal], applies them to registered
// [projection.Projection] handlers, tracks per-projection checkpoints via
// [event.CheckpointStore], and handles failures with automatic restart and
// exponential backoff. Poison messages that exceed a configurable retry
// threshold are captured to a [DeadLetterStore] and the checkpoint advances,
// preventing a single bad event from blocking the entire stream.
//
// This is the "last loop every consumer rewrites" — per-projection goroutines,
// crash auto-restart, health/liveness exposure, and graceful drain on shutdown —
// wrapped in a single embeddable component that stays a library, not a framework.
//
// # Quick Start
//
//	store := storage.NewMemoryStore()
//	journal := store // implements event.SeekableJournal
//	cpStore := storage.NewMemoryCheckpointStore()
//
//	host, _ := projectionhost.New(journal, cpStore)
//	host.Register(&MyProjection{})
//	host.Register(&AnotherProjection{})
//
//	go host.Start(ctx)
//	// ... process events ...
//	host.Stop() // graceful drain
//
// # Observability
//
// The host integrates with OpenTelemetry automatically — when a global tracer
// provider is configured, each event handler invocation creates a span
// (projectionhost.handle_event) with projection name, event type, and event ID
// attributes. The drain phase wraps in a projectionhost.drain span. No explicit
// tracer setup is needed; without a provider, all spans are no-ops.
//
// For metrics, implement [MetricsRecorder] and wire it via [WithMetrics]. The
// host reports event processing, errors, dead-letter captures, worker restarts,
// worker failures (terminal), and checkpoint lag.
//
// For failure alerting, use [WithOnFailed] to register a callback invoked when
// a worker exhausts its restart budget and transitions to [WorkerFailed]. This
// is a terminal state — the worker will not restart without [Host.Reset].
//
//	// Alert when a projection permanently fails
//	host, _ := projectionhost.New(journal, cpStore,
//	    projectionhost.WithOnFailed(func(name, err string) {
//	        alerting.Notify(ctx, fmt.Sprintf("projection %q failed: %s", name, err))
//	    }),
//	)
//
// # Rebuilding Projections
//
// [Host.Reset] drops the checkpoint for a named projection and, if the
// projection implements [Resettable], calls its Reset method to clear
// read-model state. After Reset, the next Start replays all events from the
// beginning of the journal.
//
//	type UserProjection struct { /* ... */ }
//	func (p *UserProjection) Reset(ctx context.Context) error {
//	    return p.store.DeleteAll(ctx) // clear read model
//	}
//
//	// After fixing a handler bug:
//	host.Stop()
//	host.Reset(ctx, "users") // drops checkpoint + calls Resettable.Reset
//	host.Start(ctx)           // replays from zero
//
// # Live Event Processing
//
// By default, the host is a batch-drainer: workers exit after catching up.
// Use [WithSubscriber] to enable live event processing after journal drain.
// Events seen during replay are deduplicated via a bounded [dedup.Ring] to
// prevent double-processing at the replay→live boundary.
//
// # Graceful Shutdown
//
// [Host.Stop] marks all active workers as [WorkerDraining], signals them to
// stop, and waits for in-flight events to complete. The shutdown timeout
// defaults to 30s and is configurable via [WithShutdownTimeout].
package projectionhost
