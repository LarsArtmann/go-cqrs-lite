package encryption_test

import (
	"crypto/rand"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/encryption/v4"
)

func TestCodec_RoundTrip(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, _ := encryption.NewXChaCha20Poly1305(key)
	codec := encryption.NewCodec(codec.JSONCodec{}, enc)

	type User struct {
		Name string `json:"name"`
		SSN  string `json:"ssn"`
	}

	original := User{Name: "Alice", SSN: "123-45-6789"}

	data, err := codec.Encode(original)
	if err != nil {
		t.Fatal(err)
	}

	var decoded User
	if err := codec.Decode(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded != original {
		t.Errorf("decoded = %+v, want %+v", decoded, original)
	}
}

func TestCodec_Encoding(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, _ := encryption.NewAES256GCM(key)
	c := encryption.NewCodec(codec.JSONCodec{}, enc)

	if c.Encoding() != "encrypted" {
		t.Errorf("Encoding() = %q, want %q", c.Encoding(), "encrypted")
	}
}

func TestCodec_ProducesCiphertext(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, _ := encryption.NewXChaCha20Poly1305(key)
	c := encryption.NewCodec(codec.JSONCodec{}, enc)

	data, err := c.Encode(map[string]string{"secret": "data"})
	if err != nil {
		t.Fatal(err)
	}

	if len(data) < 24+16 {
		t.Error("encoded data should contain nonce + auth tag")
	}
}

func TestCodec_WrongKeyFails(t *testing.T) {
	t.Parallel()

	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	_, _ = rand.Read(key1)
	_, _ = rand.Read(key2)

	enc1, _ := encryption.NewXChaCha20Poly1305(key1)
	enc2, _ := encryption.NewXChaCha20Poly1305(key2)

	encodeCodec := encryption.NewCodec(codec.JSONCodec{}, enc1)
	decodeCodec := encryption.NewCodec(codec.JSONCodec{}, enc2)

	data, err := encodeCodec.Encode(map[string]string{"secret": "data"})
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]string
	err = decodeCodec.Decode(data, &result)
	if err == nil {
		t.Error("expected error decoding with wrong key")
	}
}

func TestCodec_EmptyPayload(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, _ := encryption.NewAES256GCM(key)
	c := encryption.NewCodec(codec.JSONCodec{}, enc)

	type Empty struct{}

	data, err := c.Encode(Empty{})
	if err != nil {
		t.Fatal(err)
	}

	var decoded Empty
	if err := c.Decode(data, &decoded); err != nil {
		t.Fatal(err)
	}
}
