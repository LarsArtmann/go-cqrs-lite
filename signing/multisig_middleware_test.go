package signing_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/signing"
)

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

	tampered := tamperEvent(t, clone)

	err := wrapped(context.Background(), tampered)
	if err == nil {
		t.Fatal("expected error for tampered multi-sig event")
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

	key := []byte("nil-event-test-key-thirty-two-by!")
	verifier, _ := signing.NewHMAC(key)
	verifiers := map[signing.Actor]signing.Verifier{
		signing.Actor("server"): verifier,
	}
	mw := signing.RequireMultiSigMiddleware(verifiers)
	wrapped := mw(handler)

	if err := wrapped(context.Background(), nil); err == nil {
		t.Fatal("expected error for nil event")
	}
}
