package signing_test

import (
	"crypto/ed25519"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/signing"
)

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

	tampered := tamperEvent(t, clone)
	tampered.Metadata().Custom = md.Custom

	if err := deviceMulti.Verify(tampered); err == nil {
		t.Fatal("expected verification to fail for tampered event")
	}
}

func TestMultiSigner_VerifyActor(t *testing.T) {
	t.Parallel()

	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	edSigner, _ := signing.NewEd25519(privKey)
	edVerifier, _ := signing.NewEd25519Verifier(pubKey)

	deviceMulti, err := signing.NewMultiSigner(
		signing.Actor("device"),
		signing.AlgorithmEd25519,
		edSigner,
		signing.WithVerifier(edVerifier),
	)
	if err != nil {
		t.Fatalf("create device multi-signer: %v", err)
	}
	serverMulti := newServerMultiSigner(t)
	evt := makeTestEvent(t)

	clone1, _ := deviceMulti.Sign(evt)
	clone2, _ := serverMulti.Sign(clone1)

	if verifyErr := serverMulti.VerifyActor(
		clone2,
		signing.Actor("device"),
		edVerifier,
	); verifyErr != nil {
		t.Fatalf("server verifying device: %v", verifyErr)
	}
}

func TestMultiSigner_Verify_NilEvent(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)

	if err := deviceMulti.Verify(nil); err == nil {
		t.Fatal("expected error for nil event")
	}
}

func TestMultiSigner_VerifyActor_NilEvent(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	pubKey, _, _ := ed25519.GenerateKey(nil)
	verifier, _ := signing.NewEd25519Verifier(pubKey)

	if err := deviceMulti.VerifyActor(nil, signing.Actor("device"), verifier); err == nil {
		t.Fatal("expected error for nil event")
	}
}

func TestMultiSigner_VerifyActor_NoSignature(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	serverMulti := newServerMultiSigner(t)

	evt := makeTestEvent(t)
	// Only server signs, then we try to verify device
	clone, _ := serverMulti.Sign(evt)

	pubKey, _, _ := ed25519.GenerateKey(nil)
	verifier, _ := signing.NewEd25519Verifier(pubKey)

	if err := deviceMulti.VerifyActor(clone, signing.Actor("device"), verifier); err == nil {
		t.Fatal("expected error when actor has no signature")
	}
}

func TestMultiSigner_VerifyActor_BadSignature(t *testing.T) {
	t.Parallel()

	deviceMulti, devicePubKey := newDeviceMultiSigner(t)
	evt := makeTestEvent(t)
	clone, _ := deviceMulti.Sign(evt)

	// Tamper with the event after signing
	tampered := tamperEvent(t, clone)

	deviceVerifier, _ := signing.NewEd25519Verifier(devicePubKey)

	if err := deviceMulti.VerifyActor(
		tampered,
		signing.Actor("device"),
		deviceVerifier,
	); err == nil {
		t.Fatal("expected error for tampered event")
	}
}
