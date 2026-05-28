package signing

import (
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
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

// ErrNoVerifier is returned when VerifyAll cannot find a verifier for an actor.
var ErrNoVerifier = event.NewRejection("signing.no_verifier", "no verifier provided for actor")

// SignatureEntry is a single signature from one actor in the chain.
type SignatureEntry struct {
	// Actor is the identifier of the signing entity (e.g., "device", "server", "gateway").
	Actor string `json:"actor"`

	// Algorithm identifies the crypto algorithm used.
	Algorithm SignatureAlgorithm `json:"algorithm"`

	// Sig is the raw signature bytes (base64-encoded in JSON via Signature type).
	Sig Signature `json:"sig"`

	// SignedAt is when this actor produced the signature.
	SignedAt time.Time `json:"signedAt"`
}

// MultiSignature holds all signatures collected along the event's journey.
// Each actor adds their SignatureEntry without removing existing ones.
type MultiSignature struct {
	Entries []SignatureEntry `json:"entries"`
}

// Count returns the number of signature entries.
func (m MultiSignature) Count() int { return len(m.Entries) }

// HasActor reports whether the given actor has signed.
func (m MultiSignature) HasActor(actor string) bool {
	for _, entry := range m.Entries {
		if entry.Actor == actor {
			return true
		}
	}

	return false
}

// Get returns the signature entry for a given actor, or nil.
func (m MultiSignature) Get(actor string) *SignatureEntry {
	for idx := range m.Entries {
		if m.Entries[idx].Actor == actor {
			return &m.Entries[idx]
		}
	}

	return nil
}

// Actors returns a deduplicated list of all actor identifiers.
func (m MultiSignature) Actors() []string {
	seen := make(map[string]struct{}, len(m.Entries))
	result := make([]string, 0, len(m.Entries))

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
// For Ed25519, provide an Ed25519Signer for signing and an Ed25519Verifier
// via the WithVerifier option:
//
//	deviceMulti := signing.NewMultiSigner("device", signing.AlgorithmEd25519, ed25519Signer,
//	    signing.WithVerifier(ed25519Verifier))
//	serverMulti := signing.NewMultiSigner("server", signing.AlgorithmHMACSHA256, hmacSigner)
type MultiSigner struct {
	actor     string
	algorithm SignatureAlgorithm
	signer    Signer
	verifier  Verifier
	clock     Clock
}

// MultiSignerOption configures a MultiSigner.
type MultiSignerOption func(*MultiSigner)

// WithVerifier sets a separate verifier for the Verify path.
// Use this for Ed25519 where the signer (private key) cannot verify.
// For HMAC, the signer already implements both Sign and Verify.
func WithVerifier(verifier Verifier) MultiSignerOption {
	return func(multi *MultiSigner) { multi.verifier = verifier }
}

// WithClock sets a custom clock for deterministic SignedAt timestamps.
// Defaults to time.Now.
func WithClock(clock Clock) MultiSignerOption {
	return func(multi *MultiSigner) { multi.clock = clock }
}

// NewMultiSigner creates a signer for a named actor using the provided Signer.
// By default, the same Signer is used for both signing and verification.
// Pass WithVerifier to override the verifier (needed for Ed25519).
func NewMultiSigner(
	actor string,
	algorithm SignatureAlgorithm,
	signer Signer,
	opts ...MultiSignerOption,
) *MultiSigner {
	var verifier Verifier
	if sv, ok := signer.(Verifier); ok {
		verifier = sv
	}

	multi := &MultiSigner{
		actor:     actor,
		algorithm: algorithm,
		signer:    signer,
		verifier:  verifier,
		clock:     time.Now,
	}

	for _, opt := range opts {
		opt(multi)
	}

	return multi
}

// Actor returns the actor identifier.
func (m *MultiSigner) Actor() string { return m.actor }

// Algorithm returns the signature algorithm.
func (m *MultiSigner) Algorithm() SignatureAlgorithm { return m.algorithm }

// Sign computes a signature and appends it to the event's multi-sig collection.
// If the event already has signatures, they are preserved. The new signature
// is added to the end of the entries slice. If this actor has already signed,
// the previous entry is replaced.
func (m *MultiSigner) Sign(evt event.Event) (*event.ImmutableEvent, error) {
	if evt == nil {
		return nil, ErrNilEvent
	}

	sig, err := m.signer.Sign(evt)
	if err != nil {
		return nil, fmt.Errorf("sign event for actor %s: %w", m.actor, err)
	}

	existing, extractErr := ExtractMultiSignature(evt)

	var multiSig MultiSignature

	if extractErr == nil {
		multiSig = existing
	} else {
		multiSig = MultiSignature{Entries: []SignatureEntry{}}
	}

	multiSig.Entries = removeActor(multiSig.Entries, m.actor)
	multiSig.Entries = append(multiSig.Entries, SignatureEntry{
		Actor:     m.actor,
		Algorithm: m.algorithm,
		Sig:       sig,
		SignedAt:  m.clock(),
	})

	return attachMultiSignature(evt, multiSig)
}

// Verify verifies only this actor's signature from the event's multi-sig collection.
func (m *MultiSigner) Verify(evt event.Event) error {
	if evt == nil {
		return ErrNilEvent
	}

	multiSig, err := ExtractMultiSignature(evt)
	if err != nil {
		return err
	}

	entry := multiSig.Get(m.actor)
	if entry == nil {
		return fmt.Errorf("%w: no signature found for actor %s", ErrNilSignature, m.actor)
	}

	verifyErr := m.verifier.Verify(evt, entry.Sig)
	if verifyErr != nil {
		return fmt.Errorf("verify actor %s: %w", m.actor, verifyErr)
	}

	return nil
}

// VerifyActor verifies a specific actor's signature using the provided verifier.
// Useful when one actor wants to check another actor's signature.
func (m *MultiSigner) VerifyActor(evt event.Event, actor string, verifier Verifier) error {
	if evt == nil {
		return ErrNilEvent
	}

	multiSig, err := ExtractMultiSignature(evt)
	if err != nil {
		return err
	}

	entry := multiSig.Get(actor)
	if entry == nil {
		return fmt.Errorf("%w: no signature found for actor %s", ErrNilSignature, actor)
	}

	verifyErr := verifier.Verify(evt, entry.Sig)
	if verifyErr != nil {
		return fmt.Errorf("verify actor %s: %w", actor, verifyErr)
	}

	return nil
}
