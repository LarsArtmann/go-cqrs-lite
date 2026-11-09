package metaengine

import (
	"context"
	"sync"
)

// snapshotEntry stores one serialized aggregate state at a version.
type snapshotEntry struct {
	version int64
	data    []byte
}

// memorySnapshotBackend implements SnapshotBackend for the memory engine.
// Snapshots are stored in-process maps, lost when the engine closes.
type memorySnapshotBackend struct {
	mu   sync.Mutex
	data map[string]map[string]snapshotEntry // collection → streamID → entry
}

func newMemorySnapshotBackend() *memorySnapshotBackend {
	return &memorySnapshotBackend{
		data: make(map[string]map[string]snapshotEntry),
	}
}

// NewMemorySnapshotBackend creates an in-memory SnapshotBackend for testing.
// Each instance has isolated data (no shared global state).
func NewMemorySnapshotBackend() SnapshotBackend {
	return newMemorySnapshotBackend()
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
		return nil, 0, ErrNotFound
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
		return nil, 0, ErrNotFound
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
