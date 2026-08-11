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
	//art-dupl:accept intentional cross-module duplicate — separate go.mod
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
// Metadata is stored as JSON bytes inside the CBOR envelope so that types
// implementing json.Marshaler (e.g. id.ActorID) serialize correctly —
// fxamacker/cbor does not invoke json.Marshaler.
func (a *EventStore) serializeEvent(evt event.Event) ([]byte, error) {
	metadataJSON, err := event.MarshalMetadataJSON(
		evt.Metadata(), "pebble.serialize_metadata",
	)
	if err != nil {
		return nil, err
	}

	s := serializableEvent{
		ID:            evt.ID(),
		Type:          string(evt.Type()),
		StreamID:      evt.StreamID(),
		StreamType:    string(evt.StreamType()),
		Version:       evt.Version().Int(),
		SchemaVersion: evt.SchemaVersion().Int(),
		Payload:       event.PayloadReadOnly(evt),
		OccurredAt:    evt.OccurredAt().UnixNano(),
		Metadata:      metadataPayload(metadataJSON),
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

	evt, err := event.ReconstructEventFromFields(
		s.ID, event.Type(s.Type), id.StreamType(s.StreamType), s.StreamID,
		s.Version, s.SchemaVersion,
		s.Payload, []byte(s.Metadata),
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
	ID            id.EventID      `json:"id"`
	Type          string          `json:"type"`
	StreamID      id.StreamID     `json:"aggregate_id"`
	StreamType    string          `json:"aggregate_type"`
	Version       int             `json:"version"`
	SchemaVersion int             `json:"schema_version,omitempty"`
	Payload       []byte          `json:"payload"`
	OccurredAt    int64           `json:"occurred_at"`
	Metadata      metadataPayload `json:"metadata"`
	Encoding      string          `json:"encoding,omitempty"`
}

// metadataPayload stores event.Metadata as JSON bytes within the CBOR envelope.
// This ensures types implementing json.Marshaler (e.g. id.ActorID, which has
// unexported fields) serialize correctly, since fxamacker/cbor does not invoke
// json.Marshaler. On decode, legacy CBOR data (where metadata was a CBOR map) is
// handled by falling back to struct reflection and re-marshaling to JSON.
//
//nolint:recvcheck // Marshal=value receiver, Unmarshal=pointer (standard Go pattern)
type metadataPayload []byte

func (m metadataPayload) MarshalJSON() ([]byte, error) {
	if len(m) == 0 {
		return []byte("null"), nil
	}

	return m, nil
}

func (m *metadataPayload) UnmarshalJSON(data []byte) error { *m = data; return nil }

func (m metadataPayload) MarshalCBOR() ([]byte, error) {
	return marshalCBOR([]byte(m))
}

func (m *metadataPayload) UnmarshalCBOR(data []byte) error {
	var jsonBytes []byte
	if err := unmarshalCBOR(data, &jsonBytes); err == nil {
		*m = jsonBytes
		return nil
	}
	var meta event.Metadata
	if err := unmarshalCBOR(data, &meta); err != nil {
		return err
	}
	jsonBytes, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	*m = jsonBytes
	return nil
}
