package signing_test

import (
	"crypto/ed25519"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/signing"
)

func newDeviceMultiSigner(t *testing.T) (*signing.MultiSigner, ed25519.PublicKey) {
	t.Helper()

	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	signer, signerErr := signing.NewEd25519(privKey)
	if signerErr != nil {
		t.Fatalf("create signer: %v", signerErr)
	}

	verifier, verifierErr := signing.NewEd25519Verifier(pubKey)
	if verifierErr != nil {
		t.Fatalf("create verifier: %v", verifierErr)
	}

	deviceMulti, err := signing.NewMultiSigner(
		signing.Actor("device"),
		signing.AlgorithmEd25519,
		signer,
		signing.WithVerifier(verifier),
	)
	if err != nil {
		t.Fatalf("create device multi-signer: %v", err)
	}

	return deviceMulti, pubKey
}

// newServerMultiSigner creates a test MultiSigner for the "server" actor using HMAC.
func newServerMultiSigner(t *testing.T) *signing.MultiSigner {
	t.Helper()

	key := []byte("server-secret-key-thirty-two-by!")
	signer, err := signing.NewHMAC(key)
	if err != nil {
		t.Fatalf("create HMAC signer: %v", err)
	}

	serverMulti, err := signing.NewMultiSigner(
		signing.Actor("server"),
		signing.AlgorithmHMACSHA256,
		signer,
	)
	if err != nil {
		t.Fatalf("create server multi-signer: %v", err)
	}

	return serverMulti
}
