package multisig_test

import (
	"crypto/ed25519"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
	"github.com/larsartmann/go-cqrs-lite/signing/v4"
	"github.com/larsartmann/go-cqrs-lite/signing/v4/internal/testutil"
	"github.com/larsartmann/go-cqrs-lite/signing/v4/multisig"
)

func TestExtractMultiSignature(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	evt := testutil.MakeTestEvent(t)

	t.Run("extract from unsigned event", func(t *testing.T) {
		t.Parallel()
		if _, err := multisig.ExtractMultiSignature(evt); err == nil {
			t.Fatal("expected error for unsigned event")
		}
	})

	t.Run("extract from nil event", func(t *testing.T) {
		t.Parallel()
		if _, err := multisig.ExtractMultiSignature(nil); err == nil {
			t.Fatal("expected error for nil event")
		}
	})

	t.Run("has multi-sig", func(t *testing.T) {
		t.Parallel()
		if multisig.HasMultiSignature(evt) {
			t.Fatal("original event should not have multi-sig")
		}

		clone, _ := deviceMulti.Sign(evt)
		if !multisig.HasMultiSignature(clone) {
			t.Fatal("signed event should have multi-sig")
		}
	})
}

func TestVerifyAll_MissingVerifier(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	evt := testutil.MakeTestEvent(t)

	clone, _ := deviceMulti.Sign(evt)

	verifiers := map[multisig.Actor]signing.Verifier{}
	err := multisig.VerifyAll(clone, verifiers)
	if err == nil {
		t.Fatal("expected error for missing verifier")
	}
}

func TestVerifyAll_FailingVerifier(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	evt := testutil.MakeTestEvent(t)

	clone, _ := deviceMulti.Sign(evt)
	tampered := testutil.TamperEvent(t, clone)

	pubKey, _, _ := ed25519.GenerateKey(nil)
	verifier, _ := signing.NewEd25519Verifier(pubKey)

	verifiers := map[multisig.Actor]signing.Verifier{multisig.Actor("device"): verifier}
	err := multisig.VerifyAll(tampered, verifiers)
	if err == nil {
		t.Fatal("expected error for tampered event with wrong verifier")
	}
}

func TestExtractMultiSignature_InvalidJSON(t *testing.T) {
	t.Parallel()

	streamID := idtest.ParseStreamID(t, "01HK1540X0841Y0A6BSX1VKR95")
	evt, err := event.NewEvent(
		"test.invalid", streamID, "Test", 1, []byte(`{}`),
		event.WithMetadata(event.Metadata{
			Custom: map[event.MetadataKey]string{
				multisig.MultiSigMetadataKey: `{invalid json`,
			},
		}),
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	_, extractErr := multisig.ExtractMultiSignature(evt)
	if extractErr == nil {
		t.Fatal("expected error for invalid JSON in multi-sig metadata")
	}
}

func TestVerifyAll_NilEvent(t *testing.T) {
	t.Parallel()

	verifiers := map[multisig.Actor]signing.Verifier{}
	if err := multisig.VerifyAll(nil, verifiers); err == nil {
		t.Fatal("expected error for nil event")
	}
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

	deviceMulti, err := multisig.NewMultiSigner(
		multisig.Actor("device"),
		multisig.AlgorithmEd25519,
		deviceSigner,
		multisig.WithVerifier(deviceVerifier),
	)
	if err != nil {
		t.Fatalf("create device multi-signer: %v", err)
	}

	serverMulti, err := multisig.NewMultiSigner(
		multisig.Actor("server"),
		multisig.AlgorithmHMACSHA256,
		serverSigner,
	)
	if err != nil {
		t.Fatalf("create server multi-signer: %v", err)
	}

	streamID := idtest.ParseStreamID(t, "01HK1540X0841Y0A6BSX1VKR95")
	deviceEvent, evtErr := event.NewEvent(
		"user.created",
		streamID,
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

	if verifyErr := deviceMulti.Verify(deviceSigned); verifyErr != nil {
		t.Fatalf("server verifies device: %v", verifyErr)
	}

	serverSigned, err := serverMulti.Sign(deviceSigned)
	if err != nil {
		t.Fatalf("server sign: %v", err)
	}

	extracted, err := multisig.ExtractMultiSignature(serverSigned)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if extracted.Count() != 2 {
		t.Fatalf("expected 2 signatures, got %d", extracted.Count())
	}

	if verifyErr := deviceMulti.Verify(serverSigned); verifyErr != nil {
		t.Fatalf("verify device on final: %v", verifyErr)
	}
	if verifyErr := serverMulti.Verify(serverSigned); verifyErr != nil {
		t.Fatalf("verify server on final: %v", verifyErr)
	}

	verifiers := map[multisig.Actor]signing.Verifier{
		multisig.Actor("device"): deviceVerifier,
		multisig.Actor("server"): serverSigner,
	}
	if verifyErr := multisig.VerifyAll(serverSigned, verifiers); verifyErr != nil {
		t.Fatalf("verify all: %v", verifyErr)
	}

	tampered := eventtest.TamperEvent(serverSigned, []byte(`{"name":"Bob"}`))

	if verifyErr := deviceMulti.Verify(tampered); verifyErr == nil {
		t.Fatal("expected verification to fail for tampered event")
	}
}
