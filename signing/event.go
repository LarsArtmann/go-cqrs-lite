package signing

import (
	"encoding/base64"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// MetadataKey is the custom metadata key used to store event signatures.
const MetadataKey event.MetadataKey = "event.signature"

// CloneEvent reconstructs an event preserving all fields and adding a custom
// metadata key-value pair. Returns a new ImmutableEvent; the original is unmodified.
func CloneEvent(
	evt event.Event,
	key event.MetadataKey,
	value string,
) (event.Event, error) {
	//nolint:wrapcheck // callers wrap with context
	return event.NewEvent(
		evt.Type(),
		evt.AggregateID(),
		evt.AggregateType(),
		evt.Version(),
		event.PayloadReadOnly(evt),
		event.WithEventID(evt.ID()),
		event.WithOccurredAt(evt.OccurredAt()),
		event.WithSchemaVersion(evt.SchemaVersion()),
		event.WithMetadata(evt.Metadata()),
		event.WithCustom(key, value),
	)
}

// AttachSignature stores a base64-encoded signature in the event's custom metadata.
// Returns a new event with the signature attached. The original event is unmodified.
//
// Use this to attach signatures to events before storage or transmission.
// The signature can later be extracted with ExtractSignature for verification.
func AttachSignature(evt event.Event, sig Signature) (event.Event, error) {
	if evt == nil {
		return nil, ErrNilEvent
	}

	encoded := base64.URLEncoding.EncodeToString(sig.Bytes())

	clone, err := CloneEvent(evt, MetadataKey, encoded)
	if err != nil {
		return nil, event.WrapInfrastructure(
			err,
			"signing.attach_signature",
			"reconstruct event with signature",
		)
	}

	return clone, nil
}

// ExtractSignature retrieves the signature from an event's custom metadata.
// Returns ErrNilSignature if no signature is present.
func ExtractSignature(evt event.Event) (Signature, error) {
	if evt == nil {
		return nil, ErrNilEvent
	}

	decoded, found, err := event.ExtractCustomBytes(evt, MetadataKey)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "signing.extract", "extract signature from event")
	}

	if !found {
		return nil, ErrNilSignature
	}

	return Signature(decoded), nil
}

// HasSignature reports whether the event carries a valid signature in its metadata.
// Returns true only if ExtractSignature succeeds. Returns false for absent/nil signatures
// (rejection errors). Infrastructure errors (corrupt base64) indicate a signature IS
// present but malformed — the caller should attempt ExtractSignature for proper error handling.
func HasSignature(evt event.Event) bool {
	_, err := ExtractSignature(evt)
	if err == nil {
		return true
	}

	return event.Classify(err) != event.Rejection
}
