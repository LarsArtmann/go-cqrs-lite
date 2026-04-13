package event

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

// Deleter interface for deleting aggregate data.
type Deleter interface {
	Delete(ctx context.Context, aggregateType AggregateType, aggregateID id.AggregateID) error
}

// Store defines the interface for event persistence.
type Store interface {
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

// StreamOptions configures event streaming.
type StreamOptions struct {
	FromVersion   Version
	AggregateType AggregateType
	BatchSize     BatchSize
}

// BatchSize represents the number of items in a batch.
type BatchSize int

// Streamer defines streaming capabilities for event stores.
type Streamer interface {
	Stream(ctx context.Context, opts StreamOptions) (<-chan Event, error)
}
