package event

import (
	"context"
	"io"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id"
)

// Snapshot represents a point-in-time capture of aggregate state.
type Snapshot struct {
	AggregateID   id.AggregateID
	AggregateType AggregateType
	Version       Version
	State         []byte
	CreatedAt     time.Time
}

// SnapshotSink is the write side of snapshot persistence.
// Saves and deletes snapshots, never reads.
type SnapshotSink interface {
	io.Closer

	// Save persists a snapshot for an aggregate at a specific version.
	Save(ctx context.Context, snapshot Snapshot) error

	// Delete removes the snapshot for an aggregate.
	Delete(ctx context.Context, ref AggregateRef) error
}

// SnapshotSource is the read side of snapshot persistence.
// Loads snapshots, never writes.
type SnapshotSource interface {
	io.Closer

	// Load retrieves the latest snapshot for an aggregate.
	Load(
		ctx context.Context,
		ref AggregateRef,
	) (*Snapshot, error)

	// LoadAtVersion retrieves a snapshot at or before the given version.
	LoadAtVersion(
		ctx context.Context,
		ref AggregateRef,
		version Version,
	) (*Snapshot, error)
}

// SnapshotStore is the composite of SnapshotSink + SnapshotSource.
// All existing implementations satisfy SnapshotStore.
type SnapshotStore interface {
	SnapshotSink
	SnapshotSource
}
