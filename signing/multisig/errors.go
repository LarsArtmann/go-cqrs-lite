package multisig

import (
	"errors"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// ErrNilSigner is returned when VerifierMap is called with a nil *MultiSigner.
var ErrNilSigner = errors.New("signing: VerifierMap called with nil *MultiSigner")

// ErrNoVerifier is returned when VerifyAll cannot find a verifier for an actor.
var ErrNoVerifier = event.NewRejection("signing.no_verifier", "no verifier provided for actor")
