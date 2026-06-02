package multisig

import (
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/signing/v2"
)

// NewMultiSigner creates a signer for a named actor using the provided Signer.
// By default, the same Signer is used for both signing and verification.
// Pass WithVerifier to override the verifier (needed for Ed25519).
func NewMultiSigner(
	actor Actor,
	algorithm SignatureAlgorithm,
	signer signing.Signer,
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

	var verifier signing.Verifier
	if sv, ok := signer.(signing.Verifier); ok {
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
		return nil, signing.ErrNilEvent
	}

	sig, err := m.signer.Sign(evt)
	if err != nil {
		return nil, event.WrapInfrastructure(
			err,
			"signing.multi_sign",
			"sign event for actor "+string(m.actor),
		)
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
		return nil, event.WrapRejection(
			validateErr,
			"signing.invalid_entry",
			"validate signature entry",
		)
	}

	multiSig.Entries = removeActor(multiSig.Entries, m.actor)
	multiSig.Entries = append(multiSig.Entries, entry)

	return attachMultiSignature(evt, multiSig)
}

// Verify verifies only this actor's signature from the event's multi-sig collection.
func (m *MultiSigner) Verify(evt event.Event) error {
	err := verifyActorEntry(evt, m.actor, m.verifier)
	if err != nil {
		return event.WrapInfrastructure(
			err,
			"signing.verify_actor",
			"verify actor "+string(m.actor),
		)
	}

	return nil
}

// VerifyActor verifies a specific actor's signature using the provided verifier.
// Useful when one actor wants to check another actor's signature.
func (m *MultiSigner) VerifyActor(evt event.Event, actor Actor, verifier signing.Verifier) error {
	err := verifyActorEntry(evt, actor, verifier)
	if err != nil {
		return event.WrapInfrastructure(
			err,
			"signing.verify_actor",
			"verify actor "+string(actor),
		)
	}

	return nil
}

// verifyActorEntry extracts the multi-sig, finds the actor's entry, and verifies it.
func verifyActorEntry(evt event.Event, actor Actor, verifier signing.Verifier) error {
	if evt == nil {
		return signing.ErrNilEvent
	}

	multiSig, err := ExtractMultiSignature(evt)
	if err != nil {
		return err
	}

	entry := multiSig.Get(actor)
	if entry == nil {
		return event.Wrapf(
			signing.ErrNilSignature, event.Rejection,
			"signing.no_actor_signature",
			"no signature found for actor %s",
			actor,
		)
	}

	verifyErr := verifier.Verify(evt, entry.Sig)
	if verifyErr != nil {
		return event.WrapInfrastructure(
			verifyErr,
			"signing.verify_actor_entry",
			"verify entry for actor "+string(actor),
		)
	}

	return nil
}
