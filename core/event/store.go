package event

import (
	"context"
	"io"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// Deleter interface for deleting aggregate data.
type Deleter interface {
	Delete(ctx context.Context, aggregateType AggregateType, aggregateID id.AggregateID) error
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

	Deleter
}
