package pebble

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// marshalCBOR serializes a value using the canonical CBOR encoding mode
// (RFC 7049 sorted map keys). Delegates to codec.CBOREncMode so all modules
// share one deterministic encoding mode.
func marshalCBOR(v any) ([]byte, error) {
	mode, err := codec.CBOREncMode()
	if err != nil {
		return nil, fmt.Errorf("pebble: CBOR encoding mode: %w", err)
	}

	return mode.Marshal(v)
}

// unmarshalCBOR deserializes CBOR data using the canonical decoding mode
// with duplicate map key enforcement. Delegates to codec.CBORDecMode.
func unmarshalCBOR(data []byte, v any) error {
	mode, err := codec.CBORDecMode()
	if err != nil {
		return fmt.Errorf("pebble: CBOR decoding mode: %w", err)
	}

	return mode.Unmarshal(data, v)
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
		AggregateID:   evt.AggregateID(),
		AggregateType: string(evt.AggregateType()),
		Version:       evt.Version().Int(),
		SchemaVersion: evt.SchemaVersion().Int(),
		Payload:       event.PayloadReadOnly(evt),
		OccurredAt:    evt.OccurredAt().UnixNano(),
		Metadata:      evt.Metadata(),
		Encoding:      string(evt.Encoding()),
	}

	data, err := marshalCBOR(s)
	if err != nil {
		return nil, fmt.Errorf("pebble: marshal event: %w", err)
	}

	return data, nil
}

// deserializeEvent converts CBOR (or legacy JSON) to a CQRS-compatible event.
// Supports both CBOR-encoded envelopes (current) and JSON-encoded envelopes
// (legacy) for backward compatibility with data written before the migration.
func (a *EventStore) deserializeEvent(data []byte) (event.Event, error) {
	var s serializableEvent

	var err error

	if isCBOR(data) {
		err = unmarshalCBOR(data, &s)
	} else {
		err = json.Unmarshal(
			data,
			&s,
		) //nolint:nolintlint // legacy JSON fallback for backward compat
	}

	if err != nil {
		return nil, event.WrapCorruption(err, "pebble.unmarshal_event",
			"failed to unmarshal event")
	}

	metadataJSON, err := event.MarshalMetadataJSON(s.Metadata, "pebble.marshal_metadata")
	if err != nil {
		return nil, event.WrapCorruption(err, "pebble.marshal_metadata",
			"failed to marshal metadata for deserialization")
	}

	evt, err := event.ReconstructEventFromFields(
		s.ID, event.Type(s.Type), event.AggregateType(s.AggregateType), s.AggregateID,
		s.Version, s.SchemaVersion,
		s.Payload, metadataJSON,
		time.Unix(0, s.OccurredAt),
		codec.Encoding(s.Encoding),
		"pebble",
	)
	if err != nil {
		return nil, event.WrapCorruption(err, "pebble.reconstruct_event",
			"failed to reconstruct event from fields")
	}

	return evt, nil
}

// serializableEvent represents the CBOR (and legacy JSON) storage format for events.
// fxamacker/cbor reads `json` struct tags by default, so no separate `cbor` tags needed.
type serializableEvent struct {
	ID            id.EventID     `json:"id"`
	Type          string         `json:"type"`
	AggregateID   id.AggregateID `json:"aggregate_id"`
	AggregateType string         `json:"aggregate_type"`
	Version       int            `json:"version"`
	SchemaVersion int            `json:"schema_version,omitempty"`
	Payload       []byte         `json:"payload"`
	OccurredAt    int64          `json:"occurred_at"`
	Metadata      event.Metadata `json:"metadata"`
	Encoding      string         `json:"encoding,omitempty"`
}
