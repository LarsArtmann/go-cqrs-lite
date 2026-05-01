package event

import (
	"context"
	"io"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// Snapshot represents a point-in-time capture of aggregate state.
type Snapshot struct {
	AggregateID   id.AggregateID
	AggregateType AggregateType
	Version       Version
	State         []byte
	CreatedAt     time.Time
}

// SnapshotStore persists and retrieves aggregate snapshots.
// All implementations must support lifecycle management via io.Closer.
type SnapshotStore interface {
	io.Closer

	// Save persists a snapshot for an aggregate at a specific version.
	Save(ctx context.Context, snapshot Snapshot) error

	// Load retrieves the latest snapshot for an aggregate.
	Load(
		ctx context.Context,
		aggregateType AggregateType,
		aggregateID id.AggregateID,
	) (*Snapshot, error)

	// LoadAtVersion retrieves a snapshot at or before the given version.
	LoadAtVersion(
		ctx context.Context,
		aggregateType AggregateType,
		aggregateID id.AggregateID,
		version Version,
	) (*Snapshot, error)

	Deleter
}
