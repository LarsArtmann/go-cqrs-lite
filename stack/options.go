package stack

import (
	"io"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/kv/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v3"
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

// WithJournal sets the cross-aggregate event reader (projection replay).
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
