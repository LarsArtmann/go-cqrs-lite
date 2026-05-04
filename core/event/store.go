package event

import (
	"context"
	"io"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// GlobalLoader loads all events across all aggregates, ordered by occurrence.
// Implementations return events sorted by OccurredAt for deterministic replay.
// This is the core interface for projection replay.
type GlobalLoader interface {
	LoadAll(ctx context.Context) ([]Event, error)
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
//	    return ts.SaveWithOutbox(ctx, aggType, aggID, events, ver, outbox)
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
		outbox Outbox,
	) error
}
