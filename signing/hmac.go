package signing

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// MinimumKeyLength is the minimum acceptable key length in bytes for HMAC-SHA256.
// Keys shorter than this are rejected to prevent weak security.
const MinimumKeyLength = 32

// HMACSigner signs and verifies events using HMAC-SHA256.
// Use for shared-secret scenarios where all participants trust each other
// (e.g., same-organization microservices).
type hmacSigner struct {
	key []byte
}

var _ SignerVerifier = (*hmacSigner)(nil)

// NewHMAC creates an HMAC-SHA256 signer from a shared secret key.
// Returns ErrInvalidKey if the key is nil or shorter than MinimumKeyLength.
func NewHMAC(key []byte) (*hmacSigner, error) {
	if len(key) < MinimumKeyLength {
		return nil, fmt.Errorf(
			"%w: key length %d < minimum %d",
			ErrInvalidKey,
			len(key),
			MinimumKeyLength,
		)
	}

	// Copy key to prevent external mutation
	keyCopy := make([]byte, len(key))
	copy(keyCopy, key)

	return &hmacSigner{key: keyCopy}, nil
}

// Sign computes an HMAC-SHA256 signature for the event.
func (s *hmacSigner) Sign(evt event.Event) (Signature, error) {
	if evt == nil {
		return nil, ErrNilEvent
	}

	canonical := canonicalPayload(evt)

	mac := hmac.New(sha256.New, s.key)
	mac.Write(canonical)

	return Signature(mac.Sum(nil)), nil
}

// Verify checks that the HMAC-SHA256 signature matches the event.
func (s *hmacSigner) Verify(evt event.Event, sig Signature) error {
	if sig.IsZero() {
		return ErrNilSignature
	}

	expected, err := s.Sign(evt)
	if err != nil {
		return err
	}

	if !expected.Equal(sig) {
		return ErrInvalidSignature
	}

	return nil
}
