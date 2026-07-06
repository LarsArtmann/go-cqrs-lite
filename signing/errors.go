package signing

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

var (
	// ErrInvalidKey is returned when a signing key is empty or too short.
	ErrInvalidKey = errorfamily.NewRejection(
		"signing.invalid_key",
		"signing key is empty or too short",
	)

	// ErrInvalidSignature is returned when verification fails.
	ErrInvalidSignature = errorfamily.NewRejection(
		"signing.invalid_signature",
		"event signature verification failed",
	)

	// ErrNilSignature is returned when a nil/empty signature is provided to Verify.
	ErrNilSignature = errorfamily.NewRejection(
		"signing.nil_signature",
		"signature is nil or empty",
	)

	// ErrNilEvent is returned when a nil event is passed to Sign or Verify.
	ErrNilEvent = errorfamily.NewRejection(
		"signing.nil_event",
		"event is nil",
	)
)
