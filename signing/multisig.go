package signing

import (
	"encoding/json"
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
	for _, e := range m.Entries {
		if e.Actor == actor {
			return true
		}
	}

	return false
}

// Get returns the signature entry for a given actor, or nil.
func (m MultiSignature) Get(actor string) *SignatureEntry {
	for i := range m.Entries {
		if m.Entries[i].Actor == actor {
			return &m.Entries[i]
		}
	}

	return nil
}

// Actors returns a deduplicated list of all actor identifiers.
func (m MultiSignature) Actors() []string {
	seen := make(map[string]struct{}, len(m.Entries))
	result := make([]string, 0, len(m.Entries))

	for _, e := range m.Entries {
		if _, ok := seen[e.Actor]; !ok {
			seen[e.Actor] = struct{}{}
			result = append(result, e.Actor)
		}
	}

	return result
}

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
	verifier  Signer
}

// MultiSignerOption configures a MultiSigner.
type MultiSignerOption func(*MultiSigner)

// WithVerifier sets a separate verifier for the Verify path.
// Use this for Ed25519 where the signer (private key) cannot verify.
// For HMAC, the signer already implements both Sign and Verify.
func WithVerifier(verifier Signer) MultiSignerOption {
	return func(ms *MultiSigner) { ms.verifier = verifier }
}

// NewMultiSigner creates a signer for a named actor using the provided Signer.
// By default, the same Signer is used for both signing and verification.
// Pass WithVerifier to override the verifier (needed for Ed25519).
func NewMultiSigner(actor string, algorithm SignatureAlgorithm, signer Signer, opts ...MultiSignerOption) *MultiSigner {
	ms := &MultiSigner{
		actor:     actor,
		algorithm: algorithm,
		signer:    signer,
		verifier:  signer, // default: same as signer (works for HMAC)
	}

	for _, opt := range opts {
		opt(ms)
	}

	return ms
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

	ms, _ := ExtractMultiSignature(evt)
	if ms.Count() == 0 {
		ms = MultiSignature{}
	}

	// Remove any prior entry from this actor (prevents duplicates).
	ms.Entries = removeActor(ms.Entries, m.actor)
	ms.Entries = append(ms.Entries, SignatureEntry{
		Actor:     m.actor,
		Algorithm: m.algorithm,
		Sig:       sig,
		SignedAt:  time.Now(),
	})

	return attachMultiSignature(evt, ms)
}

// Verify verifies only this actor's signature from the event's multi-sig collection.
func (m *MultiSigner) Verify(evt event.Event) error {
	if evt == nil {
		return ErrNilEvent
	}

	ms, err := ExtractMultiSignature(evt)
	if err != nil {
		return err
	}

	entry := ms.Get(m.actor)
	if entry == nil {
		return fmt.Errorf("%w: no signature found for actor %s", ErrNilSignature, m.actor)
	}

	return m.verifier.Verify(evt, entry.Sig)
}

// VerifyActor verifies a specific actor's signature.
// Useful when one actor wants to check another actor's signature.
func (m *MultiSigner) VerifyActor(evt event.Event, actor string, verifier Signer) error {
	if evt == nil {
		return ErrNilEvent
	}

	ms, err := ExtractMultiSignature(evt)
	if err != nil {
		return err
	}

	entry := ms.Get(actor)
	if entry == nil {
		return fmt.Errorf("%w: no signature found for actor %s", ErrNilSignature, actor)
	}

	return verifier.Verify(evt, entry.Sig)
}

// VerifyAll verifies every signature entry in the multi-sig collection
// using each entry's algorithm-appropriate verifier from the provided map.
// The map keys are actor names; the values are their respective verifiers.
// Returns the first verification failure, or nil if all pass.
func (m *MultiSigner) VerifyAll(evt event.Event, verifiers map[string]Signer) error {
	if evt == nil {
		return ErrNilEvent
	}

	ms, err := ExtractMultiSignature(evt)
	if err != nil {
		return err
	}

	for _, entry := range ms.Entries {
		verifier, ok := verifiers[entry.Actor]
		if !ok {
			return fmt.Errorf("no verifier provided for actor %s", entry.Actor)
		}

		if err := verifier.Verify(evt, entry.Sig); err != nil {
			return fmt.Errorf("verify actor %s (%s): %w", entry.Actor, entry.Algorithm, err)
		}
	}

	return nil
}

// ExtractMultiSignature retrieves the multi-signature collection from an event.
func ExtractMultiSignature(evt event.Event) (MultiSignature, error) {
	if evt == nil {
		return MultiSignature{}, ErrNilEvent
	}

	md := evt.Metadata()
	if md == nil || md.Custom == nil {
		return MultiSignature{}, ErrNilSignature
	}

	encoded, ok := md.Custom[MultiSigMetadataKey]
	if !ok || encoded == "" {
		return MultiSignature{}, ErrNilSignature
	}

	var ms MultiSignature
	if err := json.Unmarshal([]byte(encoded), &ms); err != nil {
		return MultiSignature{}, fmt.Errorf("%w: decode multi-sig: %w", ErrInvalidSignature, err)
	}

	return ms, nil
}

// HasMultiSignature reports whether the event carries a multi-signature collection.
func HasMultiSignature(evt event.Event) bool {
	_, err := ExtractMultiSignature(evt)

	return err == nil
}

func attachMultiSignature(evt event.Event, ms MultiSignature) (*event.ImmutableEvent, error) {
	encoded, err := json.Marshal(ms)
	if err != nil {
		return nil, fmt.Errorf("encode multi-sig: %w", err)
	}

	clone, err := event.NewEvent(
		evt.Type(),
		evt.AggregateID(),
		evt.AggregateType(),
		evt.Version(),
		evt.Payload(),
		event.WithEventID(evt.ID()),
		event.WithOccurredAt(evt.OccurredAt()),
		event.WithSchemaVersion(evt.SchemaVersion()),
		event.WithMetadata(evt.Metadata()),
		event.WithCustom(MultiSigMetadataKey, string(encoded)),
	)
	if err != nil {
		return nil, fmt.Errorf("reconstruct event with multi-sig: %w", err)
	}

	return clone, nil
}

func removeActor(entries []SignatureEntry, actor string) []SignatureEntry {
	result := make([]SignatureEntry, 0, len(entries))

	for _, e := range entries {
		if e.Actor != actor {
			result = append(result, e)
		}
	}

	return result
}
