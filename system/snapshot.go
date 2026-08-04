package system

import (
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// SnapshotBackend is an alias for [metaengine.SnapshotBackend].
// Engines implement it. The SnapshotAdapter wraps it as snapshot.SnapshotStore.
//
// This enables decider.LoadAtVersion to use the latest snapshot at or below
// the target version, then replay events from there — the same optimization
// the current decider.Repository provides, but as a first-class engine interface.
type SnapshotBackend = metaengine.SnapshotBackend

// NewMemorySnapshotBackend creates an in-memory SnapshotBackend for testing.
// Each instance has isolated data (no shared global state).
func NewMemorySnapshotBackend() SnapshotBackend {
	return metaengine.NewMemorySnapshotBackend()
}
