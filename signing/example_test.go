package signing_test

import (
	"context"
	"crypto/ed25519"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/signing"
)

func ExampleNewMultiSigner() {
	_, privKey, _ := ed25519.GenerateKey(nil)
	edSigner, _ := signing.NewEd25519(privKey)

	deviceMulti, err := signing.NewMultiSigner(
		signing.Actor("device"),
		signing.AlgorithmEd25519,
		edSigner,
	)
	if err != nil {
		panic(err)
	}

	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent("order.placed", aggID, "Order", 1, []byte(`{"total":99}`))

	signed, err := deviceMulti.Sign(evt)
	if err != nil {
		panic(err)
	}

	fmt.Println("actor:", deviceMulti.Actor())
	fmt.Println("has multi-sig:", signing.HasMultiSignature(signed))

	// Output:
	// actor: device
	// has multi-sig: true
}

func ExampleVerifyAll() {
	_, devicePriv, _ := ed25519.GenerateKey(nil)
	devicePub := devicePriv.Public().(ed25519.PublicKey) //nolint:forcetypeassert // ed25519.GenerateKey always returns ed25519.PublicKey

	_, serverPriv, _ := ed25519.GenerateKey(nil)
	serverPub := serverPriv.Public().(ed25519.PublicKey) //nolint:forcetypeassert // ed25519.GenerateKey always returns ed25519.PublicKey

	deviceSigner, _ := signing.NewEd25519(devicePriv)
	serverSigner, _ := signing.NewEd25519(serverPriv)

	deviceMulti, _ := signing.NewMultiSigner(signing.Actor("device"), signing.AlgorithmEd25519, deviceSigner)
	serverMulti, _ := signing.NewMultiSigner(signing.Actor("server"), signing.AlgorithmEd25519, serverSigner)

	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent("order.shipped", aggID, "Order", 1, []byte(`{}`))

	// Device signs first, then server signs the already-signed event
	step1, _ := deviceMulti.Sign(evt)
	step2, _ := serverMulti.Sign(step1)

	// Verify all signatures using their respective public key verifiers
	deviceVerifier, _ := signing.NewEd25519Verifier(devicePub)
	serverVerifier, _ := signing.NewEd25519Verifier(serverPub)

	verifiers := map[signing.Actor]signing.Verifier{
		signing.Actor("device"): deviceVerifier,
		signing.Actor("server"): serverVerifier,
	}

	if err := signing.VerifyAll(step2, verifiers); err != nil {
		fmt.Println("verification failed:", err)
	} else {
		fmt.Println("all signatures valid")
	}

	// Output:
	// all signatures valid
}

func ExampleMultiVerifyMiddlewareFor() {
	_, devicePriv, _ := ed25519.GenerateKey(nil)
	devicePub := devicePriv.Public().(ed25519.PublicKey) //nolint:forcetypeassert // ed25519.GenerateKey always returns ed25519.PublicKey

	deviceSigner, _ := signing.NewEd25519(devicePriv)
	deviceMulti, _ := signing.NewMultiSigner(signing.Actor("device"), signing.AlgorithmEd25519, deviceSigner)
	deviceVerifier, _ := signing.NewEd25519Verifier(devicePub)

	// Create middleware that verifies the "device" actor's signature
	verifyMiddleware := signing.MultiVerifyMiddlewareFor(signing.Actor("device"), deviceVerifier)

	// The middleware rejects events without a valid device signature
	handler := func(_ context.Context, evt event.Event) error {
		fmt.Println("handling:", evt.Type())

		return nil
	}

	wrapped := verifyMiddleware(handler)

	aggID := id.NewAggregateID()
	unsignedEvt, _ := event.NewEvent("test.event", aggID, "Test", 1, nil)

	// Unsigned events pass through (to support mixed streams)
	_ = wrapped(context.Background(), unsignedEvt)

	// Signed events are verified before reaching the handler
	signedEvt, _ := deviceMulti.Sign(unsignedEvt)
	_ = wrapped(context.Background(), signedEvt)

	// Output:
	// handling: test.event
	// handling: test.event
}
