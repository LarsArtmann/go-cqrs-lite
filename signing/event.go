package signing

import (
	"encoding/base64"

	"github.com/larsartmann/go-cqrs-lite/event"
)

// MetadataKey is the custom metadata key used to store event signatures.
const MetadataKey event.MetadataKey = "event.signature"

// CloneEvent reconstructs an event preserving all fields and adding a custom
// metadata key-value pair. Returns a new ImmutableEvent; the original is unmodified.
func CloneEvent(
	evt event.Event,
	key event.MetadataKey,
	value string,
) (*event.ImmutableEvent, error) {
	//nolint:wrapcheck // callers wrap with context
	return event.NewEvent(
		evt.Type(),
		evt.AggregateID(),
		evt.AggregateType(),
		evt.Version(),
		evt.Payload(),
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
func AttachSignature(evt event.Event, sig Signature) (*event.ImmutableEvent, error) {
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

	md := evt.Metadata()
	if md.Custom == nil {
		return nil, ErrNilSignature
	}

	encoded, ok := md.Custom[MetadataKey]
	if !ok || encoded == "" {
		return nil, ErrNilSignature
	}

	decoded, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, event.WrapInfrastructure(
			err,
			"signing.decode_signature",
			"decode signature from base64",
		)
	}

	return Signature(decoded), nil
}

// HasSignature reports whether the event carries a signature in its metadata.
func HasSignature(evt event.Event) bool {
	_, err := ExtractSignature(evt)

	return err == nil
}
