package event

import (
	"context"
	"io"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// GlobalLoader loads all events across all aggregates, ordered by occurrence.
// Implementations return events sorted by OccurredAt for deterministic replay.
// This is the core interface for projection replay.
type GlobalLoader interface {
	LoadAll(ctx context.Context) ([]Event, error)
}

// PositionalLoader extends GlobalLoader with position-based loading.
// Implementations load events ordered by OccurredAt, starting after the given event ID.
// This enables efficient projection catch-up without loading all events into memory.
//
// Position is based on event ID ordering. ULID-based IDs are time-sortable, making
// them suitable for position-based loading. Using non-monotonic IDs may produce
// incorrect results.
type PositionalLoader interface {
	GlobalLoader

	// LoadAllFromPosition retrieves events ordered by OccurredAt, starting after
	// the given event ID. Returns up to limit events. Pass limit <= 0 for no limit.
	LoadAllFromPosition(ctx context.Context, afterEventID id.EventID, limit int) ([]Event, error)
}

// Store defines the interface for event persistence.
// All implementations must support lifecycle management via io.Closer.
type Store interface {
	io.Closer

	// Save appends events to the aggregate's event stream
	Save(
		ctx context.Context,
		aggregateType AggregateType,
		aggregateID id.AggregateID,
		events []Event,
		expectedVersion Version,
	) error

	// AppendBatch appends events without optimistic concurrency checks.
	// Useful for bulk imports, event replay, and migrations.
	AppendBatch(
		ctx context.Context,
		aggregateType AggregateType,
		aggregateID id.AggregateID,
		events []Event,
	) error

	// Load retrieves all events for an aggregate
	Load(
		ctx context.Context,
		aggregateType AggregateType,
		aggregateID id.AggregateID,
	) ([]Event, error)

	// LoadFromVersion retrieves events starting from a specific version
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

	// Delete removes all events for an aggregate
	Delete(ctx context.Context, aggregateType AggregateType, aggregateID id.AggregateID) error
}

// TransactionalStore extends Store with atomic save+outbox append.
// Implementations MUST guarantee that SaveWithOutbox persists events
// AND appends to the outbox within a single database transaction.
// If either operation fails, the entire transaction rolls back.
//
// Repositories detect this via type assertion and prefer it over the
// two-step Save+Append approach when available:
//
//	if ts, ok := store.(TransactionalStore); ok {
//	    return ts.SaveWithOutbox(ctx, aggType, aggID, events, ver)
//	}
type TransactionalStore interface {
	Store

	// SaveWithOutbox atomically persists events and appends them to the outbox.
	SaveWithOutbox(
		ctx context.Context,
		aggregateType AggregateType,
		aggregateID id.AggregateID,
		events []Event,
		expectedVersion Version,
	) error
}
