package signing

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// Signer computes cryptographic signatures for events.
// Implementations are stateless and safe for concurrent use.
type Signer interface {
	// Sign computes a cryptographic signature for the given event.
	// The signature covers event ID, type, stream, version, payload, and occurredAt.
	Sign(event event.Event) (Signature, error)
}

// Verifier checks cryptographic signatures on events.
// Implementations are stateless and safe for concurrent use.
type Verifier interface {
	// Verify checks that the signature matches the event's content.
	// Returns ErrInvalidSignature if the signature does not match.
	// Returns ErrNilSignature if sig is nil.
	Verify(event event.Event, sig Signature) error
}

// SignerVerifier combines signing and verification capabilities.
// NewHMAC returns a type that implements this interface because the same key handles both.
type SignerVerifier interface {
	Signer
	Verifier
}
