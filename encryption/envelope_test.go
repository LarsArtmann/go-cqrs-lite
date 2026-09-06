package encryption

import (
	"encoding/base64"
	"strings"
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

	if got.Version != EnvelopeVersionV2 {
		t.Errorf("Version: got %q, want %q", got.Version, EnvelopeVersionV2)
	}
}

func TestUnmarshalEnvelope_InvalidBase64(t *testing.T) {
	t.Parallel()

	_, err := UnmarshalEnvelope("!!!not-valid-base64-or-json!!!")
	if err == nil {
		t.Fatal("expected error for invalid envelope")
	}
}

// TestUnmarshalEnvelope_ReadsV1Base64 pins backward compatibility: v1
// envelopes (base64-wrapped JSON) written before the v2 switch stay
// readable forever.
func TestUnmarshalEnvelope_ReadsV1Base64(t *testing.T) {
	t.Parallel()

	inner := []byte(`{"v":"v1","ct":"ZGF0YQ==","kid":"key-2025"}`)
	v1Envelope := base64.URLEncoding.EncodeToString(inner)

	env, err := UnmarshalEnvelope(v1Envelope)
	if err != nil {
		t.Fatalf("UnmarshalEnvelope (v1): %v", err)
	}

	if env.Version != EnvelopeVersionV1 {
		t.Errorf("Version: got %q, want %q", env.Version, EnvelopeVersionV1)
	}

	if env.KeyID != "key-2025" {
		t.Errorf("KeyID: got %q, want %q", env.KeyID, "key-2025")
	}
}

// TestMarshalEnvelope_OutputIsRawJSON pins the v2 wire format: the output
// must be a JSON object (storable in JSONB columns), not base64.
func TestMarshalEnvelope_OutputIsRawJSON(t *testing.T) {
	t.Parallel()

	encoded, err := MarshalEnvelope(Envelope{Ciphertext: Ciphertext("data")})
	if err != nil {
		t.Fatalf("MarshalEnvelope: %v", err)
	}

	if !strings.HasPrefix(encoded, "{") {
		t.Fatalf("v2 envelope must be raw JSON, got %q", encoded)
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
