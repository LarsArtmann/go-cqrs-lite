package pebble

import (
	"encoding/json/v2"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// marshalCBOR serializes a value using the canonical CBOR encoding mode
// (RFC 7049 sorted map keys). Delegates to codec.CBOREncMode so all modules
// share one deterministic encoding mode.
func marshalCBOR(v any) ([]byte, error) {
	return codec.CBOREncMode().Marshal(v)
}

// marshalCBOROrErr marshals v to CBOR and wraps any failure as Corruption
// with the given code/msg. Caller passes a stable code (e.g.
// "pebble.serialize_event") and a short human-readable msg.
func marshalCBOROrErr(v any, code, msg string) ([]byte, error) {
	data, err := marshalCBOR(v)
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, code, msg)
	}
	return data, nil
}

// unmarshalCBOR deserializes CBOR data using the canonical decoding mode
// with duplicate map key enforcement. Delegates to codec.CBORDecMode.
func unmarshalCBOR(data []byte, v any) error {
	return codec.CBORDecMode().Unmarshal(data, v)
}

// unmarshalCBOROrJSON detects CBOR vs legacy JSON and decodes into target,
// wrapping any failure as Corruption with the given code/msg. The JSON branch
// is kept for backward compatibility with envelopes written before the CBOR
// migration.
func unmarshalCBOROrJSON(data []byte, target any, code, msg string) error {
	var err error
	if isCBOR(data) {
		err = unmarshalCBOR(data, target)
	} else {
		err = json.Unmarshal(
			data,
			target,
		) //nolint:nolintlint // legacy JSON fallback for backward compat
	}
	if err != nil {
		return errorfamily.WrapCorruption(err, code, msg)
	}
	return nil
}

// isCBOR detects CBOR-encoded data by checking for CBOR major type 5 (map).
// CBOR maps start with byte 0xa0–0xbf (major type 5, 0–23 additional info).
// JSON objects start with '{' (0x7b). This sniff is used for backward
// compatibility when reading envelopes that may be JSON or CBOR.
func isCBOR(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	return data[0] >= 0xa0 && data[0] <= 0xbf
}

// serializeEvent converts a CQRS event to CBOR.
// Uses marshalCBOR directly: fxamacker/cbor already pools encode
// buffers internally, so this is the lowest-allocation path for producing an
// owned []byte that Pebble can store.
func (a *EventStore) serializeEvent(evt event.Event) ([]byte, error) {
	s := serializableEvent{
		ID:            evt.ID(),
		Type:          string(evt.Type()),
		StreamID:      evt.StreamID(),
		StreamType:    string(evt.StreamType()),
		Version:       evt.Version().Int(),
		SchemaVersion: evt.SchemaVersion().Int(),
		Payload:       event.PayloadReadOnly(evt),
		OccurredAt:    evt.OccurredAt().UnixNano(),
		Metadata:      evt.Metadata(),
		Encoding:      string(evt.Encoding()),
	}

	return marshalCBOROrErr(s, "pebble.serialize_event", "marshal event")
}

// deserializeEvent converts CBOR (or legacy JSON) to a CQRS-compatible event.
// Supports both CBOR-encoded envelopes (current) and JSON-encoded envelopes
// (legacy) for backward compatibility with data written before the migration.
func (a *EventStore) deserializeEvent(data []byte) (event.Event, error) {
	var s serializableEvent

	if err := unmarshalCBOROrJSON(data, &s, "pebble.unmarshal_event",
		"failed to unmarshal event"); err != nil {
		return nil, err
	}

	metadataJSON, err := event.MarshalMetadataJSON(s.Metadata, "pebble.marshal_metadata")
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, "pebble.marshal_metadata",
			"failed to marshal metadata for deserialization")
	}

	evt, err := event.ReconstructEventFromFields(
		s.ID, event.Type(s.Type), id.StreamType(s.StreamType), s.StreamID,
		s.Version, s.SchemaVersion,
		s.Payload, metadataJSON,
		time.Unix(0, s.OccurredAt).UTC(),
		codec.Encoding(s.Encoding),
		"pebble",
	)
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, "pebble.reconstruct_event",
			"failed to reconstruct event from fields")
	}

	return evt, nil
}

// serializableEvent represents the CBOR (and legacy JSON) storage format for events.
// fxamacker/cbor reads `json` struct tags by default, so no separate `cbor` tags needed.
// cqrs-lint:ignore(A011) library code or intentional pattern
type serializableEvent struct {
	ID            id.EventID     `json:"id"`
	Type          string         `json:"type"`
	StreamID      id.StreamID    `json:"aggregate_id"`
	StreamType    string         `json:"aggregate_type"`
	Version       int            `json:"version"`
	SchemaVersion int            `json:"schema_version,omitempty"`
	Payload       []byte         `json:"payload"`
	OccurredAt    int64          `json:"occurred_at"`
	Metadata      event.Metadata `json:"metadata"`
	Encoding      string         `json:"encoding,omitempty"`
}
