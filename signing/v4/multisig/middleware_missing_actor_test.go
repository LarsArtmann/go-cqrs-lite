package multisig_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/signing/v4"
	"github.com/larsartmann/go-cqrs-lite/signing/v4/internal/testutil"
	"github.com/larsartmann/go-cqrs-lite/signing/v4/multisig"
)

// A collection that lacks this actor's entry is a rejection, not a pass-through:
// a stripped entry must not read like an unsigned event.
func TestMultiVerifyMiddleware_RejectsMissingActor(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	serverMulti := newServerMultiSigner(t)

	called := false
	wrapped := multisig.MultiVerifyMiddleware(deviceMulti)(func(context.Context, event.Event) error {
		called = true
		return nil
	})

	signed, err := serverMulti.Sign(testutil.MakeTestEvent(t))
	if err != nil {
		t.Fatal(err)
	}

	err = wrapped(context.Background(), signed)
	if err == nil {
		t.Fatal("expected error for a multi-sig without this actor")
	}

	if !errors.Is(err, signing.ErrNilSignature) || called {
		t.Fatalf("expected an ErrNilSignature rejection and no handler call, got %v (called=%v)", err, called)
	}
}
