package event

import (
	"context"
)

// Store defines the interface for event persistence
type Store interface {
	// Save appends events to the aggregate's event stream
	Save(ctx context.Context, aggregateType AggregateType, aggregateID string, events []Event, expectedVersion int) error

	// Load retrieves all events for an aggregate
	Load(ctx context.Context, aggregateType AggregateType, aggregateID string) ([]Event, error)

	// LoadFromVersion retrieves events starting from a specific version
	LoadFromVersion(ctx context.Context, aggregateType AggregateType, aggregateID string, version int) ([]Event, error)

	// Delete removes all events for an aggregate (for testing/snapshots)
	Delete(ctx context.Context, aggregateType AggregateType, aggregateID string) error
}

// StreamOptions configures event streaming
type StreamOptions struct {
	FromVersion   int
	AggregateType AggregateType
	BatchSize     int
}

// Streamer defines streaming capabilities for event stores
type Streamer interface {
	Stream(ctx context.Context, opts StreamOptions) (<-chan Event, error)
}
