package snapshot_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

// transformTestRef is the shared fixture stream for the TransformedStore tests.
func transformTestRef(t *testing.T) id.StreamRef {
	t.Helper()

	return id.NewStreamRef("User", idtest.ParseStreamID(t, "01HK1540X0841Y0A6BSX1VKR95"))
}

// protectTag is a visible at-rest marker: wrapped state is prefixed so tests
// can assert the inner store really saw transformed bytes.
func protectTag(state []byte) ([]byte, error) {
	return append(bytes.Clone([]byte("guarded:")), state...), nil
}

func restoreTag(state []byte) ([]byte, error) {
	rest, ok := bytes.CutPrefix(state, []byte("guarded:"))
	if !ok {
		return nil, errors.New("state not guarded")
	}

	return rest, nil
}

func newTransformedForTest(t *testing.T) (*snapshot.TransformedStore, *fakeStore) {
	t.Helper()

	inner := newFakeStore()
	store, err := snapshot.NewTransformedStore(inner, protectTag, restoreTag)
	if err != nil {
		t.Fatalf("NewTransformedStore: %v", err)
	}

	return store, inner
}

func testSnapshot(t *testing.T) snapshot.Snapshot {
	t.Helper()

	snap, err := snapshot.NewSnapshot(transformTestRef(t), event.Version(5),
		[]byte(`{"hp":42}`), record.EncodingJSON)
	if err != nil {
		t.Fatalf("NewSnapshot: %v", err)
	}

	return snap
}

func TestNewTransformedStore_RejectsNilInputs(t *testing.T) {
	t.Parallel()

	inner := newFakeStore()
	protect, restore := protectTag, restoreTag

	for name, tc := range map[string]struct {
		inner   snapshot.SnapshotStore
		protect func([]byte) ([]byte, error)
		restore func([]byte) ([]byte, error)
	}{
		"nil store":    {nil, protect, restore},
		"nil protect":  {inner, nil, restore},
		"nil restore":  {inner, protect, nil},
		"both nil fns": {inner, nil, nil},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := snapshot.NewTransformedStore(tc.inner, tc.protect, tc.restore)
			var famErr *errorfamily.Error
			if !errors.As(err, &famErr) || famErr.ErrorFamily() != errorfamily.Rejection {
				t.Fatalf("err = %v, want Rejection-family error", err)
			}
		})
	}
}

func TestTransformedStore_RoundTripProtectsAndRestores(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, inner := newTransformedForTest(t)
	snap := testSnapshot(t)

	if err := store.Save(ctx, snap); err != nil {
		t.Fatalf("Save: %v", err)
	}

	persisted := inner.data[inner.key(transformTestRef(t))]
	if persisted == nil {
		t.Fatal("snapshot missing in inner store")
	}

	if !bytes.HasPrefix(persisted.State, []byte("guarded:")) {
		t.Errorf("inner state = %q, want guarded prefix", persisted.State)
	}

	if persisted.Encoding != snap.Encoding {
		t.Errorf(
			"inner Encoding = %s, want %s (routing metadata must survive)",
			persisted.Encoding,
			snap.Encoding,
		)
	}

	if persisted.Version != 5 {
		t.Errorf("inner Version = %d, want 5", persisted.Version)
	}

	loaded, err := store.Load(ctx, transformTestRef(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if !bytes.Equal(loaded.State, []byte(`{"hp":42}`)) {
		t.Errorf("loaded state = %q, want restored original", loaded.State)
	}
}

func TestTransformedStore_LoadAtVersion(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, _ := newTransformedForTest(t)
	snap := testSnapshot(t)

	if err := store.Save(ctx, snap); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.LoadAtVersion(ctx, transformTestRef(t), event.Version(5))
	if err != nil {
		t.Fatalf("LoadAtVersion(5): %v", err)
	}

	if !bytes.Equal(loaded.State, []byte(`{"hp":42}`)) {
		t.Errorf("state = %q, want restored", loaded.State)
	}

	if _, err := store.LoadAtVersion(ctx, transformTestRef(t), event.Version(6)); err == nil {
		t.Fatal("LoadAtVersion(6) succeeded, want error")
	}
}

func TestTransformedStore_RestoreFailureIsCorruption(t *testing.T) {
	t.Parallel()

	inner := newFakeStore()
	store, err := snapshot.NewTransformedStore(inner, protectTag,
		func([]byte) ([]byte, error) { return nil, errors.New("boom") })
	if err != nil {
		t.Fatalf("NewTransformedStore: %v", err)
	}

	if err := store.Save(context.Background(), testSnapshot(t)); err != nil {
		t.Fatalf("Save: %v", err)
	}

	_, err = store.Load(context.Background(), transformTestRef(t))
	var famErr *errorfamily.Error
	if !errors.As(err, &famErr) || famErr.ErrorFamily() != errorfamily.Corruption {
		t.Fatalf("err = %v, want Corruption-family error", err)
	}
}

func TestTransformedStore_ProtectFailureIsInfrastructure(t *testing.T) {
	t.Parallel()

	inner := newFakeStore()
	store, err := snapshot.NewTransformedStore(inner,
		func([]byte) ([]byte, error) { return nil, errors.New("boom") }, restoreTag)
	if err != nil {
		t.Fatalf("NewTransformedStore: %v", err)
	}

	err = store.Save(context.Background(), testSnapshot(t))
	var famErr *errorfamily.Error
	if !errors.As(err, &famErr) || famErr.ErrorFamily() != errorfamily.Infrastructure {
		t.Fatalf("err = %v, want Infrastructure-family error", err)
	}
}

func TestTransformedStore_LoadNotFoundStaysInfrastructure(t *testing.T) {
	t.Parallel()

	store, _ := newTransformedForTest(t)

	_, err := store.Load(context.Background(), transformTestRef(t))
	var famErr *errorfamily.Error
	if !errors.As(err, &famErr) || famErr.ErrorFamily() != errorfamily.Infrastructure {
		t.Fatalf("err = %v, want Infrastructure-family error", err)
	}
}

func TestTransformedStore_DeleteDelegates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, inner := newTransformedForTest(t)
	snap := testSnapshot(t)

	if err := store.Save(ctx, snap); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := store.Delete(ctx, transformTestRef(t)); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, ok := inner.data[inner.key(transformTestRef(t))]; ok {
		t.Fatal("snapshot still present after Delete")
	}
}

func TestTransformedStore_LoadNilResultPassesThrough(t *testing.T) {
	t.Parallel()

	store, _ := newTransformedForTest(t)

	loaded, err := store.Load(context.Background(), id.NewStreamRef("Ghost", id.NewStreamID()))
	if loaded != nil {
		t.Fatalf("loaded = %+v, want nil", loaded)
	}

	if err == nil {
		t.Fatal("Load on missing stream returned nil error, want ErrSnapshotNotFound wrap")
	}
}

// staleTag mirrors the encryption rotation shape: state prefixed with the
// writing key ("k1:" is retired, "k2:" is active). NeedsRewrite reports k1
// state; reencryptTag swaps the prefix to the active key.
func needsRewriteTag(raw []byte) bool {
	return bytes.HasPrefix(raw, []byte("k1:"))
}

func reencryptTag(raw []byte) ([]byte, error) {
	rest, ok := bytes.CutPrefix(raw, []byte("k1:"))
	if !ok {
		return nil, errors.New("state not under retired key k1")
	}

	return append(bytes.Clone([]byte("k2:")), rest...), nil
}

func activeTag(state []byte) ([]byte, error) {
	return append(bytes.Clone([]byte("k2:")), state...), nil
}

func restoreAnyKeyTag(state []byte) ([]byte, error) {
	if rest, ok := bytes.CutPrefix(state, []byte("k2:")); ok {
		return rest, nil
	}

	rest, ok := bytes.CutPrefix(state, []byte("k1:"))
	if !ok {
		return nil, errors.New("state not under any known key")
	}

	return rest, nil
}

// TestNewRewritingTransformedStore_WritesBackStaleState verifies the
// re-encrypt-on-read migration: a stale-encoded snapshot is re-encoded under
// the active transform, persisted through the inner store, and still loads
// correctly — once per snapshot (the second load sees no stale state).
func TestNewRewritingTransformedStore_WritesBackStaleState(t *testing.T) {
	t.Parallel()

	inner := newFakeStore()

	store, err := snapshot.NewRewritingTransformedStore(inner, snapshot.StateTransforms{
		Protect:      activeTag,
		Restore:      restoreAnyKeyTag,
		NeedsRewrite: needsRewriteTag,
		Reencrypt:    reencryptTag,
	})
	if err != nil {
		t.Fatalf("NewRewritingTransformedStore: %v", err)
	}

	ref := transformTestRef(t)

	stale, err := snapshot.NewSnapshot(ref, event.Version(3), []byte("k1:old-state"), record.EncodingJSON)
	if err != nil {
		t.Fatalf("seed stale snapshot: %v", err)
	}

	if err = inner.Save(t.Context(), *stale); err != nil {
		t.Fatalf("seed inner store: %v", err)
	}

	loaded, err := store.Load(t.Context(), ref)
	if err != nil {
		t.Fatalf("load through rewriting store: %v", err)
	}

	if string(loaded.State) != "old-state" {
		t.Errorf("restored state = %q, want %q", loaded.State, "old-state")
	}

	persisted, err := inner.Load(t.Context(), ref)
	if err != nil {
		t.Fatalf("reload raw from inner: %v", err)
	}

	if string(persisted.State) != "k2:old-state" {
		t.Errorf("write-back state = %q, want %q", persisted.State, "k2:old-state")
	}

	if needsRewriteTag(persisted.State) {
		t.Error("persisted state still reported as stale after write-back")
	}

	again, err := store.Load(t.Context(), ref)
	if err != nil {
		t.Fatalf("second load: %v", err)
	}

	if string(again.State) != "old-state" {
		t.Errorf("second restored state = %q, want %q", again.State, "old-state")
	}
}

// TestNewRewritingTransformedStore_RejectsPartialMigration verifies the
// NeedsRewrite/Reencrypt pair is validated up front.
func TestNewRewritingTransformedStore_RejectsPartialMigration(t *testing.T) {
	t.Parallel()

	_, err := snapshot.NewRewritingTransformedStore(newFakeStore(), snapshot.StateTransforms{
		Protect:      activeTag,
		Restore:      restoreAnyKeyTag,
		NeedsRewrite: needsRewriteTag,
	})
	if err == nil {
		t.Fatal("expected error for NeedsRewrite without Reencrypt")
	}
}
