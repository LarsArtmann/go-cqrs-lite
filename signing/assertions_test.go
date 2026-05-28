package signing

import "github.com/larsartmann/go-cqrs-lite/core/event"

// Compile-time interface assertions.
var (
	_ Signer          = (*HMACSigner)(nil)
	_ Verifier        = (*HMACSigner)(nil)
	_ SignerVerifier  = (*HMACSigner)(nil)
	_ Signer          = (*Ed25519Signer)(nil)
	_ Verifier        = (*Ed25519Verifier)(nil)
	_ event.Publisher = event.PublisherFunc(nil)
)
