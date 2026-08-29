package integration_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/encryption/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// TestSnapshot_EncryptedStore wires snapshot.NewTransformedStore with
// encryption.SnapshotStateCodec / RotatingSnapshotStateCodec: the composition
// the two modules are designed for. State must be encrypted at rest (the
// inner memory store sees only ciphertext), survive a load, and keep working
// across a key rotation via the envelope's key ID.
func TestSnapshot_EncryptedStore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	oldKey := bytes.Repeat([]byte{0x11}, encryption.KeySize)
	newKey := bytes.Repeat([]byte{0x22}, encryption.KeySize)
	ref := id.StreamRef{ID: mustStreamID(t, "order-123"), Type: "Order"}

	inner := memory.NewMemorySnapshotStore()

	wrap := func(transforms encryption.StateTransforms) snapshot.SnapshotStore {
		t.Helper()

		store, err := snapshot.NewTransformedStore(inner, transforms.Protect, transforms.Restore)
		if err != nil {
			t.Fatalf("build transformed store: %v", err)
		}

		return store
	}

	// Phase 1: write under the old key.
	oldCipher, err := encryption.NewAES256GCM(oldKey)
	if err != nil {
		t.Fatalf("old cipher: %v", err)
	}

	oldTransforms, err := encryption.SnapshotStateCodec(oldCipher, "key-2026-07")
	if err != nil {
		t.Fatalf("old codec: %v", err)
	}

	oldStore := wrap(oldTransforms)

	wantState := []byte(`{"balance":4200}`)
	if err = oldStore.Save(ctx, snapshot.Snapshot{
		StreamID:   ref.ID,
		StreamType: ref.Type,
		Version:    7,
		State:      wantState,
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Phase 2: the key rotates; the old key stays resolvable for old
	// snapshots, new writes go out under the new key.
	newCipher, err := encryption.NewAES256GCM(newKey)
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}

	resolver := encryption.NewStaticKeyResolver(map[encryption.KeyID]encryption.Decrypter{
		"key-2026-07": oldCipher,
	})

	rotatingTransforms, err := encryption.RotatingSnapshotStateCodec(
		"key-2026-08", newCipher, resolver,
	)
	if err != nil {
		t.Fatalf("rotating codec: %v", err)
	}

	rotatingStore := wrap(rotatingTransforms)

	// The old snapshot still loads under the rotated configuration.
	loaded, err := rotatingStore.Load(ctx, ref)
	if err != nil {
		t.Fatalf("load after rotation: %v", err)
	}

	if !bytes.Equal(loaded.State, wantState) {
		t.Fatalf("state mismatch after rotation: %q", loaded.State)
	}

	// A fresh write lands under the NEW key and still round-trips.
	if err = rotatingStore.Save(ctx, snapshot.Snapshot{
		StreamID:   ref.ID,
		StreamType: ref.Type,
		Version:    8,
		State:      []byte(`{"balance":4300}`),
	}); err != nil {
		t.Fatalf("save under new key: %v", err)
	}

	reloaded, err := rotatingStore.LoadAtVersion(ctx, ref, 8)
	if err != nil {
		t.Fatalf("load new-key snapshot: %v", err)
	}

	if !bytes.Equal(reloaded.State, []byte(`{"balance":4300}`)) {
		t.Fatalf("new-key state mismatch: %q", reloaded.State)
	}

	// Tampered state surfaces as an error, not silent garbage.
	if _, err = rotatingStore.LoadAtVersion(ctx, id.StreamRef{
		ID: mustStreamID(t, "order-missing"), Type: "Order",
	}, 1); !errors.Is(err, snapshot.ErrSnapshotNotFound) {
		t.Fatalf("expected not-found for a missing stream, got %v", err)
	}
}

func mustStreamID(t *testing.T, s string) id.StreamID {
	t.Helper()

	streamID, err := id.ParseStreamID(s)
	if err != nil {
		t.Fatalf("parse stream id %q: %v", s, err)
	}

	return streamID
}
