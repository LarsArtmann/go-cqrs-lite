package multisig_test

import (
	"context"
	"crypto/ed25519"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/signing/v4"
	"github.com/larsartmann/go-cqrs-lite/signing/v4/multisig"
)

func ExampleNewMultiSigner() {
	key := []byte("example-key-with-at-least-32-bytes!")
	signerverifier, _ := signing.NewHMAC(key)

	deviceMulti, err := multisig.NewMultiSigner(
		multisig.Actor("device"),
		multisig.AlgorithmHMACSHA256,
		signerverifier,
	)
	if err != nil {
		panic(err)
	}

	aggID := id.NewStreamID()
	evt, _ := event.NewEvent("order.placed", aggID, "Order", 1, []byte(`{"total":99}`))

	signed, err := deviceMulti.Sign(evt)
	if err != nil {
		panic(err)
	}

	fmt.Println("actor:", deviceMulti.Actor())
	fmt.Println("has multi-sig:", multisig.HasMultiSignature(signed))

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

	deviceVerifier, _ := signing.NewEd25519Verifier(devicePub)
	serverVerifier, _ := signing.NewEd25519Verifier(serverPub)

	deviceMulti, _ := multisig.NewMultiSigner(
		multisig.Actor("device"),
		multisig.AlgorithmEd25519,
		deviceSigner,
		multisig.WithVerifier(deviceVerifier),
	)
	serverMulti, _ := multisig.NewMultiSigner(
		multisig.Actor("server"),
		multisig.AlgorithmEd25519,
		serverSigner,
		multisig.WithVerifier(serverVerifier),
	)

	aggID := id.NewStreamID()
	evt, _ := event.NewEvent("order.shipped", aggID, "Order", 1, []byte(`{}`))

	step1, _ := deviceMulti.Sign(evt)
	step2, _ := serverMulti.Sign(step1)

	verifiers, _ := multisig.VerifierMap(deviceMulti, serverMulti)

	if err := multisig.VerifyAll(
		step2,
		verifiers,
	); err != nil {
		fmt.Println("verification failed:", err)
	} else {
		fmt.Println("all signatures valid")
	}

	// Output:
	// all signatures valid
}

func ExampleVerifierMap() {
	key := []byte("example-key-with-at-least-32-bytes!")
	signer, _ := signing.NewHMAC(key)

	deviceMulti, _ := multisig.NewMultiSigner(
		multisig.Actor("device"),
		multisig.AlgorithmHMACSHA256,
		signer,
	)
	serverMulti, _ := multisig.NewMultiSigner(
		multisig.Actor("server"),
		multisig.AlgorithmHMACSHA256,
		signer,
	)

	verifiers, _ := multisig.VerifierMap(deviceMulti, serverMulti)
	fmt.Println("device actor:", verifiers[multisig.Actor("device")] != nil)
	fmt.Println("server actor:", verifiers[multisig.Actor("server")] != nil)
	fmt.Println("total:", len(verifiers))

	// Output:
	// device actor: true
	// server actor: true
	// total: 2
}

func ExampleMultiVerifyMiddlewareFor() {
	_, devicePriv, _ := ed25519.GenerateKey(nil)
	devicePub := devicePriv.Public().(ed25519.PublicKey) //nolint:forcetypeassert // ed25519.GenerateKey always returns ed25519.PublicKey

	deviceSigner, _ := signing.NewEd25519(devicePriv)
	deviceVerifier, _ := signing.NewEd25519Verifier(devicePub)
	deviceMulti, _ := multisig.NewMultiSigner(
		multisig.Actor("device"),
		multisig.AlgorithmEd25519,
		deviceSigner,
		multisig.WithVerifier(deviceVerifier),
	)

	verifyMiddleware := multisig.MultiVerifyMiddlewareFor(multisig.Actor("device"), deviceVerifier)

	handler := func(_ context.Context, evt event.Event) error {
		fmt.Println("handling:", evt.Type())

		return nil
	}

	wrapped := verifyMiddleware(handler)

	aggID := id.NewStreamID()
	unsignedEvt, _ := event.NewEvent("test.event", aggID, "Test", 1, nil)

	_ = wrapped(context.Background(), unsignedEvt)

	signedEvt, _ := deviceMulti.Sign(unsignedEvt)
	_ = wrapped(context.Background(), signedEvt)

	// Output:
	// handling: test.event
	// handling: test.event
}
