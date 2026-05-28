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

	ms := signing.MultiSignature{
		Entries: []signing.SignatureEntry{
			{Actor: "device", Algorithm: signing.AlgorithmEd25519, Sig: []byte("sig1")},
			{Actor: "server", Algorithm: signing.AlgorithmHMACSHA256, Sig: []byte("sig2")},
		},
	}

	t.Run("count", func(t *testing.T) {
		t.Parallel()
		if got, want := ms.Count(), 2; got != want {
			t.Fatalf("Count: got %d, want %d", got, want)
		}
	})

	t.Run("has actor", func(t *testing.T) {
		t.Parallel()
		if !ms.HasActor("device") {
			t.Fatal("expected HasActor(device) = true")
		}
		if ms.HasActor("gateway") {
			t.Fatal("expected HasActor(gateway) = false")
		}
	})

	t.Run("get", func(t *testing.T) {
		t.Parallel()
		entry := ms.Get("server")
		if entry == nil {
			t.Fatal("expected entry for server")
		}
		if entry.Algorithm != signing.AlgorithmHMACSHA256 {
			t.Fatalf("algorithm: got %s, want HMAC-SHA256", entry.Algorithm)
		}
		if ms.Get("gateway") != nil {
			t.Fatal("expected nil for unknown actor")
		}
	})

	t.Run("actors", func(t *testing.T) {
		t.Parallel()
		actors := ms.Actors()
		if len(actors) != 2 {
			t.Fatalf("expected 2 actors, got %d", len(actors))
		}
	})
}

func TestMultiSigner(t *testing.T) {
	t.Parallel()

	pubKey, devicePrivKey, _ := ed25519.GenerateKey(nil)
	serverKey := []byte("server-key-thirty-two-bytes!!")

	deviceSigner, _ := signing.NewEd25519(devicePrivKey)
	deviceVerifier, _ := signing.NewEd25519Verifier(pubKey)
	serverSigner, _ := signing.NewHMAC(serverKey)

	deviceMulti := signing.NewMultiSigner("device", signing.AlgorithmEd25519, deviceSigner,
		signing.WithVerifier(deviceVerifier))
	serverMulti := signing.NewMultiSigner("server", signing.AlgorithmHMACSHA256, serverSigner)

	evt := makeTestEvent(t)

	t.Run("sign adds actor signature", func(t *testing.T) {
		t.Parallel()

		clone, err := deviceMulti.Sign(evt)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}

		ms, err := signing.ExtractMultiSignature(clone)
		if err != nil {
			t.Fatalf("extract: %v", err)
		}
		if ms.Count() != 1 {
			t.Fatalf("expected 1 entry, got %d", ms.Count())
		}
		if !ms.HasActor("device") {
			t.Fatal("expected device actor")
		}
	})

	t.Run("multiple actors sign same event", func(t *testing.T) {
		t.Parallel()

		clone1, err := deviceMulti.Sign(evt)
		if err != nil {
			t.Fatalf("device sign: %v", err)
		}

		clone2, err := serverMulti.Sign(clone1)
		if err != nil {
			t.Fatalf("server sign: %v", err)
		}

		ms, err := signing.ExtractMultiSignature(clone2)
		if err != nil {
			t.Fatalf("extract: %v", err)
		}
		if ms.Count() != 2 {
			t.Fatalf("expected 2 entries, got %d", ms.Count())
		}
		if !ms.HasActor("device") || !ms.HasActor("server") {
			t.Fatal("expected both device and server actors")
		}

		actors := ms.Actors()
		if len(actors) != 2 {
			t.Fatalf("expected 2 actors, got %d", len(actors))
		}
	})

	t.Run("same actor re-signing replaces old signature", func(t *testing.T) {
		t.Parallel()

		clone1, err := deviceMulti.Sign(evt)
		if err != nil {
			t.Fatalf("first sign: %v", err)
		}

		ms1, _ := signing.ExtractMultiSignature(clone1)
		entry1 := ms1.Get("device")
		if entry1 == nil {
			t.Fatal("expected device entry after first sign")
		}

		clone2, err := deviceMulti.Sign(clone1)
		if err != nil {
			t.Fatalf("second sign: %v", err)
		}

		ms2, _ := signing.ExtractMultiSignature(clone2)
		if ms2.Count() != 1 {
			t.Fatalf("expected 1 entry after re-sign, got %d", ms2.Count())
		}

		entry2 := ms2.Get("device")
		if entry2.SignedAt.Equal(entry1.SignedAt) {
			t.Fatal("re-signed entry should have different timestamp")
		}
	})

	t.Run("verify own signature", func(t *testing.T) {
		t.Parallel()

		clone, _ := deviceMulti.Sign(evt)
		err := deviceMulti.Verify(clone)
		if err != nil {
			t.Fatalf("verify device: %v", err)
		}
	})

	t.Run("verify server signature on dual-signed event", func(t *testing.T) {
		t.Parallel()

		clone1, _ := deviceMulti.Sign(evt)
		clone2, _ := serverMulti.Sign(clone1)

		err := serverMulti.Verify(clone2)
		if err != nil {
			t.Fatalf("verify server: %v", err)
		}

		err = deviceMulti.Verify(clone2)
		if err != nil {
			t.Fatalf("verify device on dual-signed: %v", err)
		}
	})

	t.Run("verify missing actor", func(t *testing.T) {
		t.Parallel()

		clone, _ := deviceMulti.Sign(evt)
		err := serverMulti.Verify(clone)
		if err == nil {
			t.Fatal("expected error verifying missing actor")
		}
	})

	t.Run("verify tampered event", func(t *testing.T) {
		t.Parallel()

		clone, _ := deviceMulti.Sign(evt)

		// Reconstruct with tampered payload but same metadata (including multi-sig)
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

		err := deviceMulti.Verify(tampered)
		if err == nil {
			t.Fatal("expected verification to fail for tampered event")
		}
	})

	t.Run("sign nil event", func(t *testing.T) {
		t.Parallel()
		_, err := deviceMulti.Sign(nil)
		if err == nil {
			t.Fatal("expected error for nil event")
		}
	})

	t.Run("verify nil event", func(t *testing.T) {
		t.Parallel()
		err := deviceMulti.Verify(nil)
		if err == nil {
			t.Fatal("expected error for nil event")
		}
	})
}

func TestExtractMultiSignature(t *testing.T) {
	t.Parallel()

	_, privKey, _ := ed25519.GenerateKey(nil)
	signer, _ := signing.NewEd25519(privKey)
	verifier, _ := signing.NewEd25519Verifier(privKey.Public().(ed25519.PublicKey))
	multi := signing.NewMultiSigner("device", signing.AlgorithmEd25519, signer,
		signing.WithVerifier(verifier))

	evt := makeTestEvent(t)

	t.Run("extract from unsigned event", func(t *testing.T) {
		t.Parallel()
		_, err := signing.ExtractMultiSignature(evt)
		if err == nil {
			t.Fatal("expected error for unsigned event")
		}
	})

	t.Run("extract from nil event", func(t *testing.T) {
		t.Parallel()
		_, err := signing.ExtractMultiSignature(nil)
		if err == nil {
			t.Fatal("expected error for nil event")
		}
	})

	t.Run("has multi-sig", func(t *testing.T) {
		t.Parallel()
		if signing.HasMultiSignature(evt) {
			t.Fatal("original event should not have multi-sig")
		}

		clone, _ := multi.Sign(evt)
		if !signing.HasMultiSignature(clone) {
			t.Fatal("signed event should have multi-sig")
		}
	})
}

func TestMultiSignatureActors(t *testing.T) {
	t.Parallel()

	ms := signing.MultiSignature{
		Entries: []signing.SignatureEntry{
			{Actor: "device", Algorithm: signing.AlgorithmEd25519, Sig: []byte("a")},
			{Actor: "device", Algorithm: signing.AlgorithmEd25519, Sig: []byte("b")},
			{Actor: "server", Algorithm: signing.AlgorithmHMACSHA256, Sig: []byte("c")},
		},
	}

	actors := ms.Actors()
	if len(actors) != 2 {
		t.Fatalf("expected 2 unique actors, got %d: %v", len(actors), actors)
	}
}

func TestMultiSignMiddleware(t *testing.T) {
	t.Parallel()

	_, privKey, _ := ed25519.GenerateKey(nil)
	signer, _ := signing.NewEd25519(privKey)
	verifier, _ := signing.NewEd25519Verifier(privKey.Public().(ed25519.PublicKey))
	multi := signing.NewMultiSigner("device", signing.AlgorithmEd25519, signer,
		signing.WithVerifier(verifier))

	t.Run("signs events before publishing", func(t *testing.T) {
		t.Parallel()

		var published []event.Event
		pub := event.PublisherFunc(func(_ context.Context, events ...event.Event) error {
			published = append(published, events...)
			return nil
		})

		mw := signing.MultiSignMiddleware(multi)
		signedPub := mw(pub)

		evt := makeTestEvent(t)
		err := signedPub.Publish(context.Background(), evt)
		if err != nil {
			t.Fatalf("publish: %v", err)
		}

		if len(published) != 1 {
			t.Fatalf("expected 1 event, got %d", len(published))
		}

		if !signing.HasMultiSignature(published[0]) {
			t.Fatal("event should have multi-sig after middleware")
		}

		ms, _ := signing.ExtractMultiSignature(published[0])
		if !ms.HasActor("device") {
			t.Fatal("expected device signature")
		}
	})
}

func TestRequireMultiSigMiddleware(t *testing.T) {
	t.Parallel()

	pubKey, devicePrivKey, _ := ed25519.GenerateKey(nil)
	serverKey := []byte("server-key-thirty-two-bytes!!")

	deviceSigner, _ := signing.NewEd25519(devicePrivKey)
	deviceVerifier, _ := signing.NewEd25519Verifier(pubKey)
	serverSigner, _ := signing.NewHMAC(serverKey)

	deviceMulti := signing.NewMultiSigner("device", signing.AlgorithmEd25519, deviceSigner,
		signing.WithVerifier(deviceVerifier))
	serverMulti := signing.NewMultiSigner("server", signing.AlgorithmHMACSHA256, serverSigner)

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
		err := wrapped(context.Background(), evt)
		if err == nil {
			t.Fatal("expected error for unsigned event")
		}
		if called {
			t.Fatal("handler should not have been called")
		}
	})

	t.Run("rejects partially signed events", func(t *testing.T) {
		t.Parallel()

		handler := func(_ context.Context, _ event.Event) error { return nil }
		mw := signing.RequireMultiSigMiddleware("device", "server")
		wrapped := mw(handler)

		evt := makeTestEvent(t)
		clone, _ := deviceMulti.Sign(evt)

		err := wrapped(context.Background(), clone)
		if err == nil {
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

		err := wrapped(context.Background(), clone2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Fatal("handler should have been called")
		}
	})
}

func TestMultiSignerEndToEnd(t *testing.T) {
	t.Parallel()

	pubKey, devicePrivKey, _ := ed25519.GenerateKey(nil)
	serverKey := []byte("server-key-thirty-two-bytes!!")

	deviceSigner, _ := signing.NewEd25519(devicePrivKey)
	deviceVerifier, _ := signing.NewEd25519Verifier(pubKey)
	serverSigner, _ := signing.NewHMAC(serverKey)

	deviceMulti := signing.NewMultiSigner("device", signing.AlgorithmEd25519, deviceSigner,
		signing.WithVerifier(deviceVerifier))
	serverMulti := signing.NewMultiSigner("server", signing.AlgorithmHMACSHA256, serverSigner)

	// Step 1: Device creates and signs the event
	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	deviceEvent, _ := event.NewEvent("user.created", aggID, "User", 1, []byte(`{"name":"Alice"}`))

	deviceSigned, err := deviceMulti.Sign(deviceEvent)
	if err != nil {
		t.Fatalf("device sign: %v", err)
	}

	// Step 2: Server verifies device's signature
	if err := deviceMulti.Verify(deviceSigned); err != nil {
		t.Fatalf("server verifies device: %v", err)
	}

	// Step 3: Server adds its own signature
	serverSigned, err := serverMulti.Sign(deviceSigned)
	if err != nil {
		t.Fatalf("server sign: %v", err)
	}

	// Step 4: Final event has both signatures
	ms, err := signing.ExtractMultiSignature(serverSigned)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if ms.Count() != 2 {
		t.Fatalf("expected 2 signatures, got %d", ms.Count())
	}

	// Step 5: Both signatures verify independently
	if err := deviceMulti.Verify(serverSigned); err != nil {
		t.Fatalf("verify device on final: %v", err)
	}
	if err := serverMulti.Verify(serverSigned); err != nil {
		t.Fatalf("verify server on final: %v", err)
	}

	// Step 6: VerifyAll with a verifier map
	verifiers := map[string]signing.Signer{
		"device": deviceVerifier,
		"server": serverSigner,
	}
	if err := deviceMulti.VerifyAll(serverSigned, verifiers); err != nil {
		t.Fatalf("verify all: %v", err)
	}

	// Step 7: Tamper detection
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

	if err := deviceMulti.Verify(tampered); err == nil {
		t.Fatal("expected verification to fail for tampered event")
	}
}
