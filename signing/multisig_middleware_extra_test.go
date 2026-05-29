package signing_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/signing"
)

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

		tampered := tamperEvent(t, clone)

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

func TestMultiSigMiddlewareNilGuards(t *testing.T) {
	t.Parallel()

	t.Run("MultiSignMiddleware panics on nil signer", func(t *testing.T) {
		t.Parallel()
		assertPanicMessage(t, func() { signing.MultiSignMiddleware(nil) },
			"signing: MultiSignMiddleware called with nil signer")
	})

	t.Run("MultiVerifyMiddleware panics on nil signer", func(t *testing.T) {
		t.Parallel()
		assertPanicMessage(t, func() { signing.MultiVerifyMiddleware(nil) },
			"signing: MultiVerifyMiddleware called with nil signer")
	})

	t.Run("MultiVerifyMiddlewareFor panics on nil verifier", func(t *testing.T) {
		t.Parallel()
		assertPanicMessage(t, func() { signing.MultiVerifyMiddlewareFor(signing.Actor("device"), nil) },
			"signing: MultiVerifyMiddlewareFor called with nil verifier")
	})

	t.Run("RequireMultiSigMiddleware panics on empty map", func(t *testing.T) {
		t.Parallel()
		assertPanicMessage(t, func() { signing.RequireMultiSigMiddleware(map[signing.Actor]signing.Verifier{}) },
			"signing: RequireMultiSigMiddleware called with empty verifiers map")
	})
}

func TestCorruptedMultiSigMiddleware(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)

	makeCorruptMultiSigEvent := func(t *testing.T) event.Event {
		t.Helper()
		evt := makeTestEvent(t)
		corrupt, err := event.NewEvent(
			evt.Type(), evt.AggregateID(), evt.AggregateType(), evt.Version(),
			evt.Payload(),
			event.WithEventID(evt.ID()),
			event.WithOccurredAt(evt.OccurredAt()),
			event.WithCustom(signing.MultiSigMetadataKey, "{invalid json"),
		)
		if err != nil {
			t.Fatalf("create corrupt event: %v", err)
		}

		return corrupt
	}

	t.Run("MultiVerifyMiddleware rejects corrupt multi-sig", func(t *testing.T) {
		t.Parallel()

		handler := func(_ context.Context, _ event.Event) error { return nil }
		mw := signing.MultiVerifyMiddleware(deviceMulti)
		wrapped := mw(handler)

		evt := makeCorruptMultiSigEvent(t)
		err := wrapped(context.Background(), evt)
		if err == nil {
			t.Fatal("expected error for corrupt multi-sig metadata")
		}
	})

	t.Run("MultiVerifyMiddlewareFor rejects corrupt multi-sig", func(t *testing.T) {
		t.Parallel()

		key := []byte("corrupt-test-key-thirty-two-bytes!")
		verifier, _ := signing.NewHMAC(key)
		handler := func(_ context.Context, _ event.Event) error { return nil }
		mw := signing.MultiVerifyMiddlewareFor(signing.Actor("device"), verifier)
		wrapped := mw(handler)

		evt := makeCorruptMultiSigEvent(t)
		err := wrapped(context.Background(), evt)
		if err == nil {
			t.Fatal("expected error for corrupt multi-sig metadata")
		}
	})
}
