package system

import (
	"context"
	"fmt"
	"sync"
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
// Each instance has its own isolated data — no shared global state.
type memorySnapshotBackend struct {
	mu   sync.Mutex
	data map[string]map[string]snapshotEntry // collection → streamID → entry
}

func newMemorySnapshotBackend() *memorySnapshotBackend {
	return &memorySnapshotBackend{
		data: make(map[string]map[string]snapshotEntry),
	}
}

func (m *memorySnapshotBackend) SnapshotSave(
	_ context.Context,
	collection, streamID string,
	version int64,
	data []byte,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.data[collection] == nil {
		m.data[collection] = make(map[string]snapshotEntry)
	}

	m.data[collection][streamID] = snapshotEntry{version: version, data: data}

	return nil
}

func (m *memorySnapshotBackend) SnapshotLoad(
	_ context.Context,
	collection, streamID string,
) ([]byte, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.data[collection][streamID]
	if !ok {
		return nil, 0, fmt.Errorf("snapshot not found for stream %s", streamID)
	}

	return entry.data, entry.version, nil
}

func (m *memorySnapshotBackend) SnapshotLoadAtVersion(
	_ context.Context,
	collection, streamID string,
	maxVersion int64,
) ([]byte, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.data[collection][streamID]
	if !ok || entry.version > maxVersion {
		return nil, 0, fmt.Errorf(
			"snapshot not found for stream %s at version <= %d",
			streamID,
			maxVersion,
		)
	}

	return entry.data, entry.version, nil
}

func (m *memorySnapshotBackend) SnapshotDelete(
	_ context.Context,
	collection, streamID string,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data[collection], streamID)

	return nil
}

// Compile-time assertion.
var _ SnapshotBackend = (*memorySnapshotBackend)(nil)
