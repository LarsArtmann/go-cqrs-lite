package multisig

import (
	"encoding/json"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/signing/v3"
)

// VerifyAll verifies every signature entry in the multi-sig collection
// using each entry's algorithm-appropriate verifier from the provided map.
// The map keys are actor names; the values are their respective verifiers.
// Returns the first verification failure, or nil if all pass.
func VerifyAll(evt event.Event, verifiers map[Actor]signing.Verifier) error {
	if evt == nil {
		return signing.ErrNilEvent
	}

	multiSig, err := ExtractMultiSignature(evt)
	if err != nil {
		return err
	}

	for _, entry := range multiSig.Entries {
		verifier, ok := verifiers[entry.Actor]
		if !ok {
			return errorfamily.Newf(
				errorfamily.Rejection,
				"signing.no_verifier",
				"%s",
				entry.Actor,
			)
		}

		verifyErr := verifier.Verify(evt, entry.Sig)
		if verifyErr != nil {
			return errorfamily.WrapInfrastructure(
				verifyErr,
				"signing.verify_all",
				"verify actor "+string(entry.Actor)+" ("+string(entry.Algorithm)+")",
			)
		}
	}

	return nil
}

// ExtractMultiSignature retrieves the multi-signature collection from an event.
func ExtractMultiSignature(evt event.Event) (MultiSignature, error) {
	if evt == nil {
		return MultiSignature{Entries: nil}, signing.ErrNilEvent
	}

	md := evt.Metadata()
	if md.Custom == nil {
		return MultiSignature{Entries: nil}, signing.ErrNilSignature
	}

	encoded, ok := md.Custom[MultiSigMetadataKey]
	if !ok || encoded == "" {
		return MultiSignature{Entries: nil}, signing.ErrNilSignature
	}

	var multiSig MultiSignature

	unmarshalErr := json.Unmarshal([]byte(encoded), &multiSig)
	if unmarshalErr != nil {
		return MultiSignature{Entries: nil}, errorfamily.WrapInfrastructure(
			unmarshalErr,
			"signing.decode_multi_sig",
			"decode multi-signature",
		)
	}

	return multiSig, nil
}

// VerifierMap builds an Actor→Verifier map from one or more MultiSigners.
// This is a convenience for the common case where you already have MultiSigners
// for signing and need the same actor-to-verifier mapping for VerifyAll
// or RequireMultiSigMiddleware.
//
// Returns ErrNilSigner if any signer is nil.
func VerifierMap(signers ...*MultiSigner) (map[Actor]signing.Verifier, error) {
	verifiers := make(map[Actor]signing.Verifier, len(signers))

	for _, s := range signers {
		if s == nil {
			return nil, ErrNilSigner
		}

		verifiers[s.Actor()] = s.verifier
	}

	return verifiers, nil
}

// HasMultiSignature reports whether the event carries a multi-signature collection.
func HasMultiSignature(evt event.Event) bool {
	_, err := ExtractMultiSignature(evt)

	return err == nil
}

func attachMultiSignature(
	evt event.Event,
	multiSig MultiSignature,
) (event.Event, error) {
	encoded, err := json.Marshal(multiSig)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"signing.encode_multi_sig",
			"encode multi-signature",
		)
	}

	clone, err := signing.CloneEvent(evt, MultiSigMetadataKey, string(encoded))
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"signing.reconstruct_multi_sig",
			"reconstruct event with multi-sig",
		)
	}

	return clone, nil
}

func removeActor(entries []SignatureEntry, actor Actor) []SignatureEntry {
	result := make([]SignatureEntry, 0, len(entries))

	for _, entry := range entries {
		if entry.Actor != actor {
			result = append(result, entry)
		}
	}

	return result
}
