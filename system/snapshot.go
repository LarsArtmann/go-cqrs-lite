package system

import (
	"context"
	"fmt"
)

// SnapshotBackend is a metaengine-level interface for snapshot storage (D12).
// Engines implement it. The SnapshotAdapter wraps it as snapshot.SnapshotStore.
//
// This enables decider.LoadAtVersion to use the latest snapshot at or below
// the target version, then replay events from there — the same optimization
// the current decider.Repository provides, but as a first-class engine interface.
type SnapshotBackend interface {
	SnapshotSave(ctx context.Context, collection, streamID string, version int64, data []byte) error
	SnapshotLoad(ctx context.Context, collection, streamID string) ([]byte, int64, error)
	SnapshotLoadAtVersion(
		ctx context.Context,
		collection, streamID string,
		maxVersion int64,
	) ([]byte, int64, error)
	SnapshotDelete(ctx context.Context, collection, streamID string) error
}

// ── Memory engine SnapshotBackend implementation ──

type snapshotEntry struct {
	version int64
	data    []byte
}

// memorySnapshotBackend stores snapshots in-memory for testing.
type memorySnapshotBackend struct{}

func newMemorySnapshotBackend() *memorySnapshotBackend {
	return &memorySnapshotBackend{}
}

// snapshotStore holds snapshot data per collection+streamID.
var snapshotStore = make(map[string]map[string]snapshotEntry)

func (m *memorySnapshotBackend) SnapshotSave(
	_ context.Context,
	_ string,
	streamID string,
	version int64,
	data []byte,
) error {
	if snapshotStore["snapshots"] == nil {
		snapshotStore["snapshots"] = make(map[string]snapshotEntry)
	}

	snapshotStore["snapshots"][streamID] = snapshotEntry{version: version, data: data}

	return nil
}

func (m *memorySnapshotBackend) SnapshotLoad(
	_ context.Context,
	_ string,
	streamID string,
) ([]byte, int64, error) {
	entry, ok := snapshotStore["snapshots"][streamID]
	if !ok {
		return nil, 0, fmt.Errorf("snapshot not found for stream %s", streamID)
	}

	return entry.data, entry.version, nil
}

func (m *memorySnapshotBackend) SnapshotLoadAtVersion(
	_ context.Context,
	_ string,
	streamID string,
	maxVersion int64,
) ([]byte, int64, error) {
	entry, ok := snapshotStore["snapshots"][streamID]
	if !ok || entry.version > maxVersion {
		return nil, 0, fmt.Errorf(
			"snapshot not found for stream %s at version <= %d",
			streamID,
			maxVersion,
		)
	}

	return entry.data, entry.version, nil
}

func (m *memorySnapshotBackend) SnapshotDelete(_ context.Context, _ string, streamID string) error {
	delete(snapshotStore["snapshots"], streamID)

	return nil
}
