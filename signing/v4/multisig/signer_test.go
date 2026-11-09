package multisig_test

import (
	"crypto/ed25519"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/signing/v4"
	"github.com/larsartmann/go-cqrs-lite/signing/v4/internal/testutil"
	"github.com/larsartmann/go-cqrs-lite/signing/v4/multisig"
)

func TestMultiSigner_SignAddsActor(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	evt := testutil.MakeTestEvent(t)

	clone, err := deviceMulti.Sign(evt)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	extracted, err := multisig.ExtractMultiSignature(clone)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if extracted.Count() != 1 {
		t.Fatalf("expected 1 entry, got %d", extracted.Count())
	}
	if !extracted.HasActor(multisig.Actor("device")) {
		t.Fatal("expected device actor")
	}
}

func TestMultiSigner_MultipleActors(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	serverMulti := newServerMultiSigner(t)
	evt := testutil.MakeTestEvent(t)

	clone1, err := deviceMulti.Sign(evt)
	if err != nil {
		t.Fatalf("device sign: %v", err)
	}

	clone2, err := serverMulti.Sign(clone1)
	if err != nil {
		t.Fatalf("server sign: %v", err)
	}

	extracted, err := multisig.ExtractMultiSignature(clone2)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if extracted.Count() != 2 {
		t.Fatalf("expected 2 entries, got %d", extracted.Count())
	}
	if !extracted.HasActor(multisig.Actor("device")) ||
		!extracted.HasActor(multisig.Actor("server")) {
		t.Fatal("expected both device and server actors")
	}
}

func TestMultiSigner_ReSignReplaces(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	evt := testutil.MakeTestEvent(t)

	clone1, err := deviceMulti.Sign(evt)
	if err != nil {
		t.Fatalf("first sign: %v", err)
	}

	extracted1, _ := multisig.ExtractMultiSignature(clone1)
	entry1 := extracted1.Get(multisig.Actor("device"))
	if entry1 == nil {
		t.Fatal("expected device entry after first sign")
	}

	clone2, err := deviceMulti.Sign(clone1)
	if err != nil {
		t.Fatalf("second sign: %v", err)
	}

	extracted2, _ := multisig.ExtractMultiSignature(clone2)
	if extracted2.Count() != 1 {
		t.Fatalf("expected 1 entry after re-sign, got %d", extracted2.Count())
	}

	entry2 := extracted2.Get(multisig.Actor("device"))
	if entry2.SignedAt.Equal(entry1.SignedAt) {
		t.Fatal("re-signed entry should have different timestamp")
	}
}

func TestMultiSigner_NilEvent(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)

	if _, err := deviceMulti.Sign(nil); err == nil {
		t.Fatal("expected error for nil event on sign")
	}

	if err := deviceMulti.Verify(nil); err == nil {
		t.Fatal("expected error for nil event on verify")
	}
}

func TestMultiSigner_VerifyOwnSignature(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	evt := testutil.MakeTestEvent(t)

	clone, _ := deviceMulti.Sign(evt)
	if err := deviceMulti.Verify(clone); err != nil {
		t.Fatalf("verify device: %v", err)
	}
}

func TestMultiSigner_VerifyDualSigned(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	serverMulti := newServerMultiSigner(t)
	evt := testutil.MakeTestEvent(t)

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
	evt := testutil.MakeTestEvent(t)

	clone, _ := deviceMulti.Sign(evt)
	if err := serverMulti.Verify(clone); err == nil {
		t.Fatal("expected error verifying missing actor")
	}
}

func TestMultiSigner_VerifyTampered(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	evt := testutil.MakeTestEvent(t)

	clone, _ := deviceMulti.Sign(evt)
	origMD := clone.Metadata()

	tampered := testutil.TamperEvent(t, clone)
	_ = origMD // tampered already has different metadata; test verifies Verify detects tampering

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

	deviceMulti, err := multisig.NewMultiSigner(
		multisig.Actor("device"),
		multisig.AlgorithmEd25519,
		edSigner,
		multisig.WithVerifier(edVerifier),
	)
	if err != nil {
		t.Fatalf("create device multi-signer: %v", err)
	}
	serverMulti := newServerMultiSigner(t)
	evt := testutil.MakeTestEvent(t)

	clone1, _ := deviceMulti.Sign(evt)
	clone2, _ := serverMulti.Sign(clone1)

	if verifyErr := serverMulti.VerifyActor(
		clone2,
		multisig.Actor("device"),
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

	if err := deviceMulti.VerifyActor(nil, multisig.Actor("device"), verifier); err == nil {
		t.Fatal("expected error for nil event")
	}
}

func TestMultiSigner_VerifyActor_NoSignature(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	serverMulti := newServerMultiSigner(t)

	evt := testutil.MakeTestEvent(t)
	clone, _ := serverMulti.Sign(evt)

	pubKey, _, _ := ed25519.GenerateKey(nil)
	verifier, _ := signing.NewEd25519Verifier(pubKey)

	if err := deviceMulti.VerifyActor(clone, multisig.Actor("device"), verifier); err == nil {
		t.Fatal("expected error when actor has no signature")
	}
}

func TestMultiSigner_VerifyActor_BadSignature(t *testing.T) {
	t.Parallel()

	deviceMulti, devicePubKey := newDeviceMultiSigner(t)
	evt := testutil.MakeTestEvent(t)
	clone, _ := deviceMulti.Sign(evt)

	tampered := testutil.TamperEvent(t, clone)

	deviceVerifier, _ := signing.NewEd25519Verifier(devicePubKey)

	if err := deviceMulti.VerifyActor(
		tampered,
		multisig.Actor("device"),
		deviceVerifier,
	); err == nil {
		t.Fatal("expected error for tampered event")
	}
}
