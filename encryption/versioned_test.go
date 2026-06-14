package encryption

import (
	"testing"
)

func TestWrapUnwrapCiphertext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		alg  Algorithm
		raw  Ciphertext
	}{
		{"AES-256-GCM", AES256GCM, Ciphertext{0xAA, 0xBB, 0xCC, 0xDD}},
		{"XChaCha20-Poly1305", XChaCha20Poly1305, Ciphertext{0x01, 0x02, 0x03}},
		{"empty payload", AES256GCM, Ciphertext{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			wrapped, err := WrapCiphertext(tt.raw, tt.alg)
			if err != nil {
				t.Fatalf("WrapCiphertext: %v", err)
			}

			if len(wrapped) != len(tt.raw)+2 {
				t.Errorf("wrapped length = %d, want %d", len(wrapped), len(tt.raw)+2)
			}

			alg, unwrapped, err := UnwrapCiphertext(wrapped)
			if err != nil {
				t.Fatalf("UnwrapCiphertext: %v", err)
			}

			if alg != tt.alg {
				t.Errorf("algorithm = %q, want %q", alg, tt.alg)
			}

			if string(unwrapped) != string(tt.raw) {
				t.Errorf("raw = %x, want %x", unwrapped, tt.raw)
			}
		})
	}
}

func TestUnwrapCiphertext_BackwardCompat(t *testing.T) {
	t.Parallel()

	// Raw ciphertext without version header should be returned as-is
	raw := Ciphertext{0xAA, 0xBB, 0xCC}

	alg, unwrapped, err := UnwrapCiphertext(raw)
	if err != nil {
		t.Fatalf("UnwrapCiphertext: %v", err)
	}

	if alg != "" {
		t.Errorf("algorithm = %q, want empty (unversioned)", alg)
	}

	if string(unwrapped) != string(raw) {
		t.Errorf("raw = %x, want %x", unwrapped, raw)
	}
}

func TestWrapCiphertext_UnknownAlgorithm(t *testing.T) {
	t.Parallel()

	_, err := WrapCiphertext(Ciphertext{0x01}, "unknown-alg")
	if err == nil {
		t.Fatal("expected error for unknown algorithm")
	}
}
