package event

import (
	"context"
	"encoding/base64"
)

// AttachBinary stores a base64-encoded byte slice in the event's custom metadata
// under the given key. Returns a new ImmutableEvent; the original is unmodified.
//
// This is a shared building block for modules like signing and encryption that
// need to attach opaque binary blobs (signatures, ciphertexts) to events.
func AttachBinary(evt Event, key MetadataKey, data []byte) (*ImmutableEvent, error) {
	if evt == nil {
		return nil, NewRejection("event.nil_event", "event is nil")
	}

	encoded := base64.URLEncoding.EncodeToString(data)

	clone, err := NewEvent(
		evt.Type(),
		evt.AggregateID(),
		evt.AggregateType(),
		evt.Version(),
		PayloadReadOnly(evt),
		WithEventID(evt.ID()),
		WithOccurredAt(evt.OccurredAt()),
		WithSchemaVersion(evt.SchemaVersion()),
		WithMetadata(evt.Metadata()),
		WithCustom(key, encoded),
	)
	if err != nil {
		return nil, WrapInfrastructure(err, "event.attach_binary", "reconstruct event with binary metadata")
	}

	return clone, nil
}

// ExtractBinary retrieves a base64-encoded byte slice from the event's custom metadata.
// Returns ErrBinaryNotFound if the key is absent or empty.
func ExtractBinary(evt Event, key MetadataKey) ([]byte, error) {
	if evt == nil {
		return nil, NewRejection("event.nil_event", "event is nil")
	}

	md := evt.Metadata()
	if md.Custom == nil {
		return nil, ErrBinaryNotFound
	}

	encoded, ok := md.Custom[key]
	if !ok || encoded == "" {
		return nil, ErrBinaryNotFound
	}

	decoded, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, WrapInfrastructure(err, "event.decode_binary", "decode binary from base64")
	}

	return decoded, nil
}

// HasBinary reports whether the event carries binary data under the given metadata key.
// Returns true only if ExtractBinary succeeds. Returns false for absent data
// (rejection errors). Infrastructure errors (corrupt base64) indicate data IS
// present but malformed — the caller should attempt ExtractBinary for proper error handling.
func HasBinary(evt Event, key MetadataKey) bool {
	_, err := ExtractBinary(evt, key)

	return err == nil || Classify(err) != Rejection
}

// RejectingPublishMiddleware returns a PublishMiddleware that always rejects
// with a Rejection error using the given code and message.
// Use as a nil-guard fallback when a required dependency is not provided.
func RejectingPublishMiddleware(code, msg string) PublishMiddleware {
	return func(_ Publisher) Publisher {
		return PublisherFunc(func(_ context.Context, _ ...Event) error {
			return NewRejection(code, msg)
		})
	}
}

// RejectingHandlerMiddleware returns a Middleware that always rejects
// with a Rejection error using the given code and message.
// Use as a nil-guard fallback when a required dependency is not provided.
func RejectingHandlerMiddleware(code, msg string) Middleware {
	return func(_ Handler) Handler {
		return func(_ context.Context, _ Event) error {
			return NewRejection(code, msg)
		}
	}
}
