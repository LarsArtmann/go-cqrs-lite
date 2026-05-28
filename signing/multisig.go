package signing

import (
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// NewMultiSigner creates a signer for a named actor using the provided Signer.
// By default, the same Signer is used for both signing and verification.
// Pass WithVerifier to override the verifier (needed for Ed25519).
func NewMultiSigner(
	actor Actor,
	algorithm SignatureAlgorithm,
	signer Signer,
	opts ...MultiSignerOption,
) (*MultiSigner, error) {
	if actor == "" {
		return nil, event.NewRejection("signing.empty_actor", "actor name cannot be empty")
	}

	if signer == nil {
		return nil, event.NewRejection("signing.nil_signer", "signer cannot be nil")
	}

	if algorithm == "" {
		return nil, event.NewRejection("signing.empty_algorithm", "algorithm cannot be empty")
	}

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

	if multi.verifier == nil {
		return nil, event.NewRejection(
			"signing.nil_verifier",
			"verifier cannot be nil; pass WithVerifier for Ed25519 signers",
		)
	}

	if multi.clock == nil {
		return nil, event.NewRejection("signing.nil_clock", "clock option cannot be nil")
	}

	return multi, nil
}

// Actor returns the actor identifier.
func (m *MultiSigner) Actor() Actor { return m.actor }

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

	entry := SignatureEntry{
		Actor:     m.actor,
		Algorithm: m.algorithm,
		Sig:       sig,
		SignedAt:  m.clock(),
	}

	validateErr := entry.Validate()
	if validateErr != nil {
		return nil, fmt.Errorf("validate signature entry: %w", validateErr)
	}

	multiSig.Entries = removeActor(multiSig.Entries, m.actor)
	multiSig.Entries = append(multiSig.Entries, entry)

	return attachMultiSignature(evt, multiSig)
}

// Verify verifies only this actor's signature from the event's multi-sig collection.
func (m *MultiSigner) Verify(evt event.Event) error {
	err := verifyActorEntry(evt, m.actor, m.verifier)
	if err != nil {
		return fmt.Errorf("verify actor %s: %w", m.actor, err)
	}

	return nil
}

// VerifyActor verifies a specific actor's signature using the provided verifier.
// Useful when one actor wants to check another actor's signature.
func (m *MultiSigner) VerifyActor(evt event.Event, actor Actor, verifier Verifier) error {
	err := verifyActorEntry(evt, actor, verifier)
	if err != nil {
		return fmt.Errorf("verify actor %s: %w", actor, err)
	}

	return nil
}

// verifyActorEntry extracts the multi-sig, finds the actor's entry, and verifies it.
func verifyActorEntry(evt event.Event, actor Actor, verifier Verifier) error {
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
		return fmt.Errorf("verify entry for actor %s: %w", actor, verifyErr)
	}

	return nil
}
