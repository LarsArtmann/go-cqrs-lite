package signing

import (
	"github.com/larsartmann/go-cqrs-lite/core/event"
)

var (
	// ErrInvalidKey is returned when a signing key is empty or too short.
	ErrInvalidKey = event.NewRejection(
		"signing.invalid_key",
		"signing key is empty or too short",
	)

	// ErrInvalidSignature is returned when verification fails.
	ErrInvalidSignature = event.NewRejection(
		"signing.invalid_signature",
		"event signature verification failed",
	)

	// ErrNilSignature is returned when a nil/empty signature is provided to Verify.
	ErrNilSignature = event.NewRejection(
		"signing.nil_signature",
		"signature is nil or empty",
	)

	// ErrNilEvent is returned when a nil event is passed to Sign or Verify.
	ErrNilEvent = event.NewRejection(
		"signing.nil_event",
		"event is nil",
	)

	// ErrAlgorithmMismatch is returned when the signing algorithm doesn't match
	// the key type (e.g., Ed25519 key with HMAC signer).
	ErrAlgorithmMismatch = event.NewRejection(
		"signing.algorithm_mismatch",
		"signing algorithm does not match key type",
	)
)
