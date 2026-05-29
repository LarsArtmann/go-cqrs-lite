package event

import (
	"context"
	"io"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// EventSink is the write side of event persistence.
// Appends events, never reads, never deletes.
type EventSink interface {
	io.Closer

	// Save appends events with optimistic concurrency check.
	Save(
		ctx context.Context,
		aggregateType AggregateType,
		aggregateID id.AggregateID,
		events []Event,
		expectedVersion Version,
	) error

	// AppendBatch appends without concurrency checks.
	// For bulk imports, event replay, and migrations.
	AppendBatch(
		ctx context.Context,
		aggregateType AggregateType,
		aggregateID id.AggregateID,
		events []Event,
	) error
}

// EventSource is the read side of event persistence.
// Loads events, never writes.
type EventSource interface {
	io.Closer

	// Load retrieves all events for an aggregate.
	Load(
		ctx context.Context,
		aggregateType AggregateType,
		aggregateID id.AggregateID,
	) ([]Event, error)

	// LoadFromVersion retrieves events starting after version (exclusive).
	LoadFromVersion(
		ctx context.Context,
		aggregateType AggregateType,
		aggregateID id.AggregateID,
		version Version,
	) ([]Event, error)

	// LoadToVersion retrieves events up to and including maxVersion.
	// Returns ErrAggregateNotFound if no events exist for the aggregate.
	LoadToVersion(
		ctx context.Context,
		aggregateType AggregateType,
		aggregateID id.AggregateID,
		maxVersion Version,
	) ([]Event, error)

	// LoadToTimestamp retrieves events where OccurredAt <= maxTime.
	// Returns ErrAggregateNotFound if no events exist for the aggregate.
	LoadToTimestamp(
		ctx context.Context,
		aggregateType AggregateType,
		aggregateID id.AggregateID,
		maxTime time.Time,
	) ([]Event, error)
}

// Store is the composite of EventSink + EventSource.
// All existing implementations satisfy Store.
type Store interface {
	EventSink
	EventSource
}

// Journal reads all events across all aggregates, ordered by occurrence.
// "Journal" is the standard event sourcing term for the complete, ordered,
// append-only log of all domain events. This is the core interface for
// projection replay.
type Journal interface {
	// ReadAll retrieves all events across all aggregates, ordered by OccurredAt.
	ReadAll(ctx context.Context) ([]Event, error)
}

// SeekableJournal extends Journal with position-based reading.
// Enables efficient projection catch-up without loading all events into memory.
//
// Position is based on event ID ordering. ULID-based IDs are time-sortable, making
// them suitable for position-based loading. Using non-monotonic IDs may produce
// incorrect results.
type SeekableJournal interface {
	Journal

	// ReadFrom retrieves events ordered by OccurredAt, starting after
	// the given event ID. Returns up to limit events. Pass limit <= 0 for no limit.
	ReadFrom(ctx context.Context, afterEventID id.EventID, limit int) ([]Event, error)
}

// GlobalLoader loads all events across all aggregates, ordered by occurrence.
// Implementations return events sorted by OccurredAt for deterministic replay.
// This is the core interface for projection replay.
//
// Deprecated: use Journal instead.
type GlobalLoader interface {
	LoadAll(ctx context.Context) ([]Event, error)
}

// PositionalLoader extends GlobalLoader with position-based loading.
//
// Deprecated: use SeekableJournal instead.
type PositionalLoader interface {
	GlobalLoader

	// LoadAllFromPosition retrieves events ordered by OccurredAt, starting after
	// the given event ID. Returns up to limit events. Pass limit <= 0 for no limit.
	LoadAllFromPosition(ctx context.Context, afterEventID id.EventID, limit int) ([]Event, error)
}

// BackwardsSource loads events in reverse version order (newest first).
// Useful for tail-loading scenarios where only the most recent events are needed.
type BackwardsSource interface {
	EventSource
	LoadBackwards(ctx context.Context, aggType AggregateType, aggID id.AggregateID) ([]Event, error)
}

// BackwardsLoader loads events in reverse version order (newest first).
//
// Deprecated: use BackwardsSource instead.
type BackwardsLoader = BackwardsSource

// TransactionalSink extends EventSink with atomic save+outbox append.
// Implementations MUST guarantee that SaveWithOutbox persists events
// AND appends to the outbox within a single database transaction.
// If either operation fails, the entire transaction rolls back.
//
// Repositories detect this via type assertion and prefer it over the
// two-step Save+Append approach when available:
//
//	if ts, ok := sink.(TransactionalSink); ok {
//	    return ts.SaveWithOutbox(ctx, aggType, aggID, events, ver)
//	}
type TransactionalSink interface {
	EventSink

	// SaveWithOutbox atomically persists events and appends them to the outbox.
	SaveWithOutbox(
		ctx context.Context,
		aggregateType AggregateType,
		aggregateID id.AggregateID,
		events []Event,
		expectedVersion Version,
	) error
}

// TransactionalStore extends Store with atomic save+outbox append.
//
// Deprecated: use TransactionalSink instead.
type TransactionalStore interface {
	TransactionalSink
	EventSource
}
