package multisig

import (
	"slices"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/signing/v3"
)

// MultiSigMetadataKey is the custom metadata key used to store multi-party signatures.
const MultiSigMetadataKey event.MetadataKey = "event.multisig"

// SignatureAlgorithm identifies the cryptographic algorithm used for a signature.
type SignatureAlgorithm string

const (
	// AlgorithmHMACSHA256 identifies HMAC-SHA256 signatures.
	AlgorithmHMACSHA256 SignatureAlgorithm = "HMAC-SHA256"
	// AlgorithmEd25519 identifies Ed25519 signatures.
	AlgorithmEd25519 SignatureAlgorithm = "Ed25519"
)

// Actor identifies a signing entity in a multi-party chain (e.g., "device", "server").
type Actor string

// String returns the actor identifier.
func (a Actor) String() string { return string(a) }

// SignatureEntry is a single signature from one actor in the chain.
type SignatureEntry struct {
	// Actor is the identifier of the signing entity (e.g., "device", "server", "gateway").
	Actor Actor `json:"actor"`

	// Algorithm identifies the crypto algorithm used.
	Algorithm SignatureAlgorithm `json:"algorithm"`

	// Sig is the raw signature bytes (base64-encoded in JSON via Signature type).
	Sig signing.Signature `json:"sig"`

	// SignedAt is when this actor produced the signature.
	SignedAt time.Time `json:"signedAt"`
}

// Validate checks that the signature entry has all required fields.
func (e SignatureEntry) Validate() error {
	if e.Actor == "" {
		return errorfamily.NewRejection(
			"signing.empty_actor",
			"signature entry actor cannot be empty",
		)
	}

	if e.Algorithm == "" {
		return errorfamily.NewRejection(
			"signing.empty_algorithm",
			"signature entry algorithm cannot be empty",
		)
	}

	if e.Sig.IsZero() {
		return errorfamily.NewRejection("signing.empty_sig", "signature entry sig cannot be empty")
	}

	if e.SignedAt.IsZero() {
		return errorfamily.NewRejection(
			"signing.empty_signed_at",
			"signature entry signedAt cannot be zero",
		)
	}

	return nil
}

// MultiSignature holds all signatures collected along the event's journey.
// Each actor adds their SignatureEntry without removing existing ones.
type MultiSignature struct {
	Entries []SignatureEntry `json:"entries"`
}

// Count returns the number of signature entries.
func (m MultiSignature) Count() int { return len(m.Entries) }

// HasActor reports whether the given actor has signed.
func (m MultiSignature) HasActor(actor Actor) bool {
	return slices.ContainsFunc(m.Entries, func(e SignatureEntry) bool { return e.Actor == actor })
}

// Get returns a copy of the signature entry for a given actor, or nil.
// The returned pointer references an independent copy; mutations will not
// affect the MultiSignature's internal state.
func (m MultiSignature) Get(actor Actor) *SignatureEntry {
	if idx := slices.IndexFunc(
		m.Entries,
		func(e SignatureEntry) bool { return e.Actor == actor },
	); idx >= 0 {
		cp := m.Entries[idx]

		return &cp
	}

	return nil
}

// Actors returns a deduplicated list of all actor identifiers.
func (m MultiSignature) Actors() []Actor {
	seen := make(map[Actor]struct{}, len(m.Entries))
	result := make([]Actor, 0, len(m.Entries))

	for _, entry := range m.Entries {
		if _, ok := seen[entry.Actor]; !ok {
			seen[entry.Actor] = struct{}{}
			result = append(result, entry.Actor)
		}
	}

	return result
}

// Clock returns the current time. Override for deterministic testing.
type Clock func() time.Time

// MultiSigner signs events on behalf of a specific actor, appending to existing
// multi-signature entries without removing prior signatures.
//
// For HMAC, the same Signer handles both signing and verification.
// For Ed25519, create a signer with signing.NewEd25519 and a verifier with signing.NewEd25519Verifier,
// then pass the verifier via the WithVerifier option:
//
//	deviceMulti := multisig.NewMultiSigner("device", multisig.AlgorithmEd25519, ed25519Signer,
//	    multisig.WithVerifier(ed25519Verifier))
//	serverMulti := multisig.NewMultiSigner("server", multisig.AlgorithmHMACSHA256, hmacSigner)
type MultiSigner struct {
	actor     Actor
	algorithm SignatureAlgorithm
	signer    signing.Signer
	verifier  signing.Verifier
	clock     Clock
}

// MultiSignerOption configures a MultiSigner.
type MultiSignerOption func(*MultiSigner)

// WithVerifier sets a separate verifier for the Verify path.
// Use this for Ed25519 where the signer (private key) cannot verify.
// For HMAC, the signer already implements both Sign and Verify.
func WithVerifier(verifier signing.Verifier) MultiSignerOption {
	return func(multi *MultiSigner) { multi.verifier = verifier }
}

// WithClock sets a custom clock for deterministic SignedAt timestamps.
// Defaults to time.Now.
func WithClock(clock Clock) MultiSignerOption {
	return func(multi *MultiSigner) { multi.clock = clock }
}
