package signing_test

import (
	"context"
	"crypto/ed25519"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/signing"
)

func TestMultiSignature(t *testing.T) {
	t.Parallel()

	multiSig := signing.MultiSignature{
		Entries: []signing.SignatureEntry{
			{Actor: "device", Algorithm: signing.AlgorithmEd25519, Sig: []byte("sig1")},
			{Actor: "server", Algorithm: signing.AlgorithmHMACSHA256, Sig: []byte("sig2")},
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
		if !multiSig.HasActor("device") {
			t.Fatal("expected HasActor(device) = true")
		}
		if multiSig.HasActor("gateway") {
			t.Fatal("expected HasActor(gateway) = false")
		}
	})

	t.Run("get", func(t *testing.T) {
		t.Parallel()
		entry := multiSig.Get("server")
		if entry == nil {
			t.Fatal("expected entry for server")
		}
		if entry.Algorithm != signing.AlgorithmHMACSHA256 {
			t.Fatalf("algorithm: got %s, want HMAC-SHA256", entry.Algorithm)
		}
		if multiSig.Get("gateway") != nil {
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
			{Actor: "device", Algorithm: signing.AlgorithmEd25519, Sig: []byte("a")},
			{Actor: "device", Algorithm: signing.AlgorithmEd25519, Sig: []byte("b")},
			{Actor: "server", Algorithm: signing.AlgorithmHMACSHA256, Sig: []byte("c")},
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

	return signing.NewMultiSigner("device", signing.AlgorithmEd25519, signer,
		signing.WithVerifier(verifier)), pubKey
}

// newServerMultiSigner creates a test MultiSigner for the "server" actor using HMAC.
func newServerMultiSigner(t *testing.T) *signing.MultiSigner {
	t.Helper()

	key := []byte("server-secret-key-thirty-two-by!")
	signer, err := signing.NewHMAC(key)
	if err != nil {
		t.Fatalf("create HMAC signer: %v", err)
	}

	return signing.NewMultiSigner("server", signing.AlgorithmHMACSHA256, signer)
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
	if !extracted.HasActor("device") {
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
	if !extracted.HasActor("device") || !extracted.HasActor("server") {
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
	entry1 := extracted1.Get("device")
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

	entry2 := extracted2.Get("device")
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
		if !extracted.HasActor("device") {
			t.Fatal("expected device signature")
		}
	})
}

func TestRequireMultiSigMiddleware(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	serverMulti := newServerMultiSigner(t)

	t.Run("rejects unsigned events", func(t *testing.T) {
		t.Parallel()

		called := false
		handler := func(_ context.Context, _ event.Event) error {
			called = true

			return nil
		}

		mw := signing.RequireMultiSigMiddleware("device", "server")
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

		mw := signing.RequireMultiSigMiddleware("device", "server")
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

		mw := signing.RequireMultiSigMiddleware("device", "server")
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

	deviceMulti := signing.NewMultiSigner("device", signing.AlgorithmEd25519, deviceSigner,
		signing.WithVerifier(deviceVerifier))
	serverMulti := signing.NewMultiSigner("server", signing.AlgorithmHMACSHA256, serverSigner)

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
	verifiers := map[string]signing.Verifier{
		"device": deviceVerifier,
		"server": serverSigner,
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
