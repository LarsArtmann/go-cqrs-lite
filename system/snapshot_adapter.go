package system

import (
	"context"
	"errors"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

// SnapshotAdapter wraps a [metaengine.SnapshotBackend] as a
// [snapshot.SnapshotStore]. This bridges the engine-level snapshot interface
// (bytes + version) with the decider-level interface (typed Snapshot struct).
//
// The adapter uses a fixed collection name (typically "snapshots") and maps
// each stream ref to (collection, streamKey). Timestamps are not persisted by
// the backend — Load sets CreatedAt to time.Time{} (zero value).
type SnapshotAdapter struct {
	backend    metaengine.SnapshotBackend
	collection string
}

// NewSnapshotAdapter creates a snapshot.SnapshotStore from a SnapshotBackend.
func NewSnapshotAdapter(backend metaengine.SnapshotBackend, collection string) *SnapshotAdapter {
	return &SnapshotAdapter{backend: backend, collection: collection}
}

// Compile-time assertion.
var _ snapshot.SnapshotStore = (*SnapshotAdapter)(nil)

func (a *SnapshotAdapter) Save(ctx context.Context, snap snapshot.Snapshot) error {
	return a.backend.SnapshotSave(ctx, a.collection, snap.StreamID.String(), int64(snap.Version), snap.State)
}

func (a *SnapshotAdapter) Delete(ctx context.Context, ref id.StreamRef) error {
	return a.backend.SnapshotDelete(ctx, a.collection, ref.StreamKey())
}

func (a *SnapshotAdapter) Load(ctx context.Context, ref id.StreamRef) (*snapshot.Snapshot, error) {
	data, version, err := a.backend.SnapshotLoad(ctx, a.collection, ref.StreamKey())
	if err != nil {
		if errors.Is(err, metaengine.ErrNotFound) {
			return nil, nil //nolint:nilnil // no snapshot is not an error
		}

		return nil, err
	}

	return &snapshot.Snapshot{
		StreamID:   ref.ID,
		StreamType: ref.Type,
		Version:    event.Version(version),
		State:      data,
		CreatedAt:  time.Time{},
	}, nil
}

func (a *SnapshotAdapter) LoadAtVersion(
	ctx context.Context,
	ref id.StreamRef,
	version event.Version,
) (*snapshot.Snapshot, error) {
	data, actualVersion, err := a.backend.SnapshotLoadAtVersion(
		ctx, a.collection, ref.StreamKey(), int64(version),
	)
	if err != nil {
		if errors.Is(err, metaengine.ErrNotFound) {
			return nil, nil //nolint:nilnil // no snapshot is not an error
		}

		return nil, err
	}

	return &snapshot.Snapshot{
		StreamID:   ref.ID,
		StreamType: ref.Type,
		Version:    event.Version(actualVersion),
		State:      data,
		CreatedAt:  time.Time{},
	}, nil
}
