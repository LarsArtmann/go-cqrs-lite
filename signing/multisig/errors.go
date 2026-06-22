package multisig

import (
	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// ErrNilSigner is returned when VerifierMap is called with a nil *MultiSigner.
var ErrNilSigner = event.NewRejection(
	"signing.nil_signer",
	"signing: VerifierMap called with nil *MultiSigner",
)

// ErrNoVerifier is returned when VerifyAll cannot find a verifier for an actor.
var ErrNoVerifier = event.NewRejection("signing.no_verifier", "no verifier provided for actor")
