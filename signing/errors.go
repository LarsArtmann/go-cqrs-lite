package signing

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

var (
	// ErrInvalidKey is returned when a signing key is empty or too short.
	ErrInvalidKey error = errorfamily.NewRejection(
		"signing.invalid_key",
		"signing key is empty or too short",
	)

	// ErrInvalidSignature is returned when verification fails.
	ErrInvalidSignature error = errorfamily.NewRejection(
		"signing.invalid_signature",
		"event signature verification failed",
	)

	// ErrNilSignature is returned when a nil/empty signature is provided to Verify.
	ErrNilSignature error = errorfamily.NewRejection(
		"signing.nil_signature",
		"signature is nil or empty",
	)

	// ErrNilEvent is returned when a nil event is passed to Sign or Verify.
	ErrNilEvent error = errorfamily.NewRejection(
		"signing.nil_event",
		"event is nil",
	)

	// ErrNilSigner is returned when a nil signer is passed to a COSE signing function.
	ErrNilSigner error = errorfamily.NewRejection(
		"signing.nil_signer",
		"signer is nil",
	)

	// ErrNilVerifier is returned when a nil verifier is passed to a COSE verification function.
	ErrNilVerifier error = errorfamily.NewRejection(
		"signing.nil_verifier",
		"verifier is nil",
	)
)
