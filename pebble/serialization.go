package pebble

import (
	"encoding/json"
	"time"

	"github.com/fxamacker/cbor/v2"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// pebbleEncMode provides canonical CBOR encoding for event envelopes.
// Sorted map keys + shortest floats = deterministic output = signing-safe.
//
//nolint:gochecknoglobals // concurrency-safe EncMode, created once at package init
var pebbleEncMode = func() cbor.EncMode {
	em, err := cbor.CanonicalEncOptions().EncMode()
	if err != nil {
		panic("pebble: failed to create CBOR canonical encoding mode: " + err.Error())
	}

	return em
}()

// pebbleDecMode provides CBOR decoding with duplicate map key enforcement.
//
//nolint:gochecknoglobals // concurrency-safe DecMode, created once at package init
var pebbleDecMode = func() cbor.DecMode {
	//nolint:exhaustruct // only DupMapKey is intentional; all other fields use library defaults
	opts := cbor.DecOptions{DupMapKey: cbor.DupMapKeyEnforcedAPF}

	dm, err := opts.DecMode()
	if err != nil {
		panic("pebble: failed to create CBOR decoding mode: " + err.Error())
	}

	return dm
}()

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
// Falls back to JSON encoding if CBOR encoding fails.
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

	return pebbleEncMode.Marshal(s) //nolint:wrapcheck // storage serialization, not domain error
}

// deserializeEvent converts CBOR (or legacy JSON) to a CQRS-compatible event.
// Supports both CBOR-encoded envelopes (current) and JSON-encoded envelopes
// (legacy) for backward compatibility with data written before the migration.
func (a *EventStore) deserializeEvent(data []byte) (event.Event, error) {
	var s serializableEvent

	var err error

	if isCBOR(data) {
		err = pebbleDecMode.Unmarshal(data, &s)
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
	AggregateID   id.AggregateID `json:"aggregate_id"`   //nolint:tagliatelle // on-disk format uses snake_case
	AggregateType string         `json:"aggregate_type"` //nolint:tagliatelle
	Version       int            `json:"version"`
	SchemaVersion int            `json:"schema_version,omitempty"` //nolint:tagliatelle
	Payload       []byte         `json:"payload"`
	OccurredAt    int64          `json:"occurred_at"` //nolint:tagliatelle
	Metadata      event.Metadata `json:"metadata"`
	Encoding      string         `json:"encoding,omitempty"`
}
