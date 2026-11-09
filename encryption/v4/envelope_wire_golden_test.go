package encryption

import (
	"encoding/base64"
	"testing"
)

// Wire-format goldens for the versioned Envelope. The artifacts below were
// generated once from fixed ciphertext bytes (real encryption is
// nonce-randomized and cannot be pinned) and reviewed byte-for-byte: they
// pin the JSON field names (v/ct/alg/kid), their order (deterministic
// marshal), the URL-safe base64 ciphertext encoding, the v2 raw-JSON shape,
// the v1 base64-wrapped shape, and the omitempty behavior of alg/kid.
// Any change to the wire format MUST be a new envelope version, not an edit
// of these bytes.
func TestEnvelope_WireGolden_V2_Full(t *testing.T) {
	t.Parallel()

	env := Envelope{
		Version:    EnvelopeVersionV2,
		Ciphertext: Ciphertext{0xDE, 0xAD, 0xBE, 0xEF, 0x00, 0x42},
		Algorithm:  XChaCha20Poly1305,
		KeyID:      KeyID("key-v1"),
	}

	const golden = `{"v":"v2","ct":"3q2-7wBC","alg":"xchacha20-poly1305","kid":"key-v1"}`

	wire, err := MarshalEnvelope(env)
	if err != nil {
		t.Fatalf("MarshalEnvelope: %v", err)
	}

	if wire != golden {
		t.Fatalf("v2 wire drift:\n got: %s\nwant: %s", wire, golden)
	}
}

func TestEnvelope_WireGolden_V1_Base64Wrapped(t *testing.T) {
	t.Parallel()

	env := Envelope{
		Version:    EnvelopeVersionV1,
		Ciphertext: Ciphertext{0xDE, 0xAD, 0xBE, 0xEF, 0x00, 0x42},
		Algorithm:  XChaCha20Poly1305,
		KeyID:      KeyID("key-v1"),
	}

	// The v1 generation: the same JSON object, base64url-wrapped as one
	// opaque string (written by pre-v2 writers; still readable).
	const golden = "eyJ2IjoidjEiLCJjdCI6IjNxMi03d0JDIiwiYWxnIjoieGNoYWNoYTIwLXBvbHkxMzA1Iiwia2lkIjoia2V5LXYxIn0="

	inner, err := MarshalEnvelope(env)
	if err != nil {
		t.Fatalf("MarshalEnvelope: %v", err)
	}

	wire := base64.URLEncoding.EncodeToString([]byte(inner))
	if wire != golden {
		t.Fatalf("v1 wire drift:\n got: %s\nwant: %s", wire, golden)
	}

	back, err := UnmarshalEnvelope(wire)
	if err != nil {
		t.Fatalf("UnmarshalEnvelope(v1): %v", err)
	}

	if back.Version != EnvelopeVersionV1 || !back.Ciphertext.Equal(env.Ciphertext) ||
		back.Algorithm != env.Algorithm || back.KeyID != env.KeyID {
		t.Fatalf("v1 roundtrip drift: got %+v, want %+v", back, env)
	}
}

func TestEnvelope_WireGolden_V2_Minimal(t *testing.T) {
	t.Parallel()

	// Zero Version defaults to v2 on write; empty Algorithm/KeyID are
	// omitted from the wire entirely.
	env := Envelope{Ciphertext: Ciphertext{0x01}}

	const golden = `{"v":"v2","ct":"AQ=="}`

	wire, err := MarshalEnvelope(env)
	if err != nil {
		t.Fatalf("MarshalEnvelope: %v", err)
	}

	if wire != golden {
		t.Fatalf("minimal v2 wire drift:\n got: %s\nwant: %s", wire, golden)
	}

	back, err := UnmarshalEnvelope(wire)
	if err != nil {
		t.Fatalf("UnmarshalEnvelope: %v", err)
	}

	if back.Version != EnvelopeVersionV2 || !back.Ciphertext.Equal(env.Ciphertext) {
		t.Fatalf("minimal roundtrip drift: got %+v", back)
	}
}
