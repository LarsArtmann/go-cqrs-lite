package signing

import (
	"crypto/hmac"
	"crypto/sha256"
	"slices"

	"github.com/larsartmann/go-cqrs-lite/event"
)

// MinimumKeyLength is the minimum acceptable key length in bytes for HMAC-SHA256.
// Keys shorter than this are rejected to prevent weak security.
const MinimumKeyLength = 32

// hmacSigner signs and verifies events using HMAC-SHA256.
// Use for shared-secret scenarios where all participants trust each other
// (e.g., same-organization microservices).
type hmacSigner struct {
	key []byte
}

var _ SignerVerifier = (*hmacSigner)(nil)

// NewHMAC creates an HMAC-SHA256 signer from a shared secret key.
// Returns a SignerVerifier that handles both signing and verification.
// Returns ErrInvalidKey if the key is nil or shorter than MinimumKeyLength.
func NewHMAC(key []byte) (*hmacSigner, error) {
	if len(key) < MinimumKeyLength {
		return nil, event.Wrapf(
			ErrInvalidKey, event.Rejection,
			"signing.hmac_key_too_short",
			"HMAC key length %d < minimum %d",
			len(key),
			MinimumKeyLength,
		)
	}

	// Copy key to prevent external mutation
	keyCopy := slices.Clone(key)

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

	if evt == nil {
		return ErrNilEvent
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
