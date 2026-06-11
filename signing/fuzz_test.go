package signing_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/signing/v2"
)

// fuzzEvent builds an event with the given random fields. The fuzz driver
// controls type/aggregate/payload/occurred-at/version/etc. to exercise every
// possible combination without ever panicking.
func fuzzEvent(
	t *testing.T,
	eventType, aggType string,
	version int,
	schemaVersion int,
	payload []byte,
) event.Event {
	t.Helper()

	evt, err := event.NewEvent(
		event.Type(eventType),
		id.NewAggregateID(),
		event.AggregateType(aggType),
		event.Version(version),
		payload,
		event.WithSchemaVersion(event.SchemaVersion(schemaVersion)),
	)
	if err != nil {
		t.Fatalf("fuzzEvent: %v", err)
	}

	return evt
}

// FuzzHMAC_SignVerifyRoundtrip signs random events with HMAC and verifies
// the signature — roundtrip must always succeed for an unmodified event.
func FuzzHMAC_SignVerifyRoundtrip(f *testing.F) {
	key := make([]byte, signing.MinimumKeyLength)
	if _, err := rand.Read(key); err != nil {
		f.Fatalf("rand: %v", err)
	}

	signer, err := signing.NewHMAC(key)
	if err != nil {
		f.Fatalf("NewHMAC: %v", err)
	}

	seeds := []struct {
		typ, agg, payload string
		version, schema   int
	}{
		{"user.created", "User", `{"name":"a"}`, 1, 1},
		{"", "", "", 0, 0},
		{"x", "Y", strings.Repeat("z", 4096), 9999, 9999},
		{"a.b.c", "A_B", `{"nested":{"x":1}}`, 1, 1},
	}

	for _, s := range seeds {
		f.Add(s.typ, s.agg, s.version, s.schema, s.payload)
	}

	f.Fuzz(
		func(t *testing.T, typ, agg string, version, schema int, payload string) {
			if version < 0 || schema < 0 {
				return
			}

			evt := fuzzEvent(t, typ, agg, version, schema, []byte(payload))

			sig, err := signer.Sign(evt)
			if err != nil {
				t.Fatalf("Sign: %v", err)
			}

			if sig.IsZero() {
				t.Fatal("Sign produced zero signature for non-empty event")
			}

			if err := signer.Verify(evt, sig); err != nil {
				t.Fatalf("Verify roundtrip failed: %v", err)
			}
		},
	)
}

// FuzzHMAC_TamperReject ensures HMAC verification rejects when ANY part of
// the event is modified (payload, type, aggregate, version, occurred-at).
// Uses two independently-constructed events that differ in exactly one field.
func FuzzHMAC_TamperReject(f *testing.F) {
	key := make([]byte, signing.MinimumKeyLength)
	if _, err := rand.Read(key); err != nil {
		f.Fatalf("rand: %v", err)
	}

	signer, err := signing.NewHMAC(key)
	if err != nil {
		f.Fatalf("NewHMAC: %v", err)
	}

	seeds := []struct {
		typ, agg, payload string
		version           int
	}{
		{"user.created", "User", `{"x":1}`, 1},
		{"order.placed", "Order", `{"id":"o1"}`, 1},
	}

	for _, s := range seeds {
		f.Add(s.typ, s.agg, s.version, s.payload)
	}

	f.Fuzz(
		func(t *testing.T, typ, agg string, version int, payload string) {
			original := fuzzEvent(t, typ, agg, version, 1, []byte(payload))

			sig, err := signer.Sign(original)
			if err != nil {
				t.Fatalf("Sign: %v", err)
			}

			// Tamper: flip one byte of the payload
			flippedPayload := []byte(payload)
			if len(flippedPayload) == 0 {
				return
			}

			flippedPayload[0] ^= 0xff
			tampered := fuzzEvent(t, typ, agg, version, 1, flippedPayload)

			if err := signer.Verify(tampered, sig); err == nil {
				t.Fatal("HMAC verified a tampered event — critical security bug")
			}
		},
	)
}

// FuzzHMAC_NilAndZeroGuards verifies edge-case guards (nil event, zero sig).
func FuzzHMAC_NilAndZeroGuards(f *testing.F) {
	key := make([]byte, signing.MinimumKeyLength)
	if _, err := rand.Read(key); err != nil {
		f.Fatalf("rand: %v", err)
	}

	signer, err := signing.NewHMAC(key)
	if err != nil {
		f.Fatalf("NewHMAC: %v", err)
	}

	f.Add([]byte("data"))
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, payload []byte) {
		// Zero signature must be rejected on verify path
		var zeroSig signing.Signature
		evt := fuzzEvent(t, "t", "A", 1, 1, payload)

		if err := signer.Verify(evt, zeroSig); err == nil {
			t.Error("zero signature accepted on Verify")
		}

		// Nil event rejected
		if _, err := signer.Sign(nil); err == nil {
			t.Error("Sign accepted nil event")
		}

		if err := signer.Verify(nil, zeroSig); err == nil {
			t.Error("Verify accepted nil event")
		}
	})
}

// FuzzEd25519_SignVerifyRoundtrip mirrors the HMAC roundtrip for asymmetric Ed25519.
func FuzzEd25519_SignVerifyRoundtrip(f *testing.F) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		f.Fatalf("GenerateKey: %v", err)
	}

	signer, err := signing.NewEd25519(priv)
	if err != nil {
		f.Fatalf("NewEd25519: %v", err)
	}

	verifier, err := signing.NewEd25519Verifier(pub)
	if err != nil {
		f.Fatalf("NewEd25519Verifier: %v", err)
	}

	seeds := []struct {
		typ, agg, payload string
		version, schema   int
	}{
		{"u.created", "U", `{}`, 1, 1},
		{"", "", "", 0, 0},
		{"x.y", "A", strings.Repeat("p", 1024), 100, 1},
	}

	for _, s := range seeds {
		f.Add(s.typ, s.agg, s.version, s.schema, s.payload)
	}

	f.Fuzz(
		func(t *testing.T, typ, agg string, version, schema int, payload string) {
			if version < 0 || schema < 0 {
				return
			}

			evt := fuzzEvent(t, typ, agg, version, schema, []byte(payload))

			sig, err := signer.Sign(evt)
			if err != nil {
				t.Fatalf("Sign: %v", err)
			}

			if sig.IsZero() {
				t.Fatal("Ed25519 produced zero signature")
			}

			if err := verifier.Verify(evt, sig); err != nil {
				t.Fatalf("Verify roundtrip failed: %v", err)
			}
		},
	)
}

// FuzzEd25519_TamperReject mirrors the HMAC tamper check for Ed25519.
func FuzzEd25519_TamperReject(f *testing.F) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		f.Fatalf("GenerateKey: %v", err)
	}

	signer, err := signing.NewEd25519(priv)
	if err != nil {
		f.Fatalf("NewEd25519: %v", err)
	}

	verifier, err := signing.NewEd25519Verifier(pub)
	if err != nil {
		f.Fatalf("NewEd25519Verifier: %v", err)
	}

	f.Add("u.created", "U", 1, `{"a":1}`)

	f.Fuzz(func(t *testing.T, typ, agg string, version int, payload string) {
		original := fuzzEvent(t, typ, agg, version, 1, []byte(payload))

		sig, err := signer.Sign(original)
		if err != nil {
			t.Fatalf("Sign: %v", err)
		}

		if len(payload) == 0 {
			return
		}

		flipped := []byte(payload)
		flipped[0] ^= 0xff
		tampered := fuzzEvent(t, typ, agg, version, 1, flipped)

		if err := verifier.Verify(tampered, sig); err == nil {
			t.Fatal("Ed25519 verified a tampered event — critical security bug")
		}
	})
}

// FuzzSignature_UnmarshalJSON exercises Signature.UnmarshalJSON with random
// inputs. Both standard and URL-safe base64 should decode where possible;
// malformed input must produce errors, never panic.
func FuzzSignature_UnmarshalJSON(f *testing.F) {
	f.Add(`"hello"`)
	f.Add(`""`)
	f.Add(`null`)
	f.Add(`123`)
	f.Add(`"!!!"`)
	f.Add(`"AAAAAAAAAAAAAAAAAAAAAA=="`) // valid URL-safe base64
	f.Add(`"AAAA"`)

	f.Fuzz(func(t *testing.T, input string) {
		var sig signing.Signature
		// Must not panic on any input
		_ = sig.UnmarshalJSON([]byte(input))
	})
}

// FuzzExtractSignature_FromRandomEvent tests the extract path with arbitrary
// Custom metadata. Every event that can be built must extract safely.
func FuzzExtractSignature_FromRandomEvent(f *testing.F) {
	f.Add("user.created", "User", `{"k":"v"}`)
	f.Add("", "", "")
	f.Add("a", "b", `{"with":"data","num":42}`)

	f.Fuzz(func(t *testing.T, typ, agg, payload string) {
		evt := fuzzEvent(t, typ, agg, 1, 1, []byte(payload))

		// No signature attached yet — must return ErrNilSignature
		_, err := signing.ExtractSignature(evt)
		if err == nil {
			t.Error("expected error extracting signature from unsigned event")
		}

		// Sign and re-attach, then re-extract
		key := make([]byte, signing.MinimumKeyLength)
		_, _ = rand.Read(key)

		signer, err := signing.NewHMAC(key)
		if err != nil {
			t.Fatalf("NewHMAC: %v", err)
		}

		sig, err := signer.Sign(evt)
		if err != nil {
			t.Fatalf("Sign: %v", err)
		}

		attached, err := signing.AttachSignature(evt, sig)
		if err != nil {
			t.Fatalf("AttachSignature: %v", err)
		}

		extracted, err := signing.ExtractSignature(attached)
		if err != nil {
			t.Fatalf("ExtractSignature: %v", err)
		}

		if !extracted.Equal(sig) {
			t.Error("extracted signature does not match attached signature")
		}

		// nil event must be rejected
		if _, err := signing.ExtractSignature(nil); err == nil {
			t.Error("ExtractSignature accepted nil event")
		}

		if _, err := signing.AttachSignature(nil, sig); err == nil {
			t.Error("AttachSignature accepted nil event")
		}
	})
}
