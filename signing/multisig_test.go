package signing_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/signing"
)

func TestMultiSignature(t *testing.T) {
	t.Parallel()

	multiSig := signing.MultiSignature{
		Entries: []signing.SignatureEntry{
			{Actor: signing.Actor("device"), Algorithm: signing.AlgorithmEd25519, Sig: []byte("sig1")},
			{Actor: signing.Actor("server"), Algorithm: signing.AlgorithmHMACSHA256, Sig: []byte("sig2")},
		},
	}

	t.Run("count", func(t *testing.T) {
		t.Parallel()
		if got, want := multiSig.Count(), 2; got != want {
			t.Fatalf("Count: got %d, want %d", got, want)
		}
	})

	t.Run("has actor", func(t *testing.T) {
		t.Parallel()
		if !multiSig.HasActor(signing.Actor("device")) {
			t.Fatal("expected HasActor(device) = true")
		}
		if multiSig.HasActor(signing.Actor("gateway")) {
			t.Fatal("expected HasActor(gateway) = false")
		}
	})

	t.Run("get", func(t *testing.T) {
		t.Parallel()
		entry := multiSig.Get(signing.Actor("server"))
		if entry == nil {
			t.Fatal("expected entry for server")
		}
		if entry.Algorithm != signing.AlgorithmHMACSHA256 {
			t.Fatalf("algorithm: got %s, want HMAC-SHA256", entry.Algorithm)
		}
		if multiSig.Get(signing.Actor("gateway")) != nil {
			t.Fatal("expected nil for unknown actor")
		}
	})

	t.Run("actors", func(t *testing.T) {
		t.Parallel()
		actors := multiSig.Actors()
		if len(actors) != 2 {
			t.Fatalf("expected 2 actors, got %d", len(actors))
		}
	})
}

func TestMultiSignatureActors(t *testing.T) {
	t.Parallel()

	multiSig := signing.MultiSignature{
		Entries: []signing.SignatureEntry{
			{Actor: signing.Actor("device"), Algorithm: signing.AlgorithmEd25519, Sig: []byte("a")},
			{Actor: signing.Actor("device"), Algorithm: signing.AlgorithmEd25519, Sig: []byte("b")},
			{Actor: signing.Actor("server"), Algorithm: signing.AlgorithmHMACSHA256, Sig: []byte("c")},
		},
	}

	actors := multiSig.Actors()
	if len(actors) != 2 {
		t.Fatalf("expected 2 unique actors, got %d: %v", len(actors), actors)
	}
}

// newDeviceMultiSigner creates a test MultiSigner for the "device" actor using Ed25519.
func newDeviceMultiSigner(t *testing.T) (*signing.MultiSigner, ed25519.PublicKey) {
	t.Helper()

	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	signer, signerErr := signing.NewEd25519(privKey)
	if signerErr != nil {
		t.Fatalf("create signer: %v", signerErr)
	}

	verifier, verifierErr := signing.NewEd25519Verifier(pubKey)
	if verifierErr != nil {
		t.Fatalf("create verifier: %v", verifierErr)
	}

	deviceMulti, err := signing.NewMultiSigner(signing.Actor("device"), signing.AlgorithmEd25519, signer,
		signing.WithVerifier(verifier))
	if err != nil {
		t.Fatalf("create device multi-signer: %v", err)
	}

	return deviceMulti, pubKey
}

// newServerMultiSigner creates a test MultiSigner for the "server" actor using HMAC.
func newServerMultiSigner(t *testing.T) *signing.MultiSigner {
	t.Helper()

	key := []byte("server-secret-key-thirty-two-by!")
	signer, err := signing.NewHMAC(key)
	if err != nil {
		t.Fatalf("create HMAC signer: %v", err)
	}

	serverMulti, err := signing.NewMultiSigner(signing.Actor("server"), signing.AlgorithmHMACSHA256, signer)
	if err != nil {
		t.Fatalf("create server multi-signer: %v", err)
	}

	return serverMulti
}

func TestMultiSigner_SignAddsActor(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	evt := makeTestEvent(t)

	clone, err := deviceMulti.Sign(evt)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	extracted, err := signing.ExtractMultiSignature(clone)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if extracted.Count() != 1 {
		t.Fatalf("expected 1 entry, got %d", extracted.Count())
	}
	if !extracted.HasActor(signing.Actor("device")) {
		t.Fatal("expected device actor")
	}
}

func TestMultiSigner_MultipleActors(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	serverMulti := newServerMultiSigner(t)
	evt := makeTestEvent(t)

	clone1, err := deviceMulti.Sign(evt)
	if err != nil {
		t.Fatalf("device sign: %v", err)
	}

	clone2, err := serverMulti.Sign(clone1)
	if err != nil {
		t.Fatalf("server sign: %v", err)
	}

	extracted, err := signing.ExtractMultiSignature(clone2)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if extracted.Count() != 2 {
		t.Fatalf("expected 2 entries, got %d", extracted.Count())
	}
	if !extracted.HasActor(signing.Actor("device")) || !extracted.HasActor(signing.Actor("server")) {
		t.Fatal("expected both device and server actors")
	}
}

func TestMultiSigner_ReSignReplaces(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	evt := makeTestEvent(t)

	clone1, err := deviceMulti.Sign(evt)
	if err != nil {
		t.Fatalf("first sign: %v", err)
	}

	extracted1, _ := signing.ExtractMultiSignature(clone1)
	entry1 := extracted1.Get(signing.Actor("device"))
	if entry1 == nil {
		t.Fatal("expected device entry after first sign")
	}

	clone2, err := deviceMulti.Sign(clone1)
	if err != nil {
		t.Fatalf("second sign: %v", err)
	}

	extracted2, _ := signing.ExtractMultiSignature(clone2)
	if extracted2.Count() != 1 {
		t.Fatalf("expected 1 entry after re-sign, got %d", extracted2.Count())
	}

	entry2 := extracted2.Get(signing.Actor("device"))
	if entry2.SignedAt.Equal(entry1.SignedAt) {
		t.Fatal("re-signed entry should have different timestamp")
	}
}

func TestMultiSigner_VerifyOwnSignature(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	evt := makeTestEvent(t)

	clone, _ := deviceMulti.Sign(evt)
	if err := deviceMulti.Verify(clone); err != nil {
		t.Fatalf("verify device: %v", err)
	}
}

func TestMultiSigner_VerifyDualSigned(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	serverMulti := newServerMultiSigner(t)
	evt := makeTestEvent(t)

	clone1, _ := deviceMulti.Sign(evt)
	clone2, _ := serverMulti.Sign(clone1)

	if err := serverMulti.Verify(clone2); err != nil {
		t.Fatalf("verify server: %v", err)
	}

	if err := deviceMulti.Verify(clone2); err != nil {
		t.Fatalf("verify device on dual-signed: %v", err)
	}
}

func TestMultiSigner_VerifyMissingActor(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	serverMulti := newServerMultiSigner(t)
	evt := makeTestEvent(t)

	clone, _ := deviceMulti.Sign(evt)
	if err := serverMulti.Verify(clone); err == nil {
		t.Fatal("expected error verifying missing actor")
	}
}

func TestMultiSigner_VerifyTampered(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	evt := makeTestEvent(t)

	clone, _ := deviceMulti.Sign(evt)
	md := clone.Metadata()

	tampered, _ := event.NewEvent(
		clone.Type(),
		clone.AggregateID(),
		clone.AggregateType(),
		clone.Version(),
		[]byte(`{"tampered":true}`),
		event.WithEventID(clone.ID()),
		event.WithOccurredAt(clone.OccurredAt()),
		event.WithSchemaVersion(clone.SchemaVersion()),
		event.WithMetadata(md),
	)

	if err := deviceMulti.Verify(tampered); err == nil {
		t.Fatal("expected verification to fail for tampered event")
	}
}

func TestMultiSigner_NilEvent(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)

	if _, err := deviceMulti.Sign(nil); err == nil {
		t.Fatal("expected error for nil event on sign")
	}

	if err := deviceMulti.Verify(nil); err == nil {
		t.Fatal("expected error for nil event on verify")
	}
}

func TestExtractMultiSignature(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	evt := makeTestEvent(t)

	t.Run("extract from unsigned event", func(t *testing.T) {
		t.Parallel()
		if _, err := signing.ExtractMultiSignature(evt); err == nil {
			t.Fatal("expected error for unsigned event")
		}
	})

	t.Run("extract from nil event", func(t *testing.T) {
		t.Parallel()
		if _, err := signing.ExtractMultiSignature(nil); err == nil {
			t.Fatal("expected error for nil event")
		}
	})

	t.Run("has multi-sig", func(t *testing.T) {
		t.Parallel()
		if signing.HasMultiSignature(evt) {
			t.Fatal("original event should not have multi-sig")
		}

		clone, _ := deviceMulti.Sign(evt)
		if !signing.HasMultiSignature(clone) {
			t.Fatal("signed event should have multi-sig")
		}
	})
}

func TestMultiSignMiddleware(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)

	t.Run("signs events before publishing", func(t *testing.T) {
		t.Parallel()

		var published []event.Event

		pub := event.PublisherFunc(func(_ context.Context, events ...event.Event) error {
			published = append(published, events...)

			return nil
		})

		mw := signing.MultiSignMiddleware(deviceMulti)
		signedPub := mw(pub)

		evt := makeTestEvent(t)
		if err := signedPub.Publish(context.Background(), evt); err != nil {
			t.Fatalf("publish: %v", err)
		}

		if len(published) != 1 {
			t.Fatalf("expected 1 event, got %d", len(published))
		}

		if !signing.HasMultiSignature(published[0]) {
			t.Fatal("event should have multi-sig after middleware")
		}

		extracted, _ := signing.ExtractMultiSignature(published[0])
		if !extracted.HasActor(signing.Actor("device")) {
			t.Fatal("expected device signature")
		}
	})
}

func TestRequireMultiSigMiddleware(t *testing.T) {
	t.Parallel()

	deviceMulti, devicePubKey := newDeviceMultiSigner(t)
	serverMulti := newServerMultiSigner(t)

	deviceVerifier, _ := signing.NewEd25519Verifier(devicePubKey)
	serverKey := []byte("server-secret-key-thirty-two-by!")
	serverHMAC, _ := signing.NewHMAC(serverKey)

	verifiers := map[signing.Actor]signing.Verifier{
		signing.Actor("device"): deviceVerifier,
		signing.Actor("server"): serverHMAC,
	}

	t.Run("rejects unsigned events", func(t *testing.T) {
		t.Parallel()

		called := false
		handler := func(_ context.Context, _ event.Event) error {
			called = true

			return nil
		}

		mw := signing.RequireMultiSigMiddleware(verifiers)
		wrapped := mw(handler)

		evt := makeTestEvent(t)
		if err := wrapped(context.Background(), evt); err == nil {
			t.Fatal("expected error for unsigned event")
		}
		if called {
			t.Fatal("handler should not have been called")
		}
	})

	t.Run("rejects partially signed events", func(t *testing.T) {
		t.Parallel()

		handler := func(_ context.Context, _ event.Event) error {
			return nil
		}

		mw := signing.RequireMultiSigMiddleware(verifiers)
		wrapped := mw(handler)

		evt := makeTestEvent(t)
		clone, _ := deviceMulti.Sign(evt)

		if err := wrapped(context.Background(), clone); err == nil {
			t.Fatal("expected error for partially signed event")
		}
	})

	t.Run("allows fully signed events", func(t *testing.T) {
		t.Parallel()

		called := false
		handler := func(_ context.Context, _ event.Event) error {
			called = true

			return nil
		}

		mw := signing.RequireMultiSigMiddleware(verifiers)
		wrapped := mw(handler)

		evt := makeTestEvent(t)
		clone1, _ := deviceMulti.Sign(evt)
		clone2, _ := serverMulti.Sign(clone1)

		if err := wrapped(context.Background(), clone2); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Fatal("handler should have been called")
		}
	})
}

func TestMultiSignerEndToEnd(t *testing.T) {
	t.Parallel()

	pubKey, devicePrivKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	deviceSigner, signerErr := signing.NewEd25519(devicePrivKey)
	if signerErr != nil {
		t.Fatalf("create device signer: %v", signerErr)
	}

	deviceVerifier, verifierErr := signing.NewEd25519Verifier(pubKey)
	if verifierErr != nil {
		t.Fatalf("create device verifier: %v", verifierErr)
	}

	serverKey := []byte("server-secret-key-thirty-two-by!")
	serverSigner, hmacErr := signing.NewHMAC(serverKey)
	if hmacErr != nil {
		t.Fatalf("create server signer: %v", hmacErr)
	}

	deviceMulti, err := signing.NewMultiSigner(signing.Actor("device"), signing.AlgorithmEd25519, deviceSigner,
		signing.WithVerifier(deviceVerifier))
	if err != nil {
		t.Fatalf("create device multi-signer: %v", err)
	}

	serverMulti, err := signing.NewMultiSigner(signing.Actor("server"), signing.AlgorithmHMACSHA256, serverSigner)
	if err != nil {
		t.Fatalf("create server multi-signer: %v", err)
	}

	// Step 1: Device creates and signs the event.
	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	deviceEvent, evtErr := event.NewEvent("user.created", aggID, "User", 1, []byte(`{"name":"Alice"}`))
	if evtErr != nil {
		t.Fatalf("create event: %v", evtErr)
	}

	deviceSigned, err := deviceMulti.Sign(deviceEvent)
	if err != nil {
		t.Fatalf("device sign: %v", err)
	}

	// Step 2: Server verifies device's signature, then adds its own.
	if verifyErr := deviceMulti.Verify(deviceSigned); verifyErr != nil {
		t.Fatalf("server verifies device: %v", verifyErr)
	}

	serverSigned, err := serverMulti.Sign(deviceSigned)
	if err != nil {
		t.Fatalf("server sign: %v", err)
	}

	// Step 3: Final event has both signatures.
	extracted, err := signing.ExtractMultiSignature(serverSigned)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if extracted.Count() != 2 {
		t.Fatalf("expected 2 signatures, got %d", extracted.Count())
	}

	// Step 4: Both signatures verify independently.
	if verifyErr := deviceMulti.Verify(serverSigned); verifyErr != nil {
		t.Fatalf("verify device on final: %v", verifyErr)
	}
	if verifyErr := serverMulti.Verify(serverSigned); verifyErr != nil {
		t.Fatalf("verify server on final: %v", verifyErr)
	}

	// Step 5: VerifyAll with a verifier map.
	verifiers := map[signing.Actor]signing.Verifier{
		signing.Actor("device"): deviceVerifier,
		signing.Actor("server"): serverSigner,
	}
	if verifyErr := signing.VerifyAll(serverSigned, verifiers); verifyErr != nil {
		t.Fatalf("verify all: %v", verifyErr)
	}

	// Step 6: Tamper detection.
	tampered, _ := event.NewEvent(
		serverSigned.Type(),
		serverSigned.AggregateID(),
		serverSigned.AggregateType(),
		serverSigned.Version(),
		[]byte(`{"name":"Bob"}`),
		event.WithEventID(serverSigned.ID()),
		event.WithOccurredAt(serverSigned.OccurredAt()),
		event.WithSchemaVersion(serverSigned.SchemaVersion()),
		event.WithMetadata(serverSigned.Metadata()),
	)

	if verifyErr := deviceMulti.Verify(tampered); verifyErr == nil {
		t.Fatal("expected verification to fail for tampered event")
	}
}

func TestSignatureEntry_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	original := signing.SignatureEntry{
		Actor:     "device",
		Algorithm: signing.AlgorithmEd25519,
		Sig:       signing.Signature([]byte("test-sig-bytes")),
		SignedAt:  time.Now().Truncate(time.Millisecond),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded signing.SignatureEntry

	unmarshalErr := json.Unmarshal(data, &decoded)
	if unmarshalErr != nil {
		t.Fatalf("unmarshal: %v", unmarshalErr)
	}

	if decoded.Actor != original.Actor ||
		decoded.Algorithm != original.Algorithm ||
		!decoded.SignedAt.Equal(original.SignedAt) {
		t.Fatalf("JSON roundtrip failed: got %+v, want %+v", decoded, original)
	}

	if !bytes.Equal(decoded.Sig, original.Sig) {
		t.Fatal("signature bytes mismatch after JSON roundtrip")
	}
}

func TestVerifyAll_MissingVerifier(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	evt := makeTestEvent(t)

	clone, _ := deviceMulti.Sign(evt)

	verifiers := map[signing.Actor]signing.Verifier{}
	err := signing.VerifyAll(clone, verifiers)
	if err == nil {
		t.Fatal("expected error for missing verifier")
	}
}

func TestVerifyAll_FailingVerifier(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	evt := makeTestEvent(t)

	clone, _ := deviceMulti.Sign(evt)
	tampered, tamperErr := event.NewEvent(
		clone.Type(),
		clone.AggregateID(),
		clone.AggregateType(),
		clone.Version(),
		[]byte(`{"tampered":true}`),
		event.WithEventID(clone.ID()),
		event.WithOccurredAt(clone.OccurredAt()),
		event.WithSchemaVersion(clone.SchemaVersion()),
		event.WithMetadata(clone.Metadata()),
	)
	if tamperErr != nil {
		t.Fatalf("tamper: %v", tamperErr)
	}

	pubKey, _, _ := ed25519.GenerateKey(nil)
	verifier, _ := signing.NewEd25519Verifier(pubKey)

	verifiers := map[signing.Actor]signing.Verifier{signing.Actor("device"): verifier}
	err := signing.VerifyAll(tampered, verifiers)
	if err == nil {
		t.Fatal("expected error for tampered event with wrong verifier")
	}
}

func TestMultiSigner_VerifyActor(t *testing.T) {
	t.Parallel()

	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	edSigner, _ := signing.NewEd25519(privKey)
	edVerifier, _ := signing.NewEd25519Verifier(pubKey)

	deviceMulti, err := signing.NewMultiSigner(signing.Actor("device"), signing.AlgorithmEd25519, edSigner,
		signing.WithVerifier(edVerifier))
	if err != nil {
		t.Fatalf("create device multi-signer: %v", err)
	}
	serverMulti := newServerMultiSigner(t)
	evt := makeTestEvent(t)

	clone1, _ := deviceMulti.Sign(evt)
	clone2, _ := serverMulti.Sign(clone1)

	if verifyErr := serverMulti.VerifyActor(clone2, signing.Actor("device"), edVerifier); verifyErr != nil {
		t.Fatalf("server verifying device: %v", verifyErr)
	}
}

func TestMultiSigner_WithClock(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	edSigner, _ := signing.NewEd25519(privKey)
	edVerifier, _ := signing.NewEd25519Verifier(pubKey)

	deterministic, err := signing.NewMultiSigner(
		signing.Actor("device"), signing.AlgorithmEd25519,
		edSigner,
		signing.WithVerifier(edVerifier),
		signing.WithClock(func() time.Time { return fixedTime }),
	)
	if err != nil {
		t.Fatalf("create deterministic multi-signer: %v", err)
	}

	evt := makeTestEvent(t)
	clone, _ := deterministic.Sign(evt)

	extracted, _ := signing.ExtractMultiSignature(clone)
	entry := extracted.Get(signing.Actor("device"))
	if entry == nil {
		t.Fatal("expected device entry")
	}

	if !entry.SignedAt.Equal(fixedTime) {
		t.Fatalf("SignedAt: got %v, want %v", entry.SignedAt, fixedTime)
	}
}

func TestMultiVerifyMiddleware_RejectsTampered(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)

	handler := func(_ context.Context, _ event.Event) error {
		return nil
	}

	mw := signing.MultiVerifyMiddleware(deviceMulti)
	wrapped := mw(handler)

	evt := makeTestEvent(t)
	clone, _ := deviceMulti.Sign(evt)

	tampered, tamperErr := event.NewEvent(
		clone.Type(),
		clone.AggregateID(),
		clone.AggregateType(),
		clone.Version(),
		[]byte(`{"tampered":true}`),
		event.WithEventID(clone.ID()),
		event.WithOccurredAt(clone.OccurredAt()),
		event.WithSchemaVersion(clone.SchemaVersion()),
		event.WithMetadata(clone.Metadata()),
	)
	if tamperErr != nil {
		t.Fatalf("tamper: %v", tamperErr)
	}

	err := wrapped(context.Background(), tampered)
	if err == nil {
		t.Fatal("expected error for tampered multi-sig event")
	}
}

func TestExtractMultiSignature_InvalidJSON(t *testing.T) {
	t.Parallel()

	// Create an event with malformed multi-sig JSON in metadata
	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	evt, err := event.NewEvent(
		"test.invalid", aggID, "Test", 1, []byte(`{}`),
		event.WithMetadata(&event.Metadata{
			Custom: map[event.MetadataKey]string{
				signing.MultiSigMetadataKey: `{invalid json`,
			},
		}),
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	_, extractErr := signing.ExtractMultiSignature(evt)
	if extractErr == nil {
		t.Fatal("expected error for invalid JSON in multi-sig metadata")
	}
}

func TestMultiSigner_Verify_NilEvent(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)

	if err := deviceMulti.Verify(nil); err == nil {
		t.Fatal("expected error for nil event")
	}
}

func TestMultiSigner_VerifyActor_NilEvent(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	pubKey, _, _ := ed25519.GenerateKey(nil)
	verifier, _ := signing.NewEd25519Verifier(pubKey)

	if err := deviceMulti.VerifyActor(nil, signing.Actor("device"), verifier); err == nil {
		t.Fatal("expected error for nil event")
	}
}

func TestMultiSigner_VerifyActor_NoSignature(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	serverMulti := newServerMultiSigner(t)

	evt := makeTestEvent(t)
	// Only server signs, then we try to verify device
	clone, _ := serverMulti.Sign(evt)

	pubKey, _, _ := ed25519.GenerateKey(nil)
	verifier, _ := signing.NewEd25519Verifier(pubKey)

	if err := deviceMulti.VerifyActor(clone, signing.Actor("device"), verifier); err == nil {
		t.Fatal("expected error when actor has no signature")
	}
}

func TestMultiSigner_VerifyActor_BadSignature(t *testing.T) {
	t.Parallel()

	deviceMulti, devicePubKey := newDeviceMultiSigner(t)
	evt := makeTestEvent(t)
	clone, _ := deviceMulti.Sign(evt)

	// Tamper with the event after signing
	tampered, _ := event.NewEvent(
		clone.Type(), clone.AggregateID(), clone.AggregateType(), clone.Version(),
		[]byte(`{"tampered":true}`),
		event.WithEventID(clone.ID()),
		event.WithOccurredAt(clone.OccurredAt()),
		event.WithSchemaVersion(clone.SchemaVersion()),
		event.WithMetadata(clone.Metadata()),
	)

	deviceVerifier, _ := signing.NewEd25519Verifier(devicePubKey)

	if err := deviceMulti.VerifyActor(tampered, signing.Actor("device"), deviceVerifier); err == nil {
		t.Fatal("expected error for tampered event")
	}
}

func TestVerifyAll_NilEvent(t *testing.T) {
	t.Parallel()

	verifiers := map[signing.Actor]signing.Verifier{}
	if err := signing.VerifyAll(nil, verifiers); err == nil {
		t.Fatal("expected error for nil event")
	}
}

func TestMultiVerifyMiddleware_NoMultiSig(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)

	called := false
	handler := func(_ context.Context, _ event.Event) error {
		called = true

		return nil
	}

	mw := signing.MultiVerifyMiddleware(deviceMulti)
	wrapped := mw(handler)

	// Unsigned event should pass through
	evt := makeTestEvent(t)
	if err := wrapped(context.Background(), evt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Fatal("handler should have been called for unsigned event")
	}
}

func TestRequireMultiSigMiddleware_NilEvent(t *testing.T) {
	t.Parallel()

	handler := func(_ context.Context, _ event.Event) error {
		return nil
	}

	verifiers := map[signing.Actor]signing.Verifier{}
	mw := signing.RequireMultiSigMiddleware(verifiers)
	wrapped := mw(handler)

	if err := wrapped(context.Background(), nil); err == nil {
		t.Fatal("expected error for nil event")
	}
}

func TestNewMultiSigner_Validation(t *testing.T) {
	t.Parallel()

	key := []byte("server-secret-key-thirty-two-by!")
	signer, _ := signing.NewHMAC(key)

	t.Run("rejects empty actor", func(t *testing.T) {
		t.Parallel()
		_, err := signing.NewMultiSigner("", signing.AlgorithmHMACSHA256, signer)
		if err == nil {
			t.Fatal("expected error for empty actor")
		}
	})

	t.Run("rejects nil signer", func(t *testing.T) {
		t.Parallel()
		_, err := signing.NewMultiSigner(signing.Actor("server"), signing.AlgorithmHMACSHA256, nil)
		if err == nil {
			t.Fatal("expected error for nil signer")
		}
	})

	t.Run("rejects nil clock", func(t *testing.T) {
		t.Parallel()
		_, err := signing.NewMultiSigner(
			signing.Actor("server"), signing.AlgorithmHMACSHA256, signer,
			signing.WithClock(nil),
		)
		if err == nil {
			t.Fatal("expected error for nil clock")
		}
	})
}

func TestMultiSigner_Algorithm(t *testing.T) {
	t.Parallel()

	key := []byte("server-secret-key-thirty-two-by!")
	signer, _ := signing.NewHMAC(key)

	multi, err := signing.NewMultiSigner(signing.Actor("server"), signing.AlgorithmHMACSHA256, signer)
	if err != nil {
		t.Fatalf("create multi-signer: %v", err)
	}

	if multi.Algorithm() != signing.AlgorithmHMACSHA256 {
		t.Fatalf("algorithm mismatch: got %s, want %s", multi.Algorithm(), signing.AlgorithmHMACSHA256)
	}
}

func TestMultiVerifyMiddlewareFor(t *testing.T) {
	t.Parallel()

	deviceMulti, devicePubKey := newDeviceMultiSigner(t)
	serverMulti := newServerMultiSigner(t)

	deviceVerifier, _ := signing.NewEd25519Verifier(devicePubKey)

	t.Run("allows valid signature", func(t *testing.T) {
		t.Parallel()

		called := false
		handler := func(_ context.Context, _ event.Event) error {
			called = true

			return nil
		}

		mw := signing.MultiVerifyMiddlewareFor(signing.Actor("device"), deviceVerifier)
		wrapped := mw(handler)

		evt := makeTestEvent(t)
		clone, _ := deviceMulti.Sign(evt)
		clone2, _ := serverMulti.Sign(clone)

		if err := wrapped(context.Background(), clone2); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !called {
			t.Fatal("handler should have been called")
		}
	})

	t.Run("rejects tampered event", func(t *testing.T) {
		t.Parallel()

		handler := func(_ context.Context, _ event.Event) error {
			return nil
		}

		mw := signing.MultiVerifyMiddlewareFor(signing.Actor("device"), deviceVerifier)
		wrapped := mw(handler)

		evt := makeTestEvent(t)
		clone, _ := deviceMulti.Sign(evt)

		tampered, _ := event.NewEvent(
			clone.Type(), clone.AggregateID(), clone.AggregateType(), clone.Version(),
			[]byte(`{"tampered":true}`),
			event.WithEventID(clone.ID()),
			event.WithOccurredAt(clone.OccurredAt()),
			event.WithSchemaVersion(clone.SchemaVersion()),
			event.WithMetadata(clone.Metadata()),
		)

		if err := wrapped(context.Background(), tampered); err == nil {
			t.Fatal("expected error for tampered event")
		}
	})

	t.Run("passes through unsigned event", func(t *testing.T) {
		t.Parallel()

		called := false
		handler := func(_ context.Context, _ event.Event) error {
			called = true

			return nil
		}

		mw := signing.MultiVerifyMiddlewareFor(signing.Actor("device"), deviceVerifier)
		wrapped := mw(handler)

		evt := makeTestEvent(t)
		if err := wrapped(context.Background(), evt); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !called {
			t.Fatal("handler should have been called for unsigned event")
		}
	})

	t.Run("rejects missing actor signature", func(t *testing.T) {
		t.Parallel()

		handler := func(_ context.Context, _ event.Event) error {
			return nil
		}

		mw := signing.MultiVerifyMiddlewareFor(signing.Actor("device"), deviceVerifier)
		wrapped := mw(handler)

		evt := makeTestEvent(t)
		clone, _ := serverMulti.Sign(evt)

		if err := wrapped(context.Background(), clone); err == nil {
			t.Fatal("expected error when actor signature is missing")
		}
	})
}

func TestSignatureEntry_Validate(t *testing.T) {
	t.Parallel()

	valid := signing.SignatureEntry{
		Actor:     signing.Actor("device"),
		Algorithm: signing.AlgorithmEd25519,
		Sig:       signing.Signature([]byte("sig")),
		SignedAt:  time.Now(),
	}

	if err := valid.Validate(); err != nil {
		t.Fatalf("valid entry should pass: %v", err)
	}

	t.Run("rejects empty actor", func(t *testing.T) {
		t.Parallel()
		entry := valid
		entry.Actor = ""
		if err := entry.Validate(); err == nil {
			t.Fatal("expected error for empty actor")
		}
	})

	t.Run("rejects empty algorithm", func(t *testing.T) {
		t.Parallel()
		entry := valid
		entry.Algorithm = ""
		if err := entry.Validate(); err == nil {
			t.Fatal("expected error for empty algorithm")
		}
	})

	t.Run("rejects empty sig", func(t *testing.T) {
		t.Parallel()
		entry := valid
		entry.Sig = nil
		if err := entry.Validate(); err == nil {
			t.Fatal("expected error for empty sig")
		}
	})

	t.Run("rejects zero signedAt", func(t *testing.T) {
		t.Parallel()
		entry := valid
		entry.SignedAt = time.Time{}
		if err := entry.Validate(); err == nil {
			t.Fatal("expected error for zero signedAt")
		}
	})
}
