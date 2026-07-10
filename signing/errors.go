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

	// ErrNilSigner is returned when a nil signer is passed to a COSE signing function.
	ErrNilSigner = errorfamily.NewRejection(
		"signing.nil_signer",
		"signer is nil",
	)

	// ErrNilVerifier is returned when a nil verifier is passed to a COSE verification function.
	ErrNilVerifier = errorfamily.NewRejection(
		"signing.nil_verifier",
		"verifier is nil",
	)

	// ErrCOSEAlgorithmOverflow is returned when a COSE algorithm value cannot fit in int64.
	ErrCOSEAlgorithmOverflow = errorfamily.NewRejection(
		"signing.cose_algorithm_overflow",
		"COSE algorithm value overflows int64",
	)

	// ErrCOSEInvalidAlgorithm is returned when a COSE algorithm value is not an integer.
	ErrCOSEInvalidAlgorithm = errorfamily.NewRejection(
		"signing.cose_invalid_algorithm",
		"COSE algorithm value is not an integer",
	)
)
