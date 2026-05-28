package signing_test

import (
	"crypto/ed25519"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/signing"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func TestEd25519Signer(t *testing.T) {
	t.Parallel()

	_, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	t.Run("new with valid key", func(t *testing.T) {
		t.Parallel()

		_, err := signing.NewEd25519(privKey)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("new with invalid key", func(t *testing.T) {
		t.Parallel()

		_, err := signing.NewEd25519([]byte("short"))
		if err == nil {
			t.Fatal("expected error for invalid key")
		}
	})

	t.Run("sign valid event", func(t *testing.T) {
		t.Parallel()

		signer, _ := signing.NewEd25519(privKey)
		evt := makeTestEvent(t)

		sig, err := signer.Sign(evt)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		if sig.IsZero() {
			t.Fatal("expected non-zero signature")
		}
	})

	t.Run("sign nil event", func(t *testing.T) {
		t.Parallel()

		signer, _ := signing.NewEd25519(privKey)

		_, err := signer.Sign(nil)
		if err == nil {
			t.Fatal("expected error for nil event")
		}
	})
}

func TestEd25519Verifier(t *testing.T) {
	t.Parallel()

	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	signer, _ := signing.NewEd25519(privKey)
	verifier, _ := signing.NewEd25519Verifier(pubKey)

	evt := makeTestEvent(t)
	sig, _ := signer.Sign(evt)

	t.Run("verify valid signature", func(t *testing.T) {
		t.Parallel()

		err := verifier.Verify(evt, sig)
		if err != nil {
			t.Fatalf("verify: %v", err)
		}
	})

	t.Run("verify tampered event", func(t *testing.T) {
		t.Parallel()

		tampered := testhelpers.QuickEvent(
			evt.Type(),
			evt.AggregateID(),
			evt.AggregateType(),
			evt.Version(),
			[]byte(`{"tampered":true}`),
		)

		err := verifier.Verify(tampered, sig)
		if err == nil {
			t.Fatal("expected verification to fail")
		}
	})

	t.Run("verify wrong signature", func(t *testing.T) {
		t.Parallel()

		wrongSig := signing.Signature(make([]byte, ed25519.SignatureSize))

		err := verifier.Verify(evt, wrongSig)
		if err == nil {
			t.Fatal("expected verification to fail")
		}
	})

	t.Run("verify nil signature", func(t *testing.T) {
		t.Parallel()

		err := verifier.Verify(evt, nil)
		if err == nil {
			t.Fatal("expected error for nil signature")
		}
	})

	t.Run("verify nil event", func(t *testing.T) {
		t.Parallel()

		err := verifier.Verify(nil, sig)
		if err == nil {
			t.Fatal("expected error for nil event")
		}
	})

	t.Run("new with invalid public key", func(t *testing.T) {
		t.Parallel()

		_, err := signing.NewEd25519Verifier([]byte("short"))
		if err == nil {
			t.Fatal("expected error for invalid public key")
		}
	})
}

func TestEd25519KeyGeneration(t *testing.T) {
	t.Parallel()

	pubKey, privKey, err := signing.GenerateEd25519KeyPair()
	if err != nil {
		t.Fatalf("generate key pair: %v", err)
	}
	if len(pubKey) != ed25519.PublicKeySize {
		t.Fatalf("expected public key size %d, got %d", ed25519.PublicKeySize, len(pubKey))
	}
	if len(privKey) != ed25519.PrivateKeySize {
		t.Fatalf("expected private key size %d, got %d", ed25519.PrivateKeySize, len(privKey))
	}
}
