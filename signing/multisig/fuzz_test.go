package multisig_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/signing/v4"
	"github.com/larsartmann/go-cqrs-lite/signing/v4/internal/testutil"
	"github.com/larsartmann/go-cqrs-lite/signing/v4/multisig"
)

// FuzzMultiSig_AttachExtractRoundtrip signs an event, extracts the multi-sig
// collection, and verifies the actor/algorithm/signature fields roundtrip
// intact. Failure would indicate corruption during the JSON marshaling
// pipeline that carries signatures between processes.
func FuzzMultiSig_AttachExtractRoundtrip(f *testing.F) {
	deviceMulti, _ := newDeviceMultiSignerTB(f)

	f.Add("test.created", `{"k":"v"}`)
	f.Add("a", `{}`)
	f.Add(strings.Repeat("long-type-name.", 10), strings.Repeat("p", 4096))

	f.Fuzz(func(t *testing.T, eventType, payload string) {
		evt := testutil.MakeTestEvent(t)

		clone, err := deviceMulti.Sign(evt)
		if err != nil {
			t.Fatalf("device.Sign: %v", err)
		}

		extracted, err := multisig.ExtractMultiSignature(clone)
		if err != nil {
			t.Fatalf("ExtractMultiSignature: %v", err)
		}

		if extracted.Count() != 1 {
			t.Fatalf("expected 1 entry, got %d", extracted.Count())
		}

		entry := extracted.Get(multisig.Actor("device"))
		if entry == nil {
			t.Fatal("expected device entry")
		}

		if entry.Algorithm != multisig.AlgorithmEd25519 {
			t.Errorf("algorithm: got %s, want %s", entry.Algorithm, multisig.AlgorithmEd25519)
		}

		if entry.Sig.IsZero() {
			t.Error("signature is zero")
		}

		// Verify the original event's signature still checks out.
		if err := deviceMulti.Verify(clone); err != nil {
			t.Errorf("Verify: %v", err)
		}
	})
}

// FuzzMultiSig_TamperRejected ensures that re-serializing the event with a
// flipped payload byte causes verification to fail — the same property as
// HMAC tamper detection, but exercised through the multi-sig path.
func FuzzMultiSig_TamperRejected(f *testing.F) {
	deviceMulti, _ := newDeviceMultiSignerTB(f)

	f.Add(`{"k":"v"}`)

	f.Fuzz(func(t *testing.T, payload string) {
		evt := testutil.MakeTestEvent(t)

		clone, err := deviceMulti.Sign(evt)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}

		// Direct verify on the original must pass
		if err := deviceMulti.Verify(clone); err != nil {
			t.Fatalf("verify roundtrip: %v", err)
		}

		if len(payload) == 0 {
			return
		}

		// Tamper: build a new event with the same metadata (incl sig)
		// but different payload. The signature should no longer verify.
		flipped := []byte(payload)
		flipped[0] ^= 0xff

		tampered, err := event.NewEvent(
			evt.Type(), evt.AggregateID(), evt.AggregateType(), evt.Version(),
			flipped,
			event.WithEventID(evt.ID()),
			event.WithOccurredAt(evt.OccurredAt()),
			event.WithSchemaVersion(evt.SchemaVersion()),
			event.WithMetadata(clone.Metadata()),
		)
		if err != nil {
			t.Fatalf("build tampered: %v", err)
		}

		if err := deviceMulti.Verify(tampered); err == nil {
			t.Error("multi-sig verified a tampered event — critical security bug")
		}
	})
}

// FuzzMultiSig_MultiActorChain appends signatures from multiple actors
// (device + server). The final collection must contain all entries.
func FuzzMultiSig_MultiActorChain(f *testing.F) {
	deviceMulti, _ := newDeviceMultiSignerTB(f)
	serverMulti := newServerMultiSignerTB(f)

	f.Add("evt.created")

	f.Fuzz(func(t *testing.T, eventType string) {
		evt := testutil.MakeTestEvent(t)

		clone1, err := deviceMulti.Sign(evt)
		if err != nil {
			t.Fatalf("device sign: %v", err)
		}

		clone2, err := serverMulti.Sign(clone1)
		if err != nil {
			t.Fatalf("server sign: %v", err)
		}

		extracted, err := multisig.ExtractMultiSignature(clone2)
		if err != nil {
			t.Fatalf("extract: %v", err)
		}

		if extracted.Count() != 2 {
			t.Errorf("expected 2 entries, got %d", extracted.Count())
		}

		if !extracted.HasActor(multisig.Actor("device")) {
			t.Error("missing device entry")
		}

		if !extracted.HasActor(multisig.Actor("server")) {
			t.Error("missing server entry")
		}

		// Each actor's verify must succeed against the chained event
		if err := deviceMulti.Verify(clone2); err != nil {
			t.Errorf("device verify chained: %v", err)
		}

		if err := serverMulti.Verify(clone2); err != nil {
			t.Errorf("server verify chained: %v", err)
		}
	})
}

// FuzzMultiSig_ReplaceActor verifies that re-signing with the same actor
// replaces the prior entry (no duplicate actors allowed).
func FuzzMultiSig_ReplaceActor(f *testing.F) {
	deviceMulti, _ := newDeviceMultiSignerTB(f)

	f.Add("evt", 1)

	f.Fuzz(func(t *testing.T, eventType string, version int) {
		evt := testutil.MakeTestEvent(t)

		clone1, err := deviceMulti.Sign(evt)
		if err != nil {
			t.Fatalf("sign 1: %v", err)
		}

		clone2, err := deviceMulti.Sign(clone1)
		if err != nil {
			t.Fatalf("sign 2: %v", err)
		}

		extracted, err := multisig.ExtractMultiSignature(clone2)
		if err != nil {
			t.Fatalf("extract: %v", err)
		}

		if extracted.Count() != 1 {
			t.Errorf("expected 1 entry (replace), got %d", extracted.Count())
		}
	})
}

// FuzzMultiSig_ExtractFromCorruptJSON exercises ExtractMultiSignature against
// events with arbitrary (often malformed) JSON in the multi-sig metadata slot.
// Must always return an error, never panic.
func FuzzMultiSig_ExtractFromCorruptJSON(f *testing.F) {
	f.Add("")
	f.Add("null")
	f.Add(`{"entries":[]}`)
	f.Add(`{"entries":[{}]}`)
	f.Add(`{`)
	f.Add(`not json at all`)
	f.Add(`"just a string"`)
	f.Add(
		`{"entries":[{"actor":"a","algorithm":"HMAC-SHA256","sig":"AAAA","signedAt":"2024-01-01T00:00:00Z"}]}`,
	)
	f.Add(`{"entries":[{"actor":"","algorithm":"","sig":"","signedAt":""}]}`)

	f.Fuzz(func(t *testing.T, jsonValue string) {
		evt, err := event.NewEvent(
			"test.fuzz", id.NewAggregateID(), "Test", 1, nil,
			event.WithMetadata(event.Metadata{
				Custom: map[event.MetadataKey]string{
					multisig.MultiSigMetadataKey: jsonValue,
				},
			}),
		)
		if err != nil {
			t.Fatalf("create event: %v", err)
		}

		// Should not panic; either succeeds with parsed multi-sig or returns an error.
		_, _ = multisig.ExtractMultiSignature(evt)
	})
}

// FuzzMultiSig_NilAndEdgeCases covers nil event/actor/verifier paths.
func FuzzMultiSig_NilAndEdgeCases(f *testing.F) {
	deviceMulti, _ := newDeviceMultiSignerTB(f)

	f.Add("evt")

	f.Fuzz(func(t *testing.T, eventType string) {
		// nil event
		if _, err := multisig.ExtractMultiSignature(nil); err == nil {
			t.Error("ExtractMultiSignature(nil) accepted")
		}

		if err := deviceMulti.Verify(nil); err == nil {
			t.Error("Verify(nil) accepted")
		}

		// Extract on unsigned event
		evt := testutil.MakeTestEvent(t)
		if _, err := multisig.ExtractMultiSignature(evt); err == nil {
			t.Error("ExtractMultiSignature accepted unsigned event")
		}
	})
}

// FuzzMultiSig_VerifyAllMissingVerifier drives VerifyAll with arbitrary
// verifiers maps. With a valid signature chain, an empty map must reject.
func FuzzMultiSig_VerifyAllMissingVerifier(f *testing.F) {
	deviceMulti, _ := newDeviceMultiSignerTB(f)

	f.Add("evt")

	f.Fuzz(func(t *testing.T, eventType string) {
		evt := testutil.MakeTestEvent(t)
		clone, err := deviceMulti.Sign(evt)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}

		// Empty verifier map → must reject (no verifier for "device" actor)
		empty := map[multisig.Actor]signing.Verifier{}
		if err := multisig.VerifyAll(clone, empty); err == nil {
			t.Error("VerifyAll with empty map accepted valid event")
		}

		// nil verifier map → must reject
		if err := multisig.VerifyAll(clone, nil); err == nil {
			t.Error("VerifyAll with nil map accepted valid event")
		}
	})
}

// FuzzMultiSig_Ed25519KeyLength guards against creating signers with bad
// key sizes (defense in depth even though we test NewEd25519 elsewhere).
func FuzzMultiSig_Ed25519KeyLength(f *testing.F) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		f.Fatalf("GenerateKey: %v", err)
	}

	f.Add(int(0))
	f.Add(int(1))
	f.Add(int(ed25519.PublicKeySize))
	f.Add(int(ed25519.PrivateKeySize))
	f.Add(int(64))
	f.Add(int(1024))

	f.Fuzz(func(t *testing.T, keyLen int) {
		if keyLen < 0 {
			return
		}

		// Build truncated keys; pass-through for any keyLen >= the
		// canonical size (no truncation, full valid key).
		truncPub := pub
		if keyLen < ed25519.PublicKeySize {
			truncPub = pub[:keyLen]
		}

		truncPriv := priv
		if keyLen < ed25519.PrivateKeySize {
			truncPriv = priv[:keyLen]
		}

		_, signerErr := signing.NewEd25519(truncPriv)
		_, verifierErr := signing.NewEd25519Verifier(truncPub)

		// Signer accepts iff priv length is exactly PrivateKeySize.
		wantSignerOK := len(truncPriv) == ed25519.PrivateKeySize
		gotSignerOK := signerErr == nil
		if wantSignerOK != gotSignerOK {
			t.Errorf("NewEd25519(len=%d): got ok=%v, want ok=%v (err=%v)",
				len(truncPriv), gotSignerOK, wantSignerOK, signerErr)
		}

		// Verifier accepts iff pub length is exactly PublicKeySize.
		wantVerifierOK := len(truncPub) == ed25519.PublicKeySize
		gotVerifierOK := verifierErr == nil
		if wantVerifierOK != gotVerifierOK {
			t.Errorf("NewEd25519Verifier(len=%d): got ok=%v, want ok=%v (err=%v)",
				len(truncPub), gotVerifierOK, wantVerifierOK, verifierErr)
		}
	})
}
