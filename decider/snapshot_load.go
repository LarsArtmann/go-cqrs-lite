package decider

import (
	"context"
	"errors"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

func (r *Repository[State]) shouldSnapshot(
	ref id.StreamRef,
	version event.Version,
) bool {
	return snapshot.ShouldSnapshotFor(
		r.snapshotStrategy,
		r.snapshotStore,
		r.codec,
		ref,
		version,
	)
}

func (r *Repository[State]) loadFromSnapshot(
	ctx context.Context,
	ref id.StreamRef,
) (State, event.Version, error) {
	snap, err := r.snapshotStore.Load(ctx, ref)
	if err != nil {
		if !errors.Is(err, snapshot.ErrSnapshotNotFound) {
			var zero State

			return zero, 0, opError(ref, "load snapshot: %w", err)
		}

		return r.loadFromStore(ctx, ref)
	}

	if snap == nil {
		return r.loadFromStore(ctx, ref)
	}

	if r.snapshotShapeMismatch(snap) {
		r.snapshotShapeDiscards.Add(1)

		return r.loadFromStore(ctx, ref)
	}

	var state State

	err = r.codec.Decode(snap.State, &state)
	if err != nil {
		var zero State

		return zero, 0, opError(ref, "decode snapshot: %w", err)
	}

	events, err := r.store.LoadFromVersion(ctx, ref, snap.Version)
	if err != nil {
		var zero State

		return zero, 0, opError(ref, "%w: %w", ErrLoadFailed, err)
	}

	state, err = r.foldEvents(state, events, ref)
	if err != nil {
		var zero State

		return zero, 0, err
	}

	return state, snap.Version.Add(uint(len(events))), nil
}

// snapshotShapeMismatch reports whether the loaded snapshot's state shape is
// known-stale for this repository: a shape version was declared via
// WithSnapshotStateVersion, the snapshot carries a (possibly different)
// stamp, and the two disagree. Unstamped snapshots never mismatch (absent =
// accept); an undeclared repository accepts everything.
func (r *Repository[State]) snapshotShapeMismatch(snap *snapshot.Snapshot) bool {
	return r.snapshotStateShape != "" &&
		snap.StateShape != "" &&
		snap.StateShape != r.snapshotStateShape
}

// SnapshotShapeDiscards returns how many loads discarded a snapshot because
// its StateShape stamp mismatched the declared version and rebuilt state
// from the journal instead. A persistently growing counter means the stored
// snapshots predate a State shape change and the strategy has not re-saved
// at the current version yet.
func (r *Repository[State]) SnapshotShapeDiscards() uint64 {
	return r.snapshotShapeDiscards.Load()
}

// recordRead notifies the snapshot strategy of a read, enabling read-pressure
// strategies like ReadPressure to track load frequency.
func (r *Repository[State]) recordRead(ref id.StreamRef, version event.Version) {
	if r.snapshotStrategy == nil {
		return
	}

	if tracker, ok := r.snapshotStrategy.(snapshot.ReadTracker); ok {
		tracker.RecordRead(ref, version)
	}
}
