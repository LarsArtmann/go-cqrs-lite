package event

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// SaveFunc is the function signature for EventSink.Save implementations.
type SaveFunc func(
	ctx context.Context,
	ref id.AggregateRef,
	events []Event,
	expectedVersion Version,
) error

// EventSink is the write side of event persistence.
// Appends events, never reads, never deletes.
type EventSink interface {
	// Save appends events with optimistic concurrency check.
	Save(
		ctx context.Context,
		ref id.AggregateRef,
		events []Event,
		expectedVersion Version,
	) error

	// AppendBatch appends without concurrency checks.
	// For bulk imports, event replay, and migrations.
	AppendBatch(
		ctx context.Context,
		ref id.AggregateRef,
		events []Event,
	) error
}

// EventSource is the read side of event persistence.
// Loads events, never writes.
type EventSource interface {
	// Load retrieves all events for an aggregate.
	Load(
		ctx context.Context,
		ref id.AggregateRef,
	) ([]Event, error)

	// LoadFromVersion retrieves events starting after version (exclusive).
	LoadFromVersion(
		ctx context.Context,
		ref id.AggregateRef,
		version Version,
	) ([]Event, error)

	// LoadToVersion retrieves events up to and including maxVersion.
	// Returns ErrAggregateNotFound if no events exist for the aggregate.
	LoadToVersion(
		ctx context.Context,
		ref id.AggregateRef,
		maxVersion Version,
	) ([]Event, error)

	// LoadToTimestamp retrieves events where OccurredAt <= maxTime.
	// Returns ErrAggregateNotFound if no events exist for the aggregate.
	LoadToTimestamp(
		ctx context.Context,
		ref id.AggregateRef,
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

// BackwardsSource loads events in reverse version order (newest first).
// Useful for tail-loading scenarios where only the most recent events are needed.
type BackwardsSource interface {
	EventSource
	LoadBackwards(ctx context.Context, ref id.AggregateRef) ([]Event, error)
}
