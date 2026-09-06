package snapshot

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// StateTransforms protects and restores serialized snapshot state. Protect
// runs before the inner store persists a snapshot; Restore runs after a load.
// Both directions leave routing metadata (StreamID, Version, Encoding,
// CreatedAt) untouched so stores keep indexing and looking snapshots up.
//
// The struct is deliberately plain functions: providers in other modules —
// e.g. encryption — satisfy it structurally, so snapshot never imports them
// and never grows a dependency for a concern it does not own (the same
// transform-composition stance as ADR-0126 for event stores).
type StateTransforms struct {
	Protect func(state []byte) ([]byte, error)
	Restore func(state []byte) ([]byte, error)
}

// TransformedStore wraps a SnapshotStore with state-level transforms. Use it
// for at-rest protection (encryption, compression) of snapshot payloads
// without hand-writing a wrapper per backend.
type TransformedStore struct {
	inner      SnapshotStore
	transforms StateTransforms
}

// NewTransformedStore protects snapshot state with protect before persisting
// it and restores it with restore after a load. Both functions are required —
// a store that protects writes but cannot restore reads (or vice versa) would
// silently corrupt data instead of failing fast. The functions are unnamed
// signatures on purpose: providers in other modules (e.g. encryption) return
// structurally identical funcs, so snapshot never imports them and never
// grows a dependency for a concern it does not own (the transform-composition
// stance of ADR-0126 for event stores).
func NewTransformedStore(
	inner SnapshotStore,
	protect func(state []byte) ([]byte, error),
	restore func(state []byte) ([]byte, error),
) (*TransformedStore, error) {
	if inner == nil {
		return nil, errorfamily.NewRejection(
			"snapshot.transform_nil_store",
			"inner snapshot store must not be nil",
		)
	}

	if protect == nil || restore == nil {
		return nil, errorfamily.NewRejection(
			"snapshot.transform_incomplete",
			"both protect and restore transforms are required",
		)
	}

	return &TransformedStore{
		inner: inner,
		transforms: StateTransforms{
			Protect: protect,
			Restore: restore,
		},
	}, nil
}

// Save protects the snapshot state, then persists through the inner store.
func (s *TransformedStore) Save(ctx context.Context, snapshot Snapshot) error {
	protected, err := s.transforms.Protect(snapshot.State)
	if err != nil {
		return errorfamily.Wrapf(
			err,
			errorfamily.Infrastructure,
			"snapshot.transform_protect",
			"protect snapshot state for %s",
			snapshot.StreamID,
		)
	}

	snapshot.State = protected

	if err = s.inner.Save(ctx, snapshot); err != nil {
		return errorfamily.Wrapf(err, errorfamily.Infrastructure, "snapshot.transformed_save",
			"save %s v%d", snapshot.StreamID, snapshot.Version)
	}

	return nil
}

// Delete removes the snapshot through the inner store.
func (s *TransformedStore) Delete(ctx context.Context, ref id.StreamRef) error {
	if err := s.inner.Delete(ctx, ref); err != nil {
		return errorfamily.Wrapf(err, errorfamily.Infrastructure, "snapshot.transformed_delete",
			"delete snapshot %s", ref)
	}

	return nil
}

// Load fetches the snapshot from the inner store and restores its state.
func (s *TransformedStore) Load(
	ctx context.Context,
	ref id.StreamRef,
) (*Snapshot, error) {
	snap, err := s.inner.Load(ctx, ref)
	if err != nil {
		return nil, errorfamily.Wrapf(err, errorfamily.Infrastructure, "snapshot.transformed_load",
			"load %s", ref)
	}

	return s.restore(snap)
}

// LoadAtVersion fetches a specific version from the inner store and restores
// its state.
func (s *TransformedStore) LoadAtVersion(
	ctx context.Context,
	ref id.StreamRef,
	version event.Version,
) (*Snapshot, error) {
	snap, err := s.inner.LoadAtVersion(ctx, ref, version)
	if err != nil {
		return nil, errorfamily.Wrapf(err, errorfamily.Infrastructure,
			"snapshot.transformed_load_at_version", "load %s v%d", ref, version)
	}

	return s.restore(snap)
}

func (s *TransformedStore) restore(snap *Snapshot) (*Snapshot, error) {
	if snap == nil {
		return nil, nil //nolint:nilnil // defensive: pass a nil inner result through
	}

	restored, err := s.transforms.Restore(snap.State)
	if err != nil {
		return nil, errorfamily.Wrapf(
			err,
			errorfamily.Corruption,
			"snapshot.transform_restore",
			"restore snapshot state for %s",
			snap.StreamID,
		)
	}

	snap.State = restored

	return snap, nil
}
