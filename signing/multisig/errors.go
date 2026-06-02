package multisig

import (
	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// ErrNoVerifier is returned when VerifyAll cannot find a verifier for an actor.
var ErrNoVerifier = event.NewRejection("signing.no_verifier", "no verifier provided for actor")
