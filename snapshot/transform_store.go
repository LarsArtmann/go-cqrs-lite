package snapshot

import (
	"context"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
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

	// NeedsRewrite reports whether raw stored state should be re-written in
	// place (e.g. it was encrypted under a retired key). Optional; consult
	// only together with Reencrypt.
	NeedsRewrite func(raw []byte) bool
	// Reencrypt maps raw stored state to raw state under the current
	// transform (e.g. decrypt with the old key, encrypt with the active
	// key). Optional; consult only together with NeedsRewrite.
	Reencrypt func(raw []byte) ([]byte, error)
}

// TransformedStore wraps a SnapshotStore with state-level transforms. Use it
// for at-rest protection (encryption, compression) of snapshot payloads
// without hand-writing a wrapper per backend.
type TransformedStore struct {
	inner      SnapshotStore
	transforms StateTransforms
}

// NewRewritingTransformedStore is NewTransformedStore for transform sets that
// carry optional migration funcs ([StateTransforms.NeedsRewrite] /
// [StateTransforms.Reencrypt]): loads of stale-encoded snapshots are
// re-encoded and written back through the inner store before being returned,
// so a key rotation converges without a maintenance window. Each load
// re-encodes at most one snapshot, and a failed write-back never fails the
// load — the snapshot is still returned correctly and retried next time.
func NewRewritingTransformedStore(inner SnapshotStore, transforms StateTransforms) (*TransformedStore, error) {
	if (transforms.NeedsRewrite == nil) != (transforms.Reencrypt == nil) {
		return nil, errorfamily.NewRejection(
			"snapshot.transform_migration_partial",
			"NeedsRewrite and Reencrypt must be set together",
		)
	}

	store, err := newTransformedStore(inner, transforms)
	if err != nil {
		return nil, err
	}

	return store, nil
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
	return newTransformedStore(inner, StateTransforms{
		Protect: protect,
		Restore: restore,
	})
}

func newTransformedStore(inner SnapshotStore, transforms StateTransforms) (*TransformedStore, error) {
	if inner == nil {
		return nil, errorfamily.NewRejection(
			"snapshot.transform_nil_store",
			"inner snapshot store must not be nil",
		)
	}

	if transforms.Protect == nil || transforms.Restore == nil {
		return nil, errorfamily.NewRejection(
			"snapshot.transform_incomplete",
			"both protect and restore transforms are required",
		)
	}

	return &TransformedStore{inner: inner, transforms: transforms}, nil
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
// When migration transforms are configured and the stored state is stale
// (NeedsRewrite), it is re-encoded (Reencrypt) and written back before the
// restored snapshot is returned.
func (s *TransformedStore) Load(
	ctx context.Context,
	ref id.StreamRef,
) (*Snapshot, error) {
	snap, err := s.inner.Load(ctx, ref)
	if err != nil {
		return nil, errorfamily.Wrapf(err, errorfamily.Infrastructure, "snapshot.transformed_load",
			"load %s", ref)
	}

	return s.restore(ctx, snap)
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

	return s.restore(ctx, snap)
}

func (s *TransformedStore) restore(ctx context.Context, snap *Snapshot) (*Snapshot, error) {
	if snap == nil {
		return nil, nil //nolint:nilnil // defensive: pass a nil inner result through
	}

	s.rewriteInPlace(ctx, snap)

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

// rewriteInPlace re-encodes a stale-encoded snapshot and writes it back
// through the inner store. Best-effort: any failure keeps the load on the
// original state, so the next load can retry the migration.
func (s *TransformedStore) rewriteInPlace(ctx context.Context, snap *Snapshot) {
	if s.transforms.NeedsRewrite == nil || !s.transforms.NeedsRewrite(snap.State) {
		return
	}

	reencoded, err := s.transforms.Reencrypt(snap.State)
	if err != nil {
		return
	}

	writeBack := *snap
	writeBack.State = reencoded
	if err := s.inner.Save(ctx, writeBack); err != nil {
		return
	}

	snap.State = reencoded
}
