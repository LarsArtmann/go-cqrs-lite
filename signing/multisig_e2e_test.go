package signing_test

import (
	"crypto/ed25519"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/signing"
)

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

	deviceMulti, err := signing.NewMultiSigner(
		signing.Actor("device"),
		signing.AlgorithmEd25519,
		deviceSigner,
		signing.WithVerifier(deviceVerifier),
	)
	if err != nil {
		t.Fatalf("create device multi-signer: %v", err)
	}

	serverMulti, err := signing.NewMultiSigner(
		signing.Actor("server"),
		signing.AlgorithmHMACSHA256,
		serverSigner,
	)
	if err != nil {
		t.Fatalf("create server multi-signer: %v", err)
	}

	// Step 1: Device creates and signs the event.
	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	deviceEvent, evtErr := event.NewEvent(
		"user.created",
		aggID,
		"User",
		1,
		[]byte(`{"name":"Alice"}`),
	)
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
