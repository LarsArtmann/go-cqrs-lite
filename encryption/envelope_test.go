package encryption

import (
	"encoding/base64"
	"testing"
)

func TestMarshalUnmarshalEnvelope_Roundtrip(t *testing.T) {
	t.Parallel()

	env := Envelope{
		Version:    EnvelopeVersionV1,
		Ciphertext: Ciphertext("encrypted-data"),
		Algorithm:  XChaCha20Poly1305,
		KeyID:      "key-v1",
	}

	encoded, err := MarshalEnvelope(env)
	if err != nil {
		t.Fatalf("MarshalEnvelope: %v", err)
	}

	got, err := UnmarshalEnvelope(encoded)
	if err != nil {
		t.Fatalf("UnmarshalEnvelope: %v", err)
	}

	if got.Version != env.Version {
		t.Errorf("Version: got %q, want %q", got.Version, env.Version)
	}

	if !got.Ciphertext.Equal(env.Ciphertext) {
		t.Errorf("Ciphertext: got %q, want %q", got.Ciphertext, env.Ciphertext)
	}

	if got.Algorithm != env.Algorithm {
		t.Errorf("Algorithm: got %q, want %q", got.Algorithm, env.Algorithm)
	}

	if got.KeyID != env.KeyID {
		t.Errorf("KeyID: got %q, want %q", got.KeyID, env.KeyID)
	}
}

func TestMarshalEnvelope_DefaultVersion(t *testing.T) {
	t.Parallel()

	env := Envelope{
		Ciphertext: Ciphertext("data"),
	}

	encoded, err := MarshalEnvelope(env)
	if err != nil {
		t.Fatalf("MarshalEnvelope: %v", err)
	}

	got, err := UnmarshalEnvelope(encoded)
	if err != nil {
		t.Fatalf("UnmarshalEnvelope: %v", err)
	}

	if got.Version != EnvelopeVersionV1 {
		t.Errorf("Version: got %q, want %q", got.Version, EnvelopeVersionV1)
	}
}

func TestUnmarshalEnvelope_InvalidBase64(t *testing.T) {
	t.Parallel()

	_, err := UnmarshalEnvelope("not-valid-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestUnmarshalEnvelope_InvalidJSON(t *testing.T) {
	t.Parallel()

	encoded := base64.URLEncoding.EncodeToString([]byte(`not json`))
	_, err := UnmarshalEnvelope(encoded)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestUnmarshalEnvelope_MissingVersion(t *testing.T) {
	t.Parallel()

	encoded := base64.URLEncoding.EncodeToString(
		[]byte(`{"ct":"ZGF0YQ==","alg":"xchacha20-poly1305"}`),
	)
	_, err := UnmarshalEnvelope(encoded)
	if err == nil {
		t.Fatal("expected error for missing version")
	}
}
