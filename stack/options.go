package stack

import (
	"io"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

// Option configures a [Bundle]. It is a field setter with no error return,
// matching the codebase convention (event.Option, snapshot.Option).
//
// Options that hand the Bundle an [io.Closer] also register it for
// [Bundle.Close] via [Bundle.registerCloser]; the deduplication in Close
// makes double-registration harmless.
type Option func(*Bundle)

// ── Events: segregated ──

// WithEventSink sets only the write side of event persistence.
// Use this when a deployment needs asymmetric event stores (e.g. a fast
// append-only log for writes, a separate read replica for loads).
func WithEventSink(s event.EventSink) Option {
	return func(b *Bundle) {
		b.EventSink = s
		b.registerCloser(s)
	}
}

// WithEventSource sets only the read side of event persistence.
func WithEventSource(s event.EventSource) Option {
	return func(b *Bundle) {
		b.EventSource = s
		b.registerCloser(s)
	}
}

// WithJournal sets the cross-stream event reader (projection replay).
func WithJournal(j event.Journal) Option {
	return func(b *Bundle) {
		b.Journal = j
	}
}

// WithSeekableJournal sets the position-based event reader.
func WithSeekableJournal(j event.SeekableJournal) Option {
	return func(b *Bundle) {
		b.SeekableJournal = j
	}
}

// WithBackwardsSource sets the reverse-order event loader.
func WithBackwardsSource(s event.BackwardsSource) Option {
	return func(b *Bundle) {
		b.BackwardsSource = s
	}
}

// ── Events: pub/sub ──

// WithPublisher sets the event publisher (write side of the bus).
func WithPublisher(p event.Publisher) Option {
	return func(b *Bundle) {
		b.Publisher = p
	}
}

// WithSubscriber sets the event subscriber (read side of the bus).
func WithSubscriber(s event.Subscriber) Option {
	return func(b *Bundle) {
		b.Subscriber = s
	}
}

// ── Events: composite conveniences ──

// WithEventStore sets EventSink, EventSource, and (if the store supports it)
// Journal and SeekableJournal from a single composite event.Store.
// This is the common case: one store backs the whole event capability set.
func WithEventStore(s event.Store) Option {
	return func(b *Bundle) {
		b.EventSink = s
		b.EventSource = s
		b.registerCloser(s)

		if j, ok := s.(event.Journal); ok {
			b.Journal = j
		}

		if sj, ok := s.(event.SeekableJournal); ok {
			b.SeekableJournal = sj
		}

		if bs, ok := s.(event.BackwardsSource); ok {
			b.BackwardsSource = bs
		}
	}
}

// WithBus sets Publisher and Subscriber from a single composite event.Bus,
// and registers it for Close.
func WithBus(bus event.Bus) Option {
	return func(b *Bundle) {
		b.Publisher = bus
		b.Subscriber = bus
		b.registerCloser(bus)
	}
}

// ── Commands ──

// WithCommandSink sets only the write side of command persistence.
func WithCommandSink(s command.CommandSink) Option {
	return func(b *Bundle) {
		b.CommandSink = s
		b.registerCloser(s)
	}
}

// WithCommandSource sets only the read side of command persistence.
func WithCommandSource(s command.CommandSource) Option {
	return func(b *Bundle) {
		b.CommandSource = s
		b.registerCloser(s)
	}
}

// WithCommandStore sets CommandSink, CommandSource, and (if supported)
// CommandJournal and SeekableCommandJournal from a single composite
// command.Store.
func WithCommandStore(s command.Store) Option {
	return func(b *Bundle) {
		b.CommandSink = s
		b.CommandSource = s
		b.registerCloser(s)

		if j, ok := s.(command.CommandJournal); ok {
			b.CommandJournal = j
		}

		if sj, ok := s.(command.SeekableCommandJournal); ok {
			b.SeekableCommandJournal = sj
		}
	}
}

// ── Queries ──

// WithQuerySink sets only the write side of query persistence.
func WithQuerySink(s query.QuerySink) Option {
	return func(b *Bundle) {
		b.QuerySink = s
		b.registerCloser(s)
	}
}

// WithQuerySource sets only the read side of query persistence.
func WithQuerySource(s query.QuerySource) Option {
	return func(b *Bundle) {
		b.QuerySource = s
		b.registerCloser(s)
	}
}

// WithQueryStore sets QuerySink, QuerySource, and (if supported)
// QueryJournal and SeekableQueryJournal from a single composite query.Store.
func WithQueryStore(s query.QueryStore) Option {
	return func(b *Bundle) {
		b.QuerySink = s
		b.QuerySource = s
		b.registerCloser(s)

		if j, ok := s.(query.QueryJournal); ok {
			b.QueryJournal = j
		}

		if sj, ok := s.(query.SeekableQueryJournal); ok {
			b.SeekableQueryJournal = sj
		}
	}
}

// ── Snapshots, checkpoints, read models ──

// WithSnapshotStore sets the snapshot store and registers it for Close.
func WithSnapshotStore(s snapshot.SnapshotStore) Option {
	return func(b *Bundle) {
		b.SnapshotStore = s
		b.registerCloser(s)
	}
}

// WithCheckpointStore sets the projection-checkpoint store and registers it
// for Close.
func WithCheckpointStore(s event.CheckpointStore) Option {
	return func(b *Bundle) {
		b.CheckpointStore = s
		b.registerCloser(s)
	}
}

// WithReadModels sets the read-model backend (a [kv.Store], which is
// an alias for [kv.Store]) and registers it for Close.
//
// The deployer chooses the backend (kv.MemStore, pebble.KVAdapter, or a
// future SQL adapter); the application accesses it via the typed
// [ReadModel] accessor.
func WithReadModels(backend kv.Store) Option {
	return func(b *Bundle) {
		b.ReadModels = backend
		b.registerCloser(backend)
	}
}

// WithCloser registers an arbitrary [io.Closer] for lifecycle management
// without setting any capability field. Presets use this for resources that
// must be closed on [Bundle.Close] but are not themselves a capability (e.g.
// a *sql.DB that backs multiple stores via disjoint access patterns).
//
// Registering the same closer more than once is harmless — Close
// deduplicates by pointer.
func WithCloser(c io.Closer) Option {
	return func(b *Bundle) {
		b.registerCloser(c)
	}
}

// WithDrainer registers a [Drainer] that [Bundle.GracefulClose] will call
// BEFORE closing resources. Use this for event subscribers, projection runners,
// and routers that need to finish in-flight work before connections close.
// If d also implements [io.Closer], register it with [WithCloser] too.
func WithDrainer(d Drainer) Option {
	return func(b *Bundle) {
		b.drainers = append(b.drainers, d)
	}
}

// WithDatabase stores the underlying database handle (e.g. *sql.DB) for
// SQL-backed bundles. The handle is type-erased to `any` so the core stack
// package does not import database/sql. Preset-specific SQLViewModel
// constructors type-assert it back to *sql.DB.
//
// Presets call this automatically; consumers typically do not.
func WithDatabase(db any) Option {
	return func(b *Bundle) { b.db = db }
}

// WithDefaultCodec sets the fallback Codec for [ReadModel] and
// [NewMaterialize] accessors when the caller passes nil. This lets a deployer
// adopt CBOR across all read models in one call instead of passing
// codec.CBORCodec{} to every accessor individually.
//
//	bundle, _ := sqlite.New(dsn, stack.WithDefaultCodec(codec.CBORCodec{}))
//	store, _ := stack.ReadModel[Todo, TodoID](bundle, nil) // uses CBOR
//
// Event payload encoding is NOT affected — events are encoded at creation
// time via event.New(event.WithCodec(c)). For event-level codec adoption use
// [WithEventCodec] or set [event.DefaultCodec] at program startup.
func WithDefaultCodec(c codec.Codec) Option {
	return func(b *Bundle) {
		if c != nil {
			b.defaultCodec = c
		}
	}
}

// WithEventCodec sets the codec used for event payload creation via [event.New].
// It stores the codec on the Bundle and exposes it via [Bundle.EventCodec] so
// that consumers can pass it to [event.WithCodec] in their decide functions.
//
// This does NOT mutate [event.DefaultCodec] — consumers who want a process-wide
// default should set [event.DefaultCodec] directly at program startup. Use this
// option when you prefer explicit, per-Bundle wiring:
//
//	bundle, _ := sqlite.New(dsn, stack.WithEventCodec(codec.CBORCodec{}))
//	// In your decide function:
//	evt, _ := event.New("user.created", id, "User", v,
//	    payload, event.WithCodec(bundle.EventCodec()))
//
// WithEventCodec also sets the read-model default (like [WithDefaultCodec]),
// so a single option adopts CBOR for both events and read models.
func WithEventCodec(c codec.Codec) Option {
	return func(b *Bundle) {
		if c != nil {
			b.eventCodec = c
			b.defaultCodec = c
		}
	}
}

// WithDiskSize registers a function that reports the on-disk database size in
// bytes. Disk-backed presets (e.g. Pebble) use this to provide precise disk
// metrics without filesystem walks. When unset, [Bundle.DiskSize] returns -1,
// signaling callers (like benchkit) to fall back to filesystem measurement.
func WithDiskSize(fn func() int64) Option {
	return func(b *Bundle) { b.diskSizeFn = fn }
}

// DiskSize returns the on-disk database size in bytes, or -1 if no disk-size
// reporter is registered. Backends that can report precise disk usage
// (e.g. Pebble via pebble.DB.DiskUsage) register this at construction time
// via [WithDiskSize].
func (b *Bundle) DiskSize() int64 {
	if b.diskSizeFn == nil {
		return -1
	}

	return b.diskSizeFn()
}
