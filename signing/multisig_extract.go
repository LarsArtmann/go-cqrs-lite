package signing

import (
	"encoding/json"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// VerifyAll verifies every signature entry in the multi-sig collection
// using each entry's algorithm-appropriate verifier from the provided map.
// The map keys are actor names; the values are their respective verifiers.
// Returns the first verification failure, or nil if all pass.
func VerifyAll(evt event.Event, verifiers map[Actor]Verifier) error {
	if evt == nil {
		return ErrNilEvent
	}

	multiSig, err := ExtractMultiSignature(evt)
	if err != nil {
		return err
	}

	for _, entry := range multiSig.Entries {
		verifier, ok := verifiers[entry.Actor]
		if !ok {
			return fmt.Errorf("%w: %s", ErrNoVerifier, entry.Actor)
		}

		verifyErr := verifier.Verify(evt, entry.Sig)
		if verifyErr != nil {
			return fmt.Errorf("verify actor %s (%s): %w", entry.Actor, entry.Algorithm, verifyErr)
		}
	}

	return nil
}

// ExtractMultiSignature retrieves the multi-signature collection from an event.
func ExtractMultiSignature(evt event.Event) (MultiSignature, error) {
	if evt == nil {
		return MultiSignature{Entries: nil}, ErrNilEvent
	}

	md := evt.Metadata()
	if md == nil || md.Custom == nil {
		return MultiSignature{Entries: nil}, ErrNilSignature
	}

	encoded, ok := md.Custom[MultiSigMetadataKey]
	if !ok || encoded == "" {
		return MultiSignature{Entries: nil}, ErrNilSignature
	}

	var multiSig MultiSignature

	unmarshalErr := json.Unmarshal([]byte(encoded), &multiSig)
	if unmarshalErr != nil {
		return MultiSignature{Entries: nil}, fmt.Errorf(
			"%w: decode multi-sig: %w",
			ErrInvalidSignature,
			unmarshalErr,
		)
	}

	return multiSig, nil
}

// HasMultiSignature reports whether the event carries a multi-signature collection.
func HasMultiSignature(evt event.Event) bool {
	_, err := ExtractMultiSignature(evt)

	return err == nil
}

func attachMultiSignature(
	evt event.Event,
	multiSig MultiSignature,
) (*event.ImmutableEvent, error) {
	encoded, err := json.Marshal(multiSig)
	if err != nil {
		return nil, fmt.Errorf("encode multi-sig: %w", err)
	}

	clone, err := cloneEvent(evt, MultiSigMetadataKey, string(encoded))
	if err != nil {
		return nil, fmt.Errorf("reconstruct event with multi-sig: %w", err)
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
