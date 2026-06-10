package multisig_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/signing/v2"
	"github.com/larsartmann/go-cqrs-lite/signing/v2/internal/testutil"
	"github.com/larsartmann/go-cqrs-lite/signing/v2/multisig"
)

func TestMultiSignMiddleware(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)

	t.Run("signs events before publishing", func(t *testing.T) {
		t.Parallel()

		pub, publishedPtr := testutil.CollectingPublisher()

		mw := multisig.MultiSignMiddleware(deviceMulti)
		signedPub := mw(pub)

		evt := testutil.MakeTestEvent(t)
		if err := signedPub.Publish(context.Background(), evt); err != nil {
			t.Fatalf("publish: %v", err)
		}

		if len(*publishedPtr) != 1 {
			t.Fatalf("expected 1 event, got %d", len(*publishedPtr))
		}

		if !multisig.HasMultiSignature((*publishedPtr)[0]) {
			t.Fatal("event should have multi-sig after middleware")
		}

		extracted, _ := multisig.ExtractMultiSignature((*publishedPtr)[0])
		if !extracted.HasActor(multisig.Actor("device")) {
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

	verifiers := map[multisig.Actor]signing.Verifier{
		multisig.Actor("device"): deviceVerifier,
		multisig.Actor("server"): serverHMAC,
	}

	t.Run("rejects unsigned events", func(t *testing.T) {
		t.Parallel()

		handler, wasCalled := testutil.TrackingHandler()

		mw := multisig.RequireMultiSigMiddleware(verifiers)
		wrapped := mw(handler)

		evt := testutil.MakeTestEvent(t)
		if err := wrapped(context.Background(), evt); err == nil {
			t.Fatal("expected error for unsigned event")
		}
		if wasCalled() {
			t.Fatal("handler should not have been called")
		}
	})

	t.Run("rejects partially signed events", func(t *testing.T) {
		t.Parallel()

		handler := testutil.NoopHandler

		mw := multisig.RequireMultiSigMiddleware(verifiers)
		wrapped := mw(handler)

		evt := testutil.MakeTestEvent(t)
		clone, _ := deviceMulti.Sign(evt)

		if err := wrapped(context.Background(), clone); err == nil {
			t.Fatal("expected error for partially signed event")
		}
	})

	t.Run("allows fully signed events", func(t *testing.T) {
		t.Parallel()

		handler, wasCalled := testutil.TrackingHandler()

		mw := multisig.RequireMultiSigMiddleware(verifiers)
		wrapped := mw(handler)

		evt := testutil.MakeTestEvent(t)
		clone1, _ := deviceMulti.Sign(evt)
		clone2, _ := serverMulti.Sign(clone1)

		if err := wrapped(context.Background(), clone2); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !wasCalled() {
			t.Fatal("handler should have been called")
		}
	})
}

func TestMultiVerifyMiddleware_RejectsTampered(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)

	handler := testutil.NoopHandler

	mw := multisig.MultiVerifyMiddleware(deviceMulti)
	wrapped := mw(handler)

	evt := testutil.MakeTestEvent(t)
	clone, _ := deviceMulti.Sign(evt)

	tampered := testutil.TamperEvent(t, clone)

	err := wrapped(context.Background(), tampered)
	if err == nil {
		t.Fatal("expected error for tampered multi-sig event")
	}
}

func TestMultiVerifyMiddleware_NoMultiSig(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)

	handler, wasCalled := testutil.TrackingHandler()

	mw := multisig.MultiVerifyMiddleware(deviceMulti)
	wrapped := mw(handler)

	evt := testutil.MakeTestEvent(t)
	if err := wrapped(context.Background(), evt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !wasCalled() {
		t.Fatal("handler should have been called for unsigned event")
	}
}

func TestRequireMultiSigMiddleware_NilEvent(t *testing.T) {
	t.Parallel()

	handler := testutil.NoopHandler

	key := []byte("nil-event-test-key-thirty-two-by!")
	verifier, _ := signing.NewHMAC(key)
	verifiers := map[multisig.Actor]signing.Verifier{
		multisig.Actor("server"): verifier,
	}
	mw := multisig.RequireMultiSigMiddleware(verifiers)
	wrapped := mw(handler)

	if err := wrapped(context.Background(), nil); err == nil {
		t.Fatal("expected error for nil event")
	}
}

func TestMultiVerifyMiddlewareFor(t *testing.T) {
	t.Parallel()

	deviceMulti, devicePubKey := newDeviceMultiSigner(t)
	serverMulti := newServerMultiSigner(t)

	deviceVerifier, _ := signing.NewEd25519Verifier(devicePubKey)

	t.Run("allows valid signature", func(t *testing.T) {
		t.Parallel()

		handler, wasCalled := testutil.TrackingHandler()

		mw := multisig.MultiVerifyMiddlewareFor(multisig.Actor("device"), deviceVerifier)
		wrapped := mw(handler)

		evt := testutil.MakeTestEvent(t)
		clone, _ := deviceMulti.Sign(evt)
		clone2, _ := serverMulti.Sign(clone)

		if err := wrapped(context.Background(), clone2); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !wasCalled() {
			t.Fatal("handler should have been called")
		}
	})

	t.Run("rejects tampered event", func(t *testing.T) {
		t.Parallel()

		handler := testutil.NoopHandler

		mw := multisig.MultiVerifyMiddlewareFor(multisig.Actor("device"), deviceVerifier)
		wrapped := mw(handler)

		evt := testutil.MakeTestEvent(t)
		clone, _ := deviceMulti.Sign(evt)

		tampered := testutil.TamperEvent(t, clone)

		if err := wrapped(context.Background(), tampered); err == nil {
			t.Fatal("expected error for tampered event")
		}
	})

	t.Run("passes through unsigned event", func(t *testing.T) {
		t.Parallel()

		handler, wasCalled := testutil.TrackingHandler()

		mw := multisig.MultiVerifyMiddlewareFor(multisig.Actor("device"), deviceVerifier)
		wrapped := mw(handler)

		evt := testutil.MakeTestEvent(t)
		if err := wrapped(context.Background(), evt); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !wasCalled() {
			t.Fatal("handler should have been called for unsigned event")
		}
	})

	t.Run("rejects missing actor signature", func(t *testing.T) {
		t.Parallel()

		handler := testutil.NoopHandler

		mw := multisig.MultiVerifyMiddlewareFor(multisig.Actor("device"), deviceVerifier)
		wrapped := mw(handler)

		evt := testutil.MakeTestEvent(t)
		clone, _ := serverMulti.Sign(evt)

		if err := wrapped(context.Background(), clone); err == nil {
			t.Fatal("expected error when actor signature is missing")
		}
	})
}

func TestMultiSigMiddlewareNilGuards(t *testing.T) {
	t.Parallel()

	t.Run("MultiSignMiddleware returns error on nil signer", func(t *testing.T) {
		t.Parallel()

		mw := multisig.MultiSignMiddleware(nil)
		pub := mw(
			event.PublisherFunc(func(_ context.Context, _ ...event.Event) error { return nil }),
		)

		err := pub.Publish(context.Background())
		if err == nil {
			t.Fatal("expected rejection error for nil signer")
		}
	})

	t.Run("MultiVerifyMiddleware returns error on nil signer", func(t *testing.T) {
		t.Parallel()

		mw := multisig.MultiVerifyMiddleware(nil)
		handler := mw(testutil.NoopHandler)

		err := handler(context.Background(), testutil.MakeTestEvent(t))
		if err == nil {
			t.Fatal("expected rejection error for nil signer")
		}
	})

	t.Run("MultiVerifyMiddlewareFor returns error on nil verifier", func(t *testing.T) {
		t.Parallel()

		mw := multisig.MultiVerifyMiddlewareFor(multisig.Actor("device"), nil)
		handler := mw(testutil.NoopHandler)

		err := handler(context.Background(), testutil.MakeTestEvent(t))
		if err == nil {
			t.Fatal("expected rejection error for nil verifier")
		}
	})

	t.Run("RequireMultiSigMiddleware returns error on empty map", func(t *testing.T) {
		t.Parallel()

		mw := multisig.RequireMultiSigMiddleware(map[multisig.Actor]signing.Verifier{})
		handler := mw(testutil.NoopHandler)

		err := handler(context.Background(), testutil.MakeTestEvent(t))
		if err == nil {
			t.Fatal("expected rejection error for empty verifiers map")
		}
	})
}

func TestCorruptedMultiSigMiddleware(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)

	makeCorruptMultiSigEvent := func(t *testing.T) event.Event {
		t.Helper()
		evt := testutil.MakeTestEvent(t)
		corrupt, err := event.NewEvent(
			evt.Type(), evt.AggregateID(), evt.AggregateType(), evt.Version(),
			evt.Payload(),
			event.WithEventID(evt.ID()),
			event.WithOccurredAt(evt.OccurredAt()),
			event.WithCustom(multisig.MultiSigMetadataKey, "{invalid json"),
		)
		if err != nil {
			t.Fatalf("create corrupt event: %v", err)
		}

		return corrupt
	}

	t.Run("MultiVerifyMiddleware rejects corrupt multi-sig", func(t *testing.T) {
		t.Parallel()

		handler := testutil.NoopHandler
		mw := multisig.MultiVerifyMiddleware(deviceMulti)
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
		handler := testutil.NoopHandler
		mw := multisig.MultiVerifyMiddlewareFor(multisig.Actor("device"), verifier)
		wrapped := mw(handler)

		evt := makeCorruptMultiSigEvent(t)
		err := wrapped(context.Background(), evt)
		if err == nil {
			t.Fatal("expected error for corrupt multi-sig metadata")
		}
	})
}
