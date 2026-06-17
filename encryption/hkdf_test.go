package encryption_test

import (
	"bytes"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/encryption/v2"
)

func TestDeriveKey_Deterministic(t *testing.T) {
	t.Parallel()

	master := []byte("super-secret-master-key")

	key1, err := encryption.DeriveKey(master, "tenant:acme", 32)
	if err != nil {
		t.Fatalf("DeriveKey: %v", err)
	}

	key2, err := encryption.DeriveKey(master, "tenant:acme", 32)
	if err != nil {
		t.Fatalf("DeriveKey (2nd): %v", err)
	}

	if !bytes.Equal(key1, key2) {
		t.Error("same master + info must produce the same key")
	}

	if len(key1) != 32 {
		t.Errorf("key length = %d, want 32", len(key1))
	}
}

func TestDeriveKey_DifferentInfoProducesDifferentKeys(t *testing.T) {
	t.Parallel()

	master := []byte("super-secret-master-key")

	key1, err := encryption.DeriveKey(master, "tenant:acme", 32)
	if err != nil {
		t.Fatalf("DeriveKey: %v", err)
	}

	key2, err := encryption.DeriveKey(master, "tenant:globex", 32)
	if err != nil {
		t.Fatalf("DeriveKey (2nd): %v", err)
	}

	if bytes.Equal(key1, key2) {
		t.Error("different info must produce different keys")
	}
}

func TestDeriveKey_DifferentMasterProducesDifferentKeys(t *testing.T) {
	t.Parallel()

	key1, err := encryption.DeriveKey([]byte("master-a"), "ctx", 32)
	if err != nil {
		t.Fatalf("DeriveKey: %v", err)
	}

	key2, err := encryption.DeriveKey([]byte("master-b"), "ctx", 32)
	if err != nil {
		t.Fatalf("DeriveKey (2nd): %v", err)
	}

	if bytes.Equal(key1, key2) {
		t.Error("different master keys must produce different derived keys")
	}
}

func TestDeriveKey_EmptyMasterKey(t *testing.T) {
	t.Parallel()

	_, err := encryption.DeriveKey(nil, "ctx", 32)
	if err == nil {
		t.Fatal("expected error for empty master key")
	}
}

func TestDeriveKey_InvalidLength(t *testing.T) {
	t.Parallel()

	master := []byte("key")

	cases := []int{0, -1, encryption.MaxHKDFKeyLen + 1}
	for _, length := range cases {
		_, err := encryption.DeriveKey(master, "ctx", length)
		if err == nil {
			t.Errorf("expected error for length %d", length)
		}
	}
}

func TestDeriveKey_VariousLengths(t *testing.T) {
	t.Parallel()

	master := []byte("master-key")
	lengths := []int{16, 32, 64}

	for _, length := range lengths {
		key, err := encryption.DeriveKey(master, "ctx", length)
		if err != nil {
			t.Fatalf("DeriveKey(length=%d): %v", length, err)
		}

		if len(key) != length {
			t.Errorf("length %d: got %d bytes", length, len(key))
		}
	}
}
