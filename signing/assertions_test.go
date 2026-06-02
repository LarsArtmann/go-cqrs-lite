package signing

import "github.com/larsartmann/go-cqrs-lite/event/v2"

// Compile-time interface assertions.
var (
	_ Signer          = (*hmacSigner)(nil)
	_ Verifier        = (*hmacSigner)(nil)
	_ SignerVerifier  = (*hmacSigner)(nil)
	_ Signer          = (*ed25519Signer)(nil)
	_ Verifier        = (*ed25519Verifier)(nil)
	_ event.Publisher = event.PublisherFunc(nil)
)
