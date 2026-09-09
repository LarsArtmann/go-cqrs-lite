package encryption

import (
	"encoding/base64"
	"testing"

	"pgregory.net/rapid"
)

// v1↔v2 decode-symmetry property: whichever generation's wire form an
// envelope is read from — v2 raw JSON, or the v1 base64url-wrapped JSON —
// UnmarshalEnvelope recovers the same envelope the writer serialized.
// This pins the forward/backward-readability contract: v2 writers can
// always read v1 rows and vice versa, for arbitrary payloads.
func TestUnmarshalEnvelope_V1V2DecodeSymmetry(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		version := rapid.SampledFrom([]string{
			EnvelopeVersionV1,
			EnvelopeVersionV2,
		}).Draw(rt, "version")

		ctBytes := rapid.SliceOf(rapid.Byte()).Draw(rt, "ciphertext")
		algorithm := rapid.SampledFrom([]Algorithm{
			"",
			AES256GCM,
			XChaCha20Poly1305,
		}).Draw(rt, "algorithm")
		keyID := rapid.SampledFrom([]KeyID{"", "key-v1", "key-v2", "tenant-42"}).Draw(rt, "keyID")

		env := Envelope{
			Version:    version,
			Ciphertext: Ciphertext(ctBytes),
			Algorithm:  algorithm,
			KeyID:      keyID,
		}

		v2Wire, err := MarshalEnvelope(env)
		if err != nil {
			rt.Fatalf("MarshalEnvelope: %v", err)
		}

		v1Wire := base64.URLEncoding.EncodeToString([]byte(v2Wire))

		fromV2, err := UnmarshalEnvelope(v2Wire)
		if err != nil {
			rt.Fatalf("UnmarshalEnvelope(v2): %v", err)
		}

		fromV1, err := UnmarshalEnvelope(v1Wire)
		if err != nil {
			rt.Fatalf("UnmarshalEnvelope(v1): %v", err)
		}

		if !envelopesEqual(fromV2, env) {
			rt.Fatalf("v2 decode drift:\n got: %+v\nwant: %+v", fromV2, env)
		}

		if !envelopesEqual(fromV1, env) {
			rt.Fatalf("v1 decode drift:\n got: %+v\nwant: %+v", fromV1, env)
		}
	})
}

// TestUnmarshalEnvelope_DefaultWrittenV2ReadsSymmetric pins the write-side
// default as part of the same contract: an envelope with no explicit version
// is written as v2 and reads back identically through both wire forms.
func TestUnmarshalEnvelope_DefaultWrittenV2ReadsSymmetric(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		ctBytes := rapid.SliceOf(rapid.Byte()).Draw(rt, "ciphertext")

		env := Envelope{Ciphertext: Ciphertext(ctBytes)}

		wire, err := MarshalEnvelope(env)
		if err != nil {
			rt.Fatalf("MarshalEnvelope: %v", err)
		}

		if wire[:9] != `{"v":"v2"` {
			rt.Fatalf("default version should be written as v2, got %s", wire[:16])
		}

		v1Wire := base64.URLEncoding.EncodeToString([]byte(wire))

		fromV1, err := UnmarshalEnvelope(v1Wire)
		if err != nil {
			rt.Fatalf("UnmarshalEnvelope(v1): %v", err)
		}

		want := env
		want.Version = EnvelopeVersionV2

		if !envelopesEqual(fromV1, want) {
			rt.Fatalf("v1 decode drift:\n got: %+v\nwant: %+v", fromV1, want)
		}
	})
}

func envelopesEqual(a, b Envelope) bool {
	return a.Version == b.Version &&
		a.Ciphertext.Equal(b.Ciphertext) &&
		a.Algorithm == b.Algorithm &&
		a.KeyID == b.KeyID
}
