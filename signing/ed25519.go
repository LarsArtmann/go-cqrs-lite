package signing

import (
	"crypto/ed25519"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// Ed25519Signer signs events with Ed25519 private keys.
// Use for asymmetric scenarios where verifiers don't have signing access
// (e.g., client devices sign, server verifies).
type Ed25519Signer struct {
	privateKey ed25519.PrivateKey
}

var _ Signer = (*Ed25519Signer)(nil)

// NewEd25519 creates an Ed25519 signer from a private key.
// Returns ErrInvalidKey if the key is nil or the wrong length.
func NewEd25519(privateKey ed25519.PrivateKey) (*Ed25519Signer, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf(
			"%w: expected Ed25519 private key of %d bytes, got %d",
			ErrInvalidKey,
			ed25519.PrivateKeySize,
			len(privateKey),
		)
	}

	// Copy to prevent external mutation
	keyCopy := make([]byte, ed25519.PrivateKeySize)
	copy(keyCopy, privateKey)

	return &Ed25519Signer{privateKey: keyCopy}, nil
}

// Sign computes an Ed25519 signature for the event.
func (s *Ed25519Signer) Sign(evt event.Event) (Signature, error) {
	if evt == nil {
		return nil, ErrNilEvent
	}

	canonical := canonicalPayload(evt)

	sig := ed25519.Sign(s.privateKey, canonical)

	return Signature(sig), nil
}

// Ed25519Verifier verifies Ed25519 signatures using a public key.
type Ed25519Verifier struct {
	publicKey ed25519.PublicKey
}

var _ Verifier = (*Ed25519Verifier)(nil)

// NewEd25519Verifier creates a verifier from an Ed25519 public key.
// Returns ErrInvalidKey if the key is nil or the wrong length.
func NewEd25519Verifier(publicKey ed25519.PublicKey) (*Ed25519Verifier, error) {
	if len(publicKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf(
			"%w: expected Ed25519 public key of %d bytes, got %d",
			ErrInvalidKey,
			ed25519.PublicKeySize,
			len(publicKey),
		)
	}

	// Copy to prevent external mutation
	keyCopy := make([]byte, ed25519.PublicKeySize)
	copy(keyCopy, publicKey)

	return &Ed25519Verifier{publicKey: keyCopy}, nil
}

// Verify checks an Ed25519 signature against an event using the public key.
func (v *Ed25519Verifier) Verify(evt event.Event, sig Signature) error {
	if sig.IsZero() {
		return ErrNilSignature
	}

	if evt == nil {
		return ErrNilEvent
	}

	canonical := canonicalPayload(evt)

	if !ed25519.Verify(v.publicKey, canonical, sig) {
		return ErrInvalidSignature
	}

	return nil
}

// GenerateEd25519KeyPair generates a new Ed25519 key pair for signing/verification.
// The private key can be used with NewEd25519; the public key with NewEd25519Verifier.
func GenerateEd25519KeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, nil, fmt.Errorf("generate ed25519 key pair: %w", err)
	}

	return pub, priv, nil
}
