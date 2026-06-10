package encryption

import (
	"crypto/rand"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
)

func TestCodecWrapper_GoldenRoundTrip(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, _ := NewAES256GCM(key)
	c := NewCodec(codec.JSONCodec{}, enc)

	type user struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	original := user{Name: "Alice", Age: 30}

	encoded, err := c.Encode(original)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	if len(encoded) == 0 {
		t.Fatal("encoded should not be empty for non-zero payload")
	}

	if c.Encoding() != EncryptionEncoding {
		t.Fatalf("expected %q, got %q", EncryptionEncoding, c.Encoding())
	}

	var decoded user
	if err := c.Decode(encoded, &decoded); err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if decoded.Name != original.Name {
		t.Fatalf("name: got %q, want %q", decoded.Name, original.Name)
	}

	if decoded.Age != original.Age {
		t.Fatalf("age: got %d, want %d", decoded.Age, original.Age)
	}
}

func TestCodecWrapper_GoldenEmptyPayload(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, _ := NewXChaCha20Poly1305(key)
	c := NewCodec(codec.JSONCodec{}, enc)

	encoded, err := c.Encode(nil)
	if err != nil {
		t.Fatalf("Encode nil: %v", err)
	}

	if len(encoded) == 0 {
		t.Fatal("nil payload encodes via JSON codec (produces 'null'), so output should be encrypted")
	}

	var result map[string]any
	if err := c.Decode(encoded, &result); err != nil {
		t.Fatalf("Decode: %v", err)
	}
}

func TestCodecWrapper_GoldenXChaCha20RoundTrip(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, _ := NewXChaCha20Poly1305(key)
	c := NewCodec(codec.JSONCodec{}, enc)

	payload := map[string]string{"secret": "value-12345"}

	encoded, err := c.Encode(payload)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	var decoded map[string]string
	if err := c.Decode(encoded, &decoded); err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if decoded["secret"] != "value-12345" {
		t.Fatalf("expected 'value-12345', got %q", decoded["secret"])
	}
}
