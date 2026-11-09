package multisig

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// ErrNilSigner is returned when VerifierMap is called with a nil *MultiSigner.
var ErrNilSigner error = errorfamily.NewRejection(
	"signing.nil_signer",
	"signing: VerifierMap called with nil *MultiSigner",
)

// ErrNoVerifier is returned when VerifyAll cannot find a verifier for an actor.
var ErrNoVerifier error = errorfamily.NewRejection(
	"signing.no_verifier",
	"no verifier provided for actor",
)
