package decider

import (
	"github.com/larsartmann/go-codec"
	flightrecorder "github.com/larsartmann/go-flightrecorder"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

// RepositoryOption configures a Repository.
type RepositoryOption[State any] func(*Repository[State])

// WithSnapshotStore enables snapshot support for the repository.
func WithSnapshotStore[State any](store snapshot.SnapshotStore) RepositoryOption[State] {
	return func(r *Repository[State]) {
		r.snapshotStore = store
	}
}

// WithCodec sets the codec for snapshot serialization.
// Required when using WithSnapshotStore — the codec encodes State to bytes
// and decodes bytes back to State.
func WithCodec[State any](codec codec.Codec) RepositoryOption[State] {
	return func(r *Repository[State]) {
		r.codec = codec
	}
}

// WithSnapshotStrategy sets the strategy for automatic snapshotting.
// When set, Execute checks the strategy after persisting events and
// creates a snapshot if the strategy triggers.
func WithSnapshotStrategy[State any](strategy snapshot.SnapshotStrategy) RepositoryOption[State] {
	return func(r *Repository[State]) {
		r.snapshotStrategy = strategy
	}
}

// WithSnapshotStateVersion declares the CURRENT shape version of State and
// opts the repository into snapshot shape guarding: every snapshot this
// repository writes is stamped with the version, and a load whose stored
// stamp differs discards the snapshot and rebuilds state from the journal
// (the journal is the truth; a stale-shaped snapshot is never loaded as if
// it were current). Discards are counted via
// [Repository.SnapshotShapeDiscards].
//
// Unstamped snapshots (written before the stamp existed, or by writers
// without a declaration) are always accepted — absent = accept. An empty
// version string disables both the stamp and the guard.
//
// The version is caller-owned: bump it whenever State's shape changes in a
// way old bytes must not be decoded into (field renames, type changes).
func WithSnapshotStateVersion[State any](version string) RepositoryOption[State] {
	return func(r *Repository[State]) {
		r.snapshotStateShape = version
	}
}

// WithEnricher sets a context enricher that automatically enriches events
// with metadata derived from context (correlation IDs, user IDs, etc.).
func WithEnricher[State any](enricher event.ContextEnricher) RepositoryOption[State] {
	return func(r *Repository[State]) {
		r.enricher = enricher
	}
}

// WithLoadCoalescing enables or disables singleflight load coalescing.
// Enabled by default — concurrent loads of the same stream share a
// single store.Load call. Pass false to disable when the store already
// provides its own caching or deduplication layer.
func WithLoadCoalescing[State any](enabled bool) RepositoryOption[State] {
	return func(r *Repository[State]) {
		r.loadCoalescing = enabled
	}
}

// WithStateCache enables an in-memory hot-state cache for folded stream
// state. On a cache hit, Load fetches only events since the cached version
// instead of replaying the full history — O(new events) instead of O(total).
//
// The cache is process-local, LRU-bounded, and best-effort: misses and
// staleness fall back to the normal load path. Execute updates the cache
// after every successful write, so it stays fresh for the current process.
//
//	repo, _ := decider.NewRepository(store, bus, d,
//	    decider.WithStateCache[MyState](decider.NewStateCache[MyState](256)))
//
// Profile before enabling: for small streams the fold cost may be
// negligible, and the cache adds map+mutex overhead on every Load/Execute.
func WithStateCache[State any](cache StateCache[State]) RepositoryOption[State] {
	return func(r *Repository[State]) {
		r.stateCache = cache
	}
}

// WithFlightRecorder captures a trace snapshot when an Execute call
// triggers the condition (e.g., slow load/save, command rejection, store
// error). The snapshot is captured asynchronously to avoid blocking the
// request path.
//
// If trigger is nil, [flightrecorder.OnError] is used (capture on any
// Execute error). Use [flightrecorder.OnLatency] or
// [flightrecorder.OnErrorOrLatency] for latency-based diagnosis:
//
//	repo, _ := decider.NewRepository(store, bus, d,
//	    decider.WithFlightRecorder[MyState](recorder,
//	        flightrecorder.OnErrorOrLatency(200*time.Millisecond)))
//
// The recorder must be started separately. Only one flight recorder may
// be active per process.
func WithFlightRecorder[State any](
	recorder *flightrecorder.Recorder,
	trigger flightrecorder.TriggerFunc,
) RepositoryOption[State] {
	return func(r *Repository[State]) {
		r.flightRecorder = recorder
		r.flightRecorderTrigger = trigger
	}
}
