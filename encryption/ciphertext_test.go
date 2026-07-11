package encryption_test

import (
	"encoding/json"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/encryption/v4"
)

func TestCiphertext_ZeroValue(t *testing.T) {
	t.Parallel()

	var ct encryption.Ciphertext

	if !ct.IsZero() {
		t.Error("empty ciphertext should be zero")
	}
}

func TestCiphertext_BytesReturnsClone(t *testing.T) {
	t.Parallel()

	ct := encryption.Ciphertext([]byte{1, 2, 3})
	b := ct.Bytes()
	b[0] = 99

	if ct[0] == 99 {
		t.Error("Bytes() should return a clone, not a reference")
	}
}

func TestCiphertext_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	original := encryption.Ciphertext([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15})

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}

	var decoded encryption.Ciphertext
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if !original.Equal(decoded) {
		t.Errorf("decoded = %v, want %v", decoded, original)
	}
}

func TestCiphertext_Equal(t *testing.T) {
	t.Parallel()

	firstCT := encryption.Ciphertext([]byte{1, 2, 3})
	secondCT := encryption.Ciphertext([]byte{1, 2, 3})
	thirdCT := encryption.Ciphertext([]byte{4, 5, 6})

	if !firstCT.Equal(secondCT) {
		t.Error("equal ciphertexts should be equal")
	}

	if firstCT.Equal(thirdCT) {
		t.Error("different ciphertexts should not be equal")
	}
}
