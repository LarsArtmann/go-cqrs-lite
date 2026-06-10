package encryption_test

import (
	"encoding/json"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/encryption/v2"
)

func TestCiphertext_NonceAndData(t *testing.T) {
	t.Parallel()

	nonce := make([]byte, 12)
	for i := range nonce {
		nonce[i] = byte(i)
	}

	data := []byte("encrypted-content-here")

	ct := make(encryption.Ciphertext, 0, len(nonce)+len(data))
	ct = append(ct, nonce...)
	ct = append(ct, data...)

	extractedNonce := ct.Nonce()
	if len(extractedNonce) != 12 {
		t.Fatalf("nonce length = %d, want 12", len(extractedNonce))
	}

	for i, b := range extractedNonce {
		if b != byte(i) {
			t.Errorf("nonce[%d] = %d, want %d", i, b, byte(i))
		}
	}

	extractedData := ct.Data()
	if string(extractedData) != string(data) {
		t.Errorf("data = %q, want %q", extractedData, data)
	}
}

func TestCiphertext_ZeroValue(t *testing.T) {
	t.Parallel()

	var ct encryption.Ciphertext

	if !ct.IsZero() {
		t.Error("empty ciphertext should be zero")
	}

	if ct.Nonce() != nil {
		t.Error("nonce of zero ciphertext should be nil")
	}

	if ct.Data() != nil {
		t.Error("data of zero ciphertext should be nil")
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

	a := encryption.Ciphertext([]byte{1, 2, 3})
	b := encryption.Ciphertext([]byte{1, 2, 3})
	c := encryption.Ciphertext([]byte{4, 5, 6})

	if !a.Equal(b) {
		t.Error("equal ciphertexts should be equal")
	}

	if a.Equal(c) {
		t.Error("different ciphertexts should not be equal")
	}
}
