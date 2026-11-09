package encryption

import (
	"bytes"
	"testing"
)

func newTestCipher(t *testing.T, key []byte) *aes256gcm {
	t.Helper()

	cipher, err := NewAES256GCM(key)
	if err != nil {
		t.Fatalf("build cipher: %v", err)
	}

	return cipher
}

func TestSnapshotStateCodec_RoundTrip(t *testing.T) {
	t.Parallel()

	key := bytes.Repeat([]byte{7}, KeySize)
	transforms, err := SnapshotStateCodec(newTestCipher(t, key), "key-2026-08")
	if err != nil {
		t.Fatalf("build transforms: %v", err)
	}

	state := []byte(`{"balance":42}`)

	protected, err := transforms.Protect(state)
	if err != nil {
		t.Fatalf("protect: %v", err)
	}

	if bytes.Equal(protected, state) {
		t.Fatal("state must not be stored as plaintext")
	}

	restored, err := transforms.Restore(protected)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}

	if !bytes.Equal(restored, state) {
		t.Fatalf("expected %q, got %q", state, restored)
	}
}

func TestSnapshotStateCodec_RejectsNilCipher(t *testing.T) {
	t.Parallel()

	if _, err := SnapshotStateCodec(nil, "k"); err == nil {
		t.Fatal("expected an error for a nil cipher")
	}
}

func TestRotatingSnapshotStateCodec_ResolveRetiredKey(t *testing.T) {
	t.Parallel()

	oldKey := bytes.Repeat([]byte{1}, KeySize)
	newKey := bytes.Repeat([]byte{2}, KeySize)
	oldCipher := newTestCipher(t, oldKey)

	// Phase 1: snapshots written under the old key.
	oldTransforms, err := SnapshotStateCodec(oldCipher, "key-old")
	if err != nil {
		t.Fatalf("build old transforms: %v", err)
	}

	protected, err := oldTransforms.Protect([]byte(`{"v":1}`))
	if err != nil {
		t.Fatalf("protect: %v", err)
	}

	// Phase 2: the active key rotated; the resolver still knows the old one.
	active := newTestCipher(t, newKey)
	resolver := NewStaticKeyResolver(map[KeyID]Decrypter{
		"key-old": oldCipher,
	})

	rotating, err := RotatingSnapshotStateCodec("key-new", active, resolver)
	if err != nil {
		t.Fatalf("build rotating transforms: %v", err)
	}

	restored, err := rotating.Restore(protected)
	if err != nil {
		t.Fatalf("restore with rotated keys: %v", err)
	}

	if !bytes.Equal(restored, []byte(`{"v":1}`)) {
		t.Fatalf("expected the retired-key snapshot to decrypt, got %q", restored)
	}
}

func TestRotatingSnapshotStateCodec_UnknownKeyIDFails(t *testing.T) {
	t.Parallel()

	oldKey := bytes.Repeat([]byte{1}, KeySize)
	oldCipher := newTestCipher(t, oldKey)

	oldTransforms, err := SnapshotStateCodec(oldCipher, "key-unknown")
	if err != nil {
		t.Fatalf("build transforms: %v", err)
	}

	protected, err := oldTransforms.Protect([]byte(`{"v":1}`))
	if err != nil {
		t.Fatalf("protect: %v", err)
	}

	rotating, err := RotatingSnapshotStateCodec(
		"key-new",
		newTestCipher(t, bytes.Repeat([]byte{3}, KeySize)),
		NewStaticKeyResolver(nil),
	)
	if err != nil {
		t.Fatalf("build rotating transforms: %v", err)
	}

	if _, err = rotating.Restore(protected); err == nil {
		t.Fatal("expected an error when the envelope key ID cannot be resolved")
	}
}

func TestSnapshotStateCodec_NonEnvelopeStateIsCorruption(t *testing.T) {
	t.Parallel()

	transforms, err := SnapshotStateCodec(
		newTestCipher(t, bytes.Repeat([]byte{9}, KeySize)),
		"k",
	)
	if err != nil {
		t.Fatalf("build transforms: %v", err)
	}

	if _, err = transforms.Restore([]byte(`{"aggregateId":"not-encrypted"}`)); err == nil {
		t.Fatal("expected a corruption error for state without an envelope")
	}
}

func TestSnapshotStateCodec_EmptyStatePassesThrough(t *testing.T) {
	t.Parallel()

	transforms, err := SnapshotStateCodec(
		newTestCipher(t, bytes.Repeat([]byte{9}, KeySize)),
		"k",
	)
	if err != nil {
		t.Fatalf("build transforms: %v", err)
	}

	out, err := transforms.Protect(nil)
	if err != nil || out != nil {
		t.Fatalf("empty protect = (%v, %v), want (nil, nil)", out, err)
	}

	out, err = transforms.Restore(nil)
	if err != nil || out != nil {
		t.Fatalf("empty restore = (%v, %v), want (nil, nil)", out, err)
	}
}
