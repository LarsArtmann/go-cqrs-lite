package event

import (
	"fmt"
	"time"

	"github.com/larsartmann/go-codec"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// ReconstructEventFromFields rebuilds an Event from persisted field values.
// This is the shared reconstruction logic for all event stores (SQL, Pebble, etc.).
// The errCodePrefix parameter is used for error attribution (e.g., "storage", "pebble").
func ReconstructEventFromFields(
	eventID id.EventID,
	eventType Type,
	aggType id.StreamType,
	aggID id.StreamID,
	version, schemaVersion int,
	payload, metadataJSON []byte,
	occurredAt time.Time,
	encoding codec.Encoding,
	errCodePrefix string,
) (Event, error) {
	metaOpts, err := UnmarshalMetadataJSON(
		metadataJSON,
		errCodePrefix+".unmarshal_metadata",
		string(eventType),
	)
	if err != nil {
		return nil, errorfamily.WrapCorruption(
			err,
			errCodePrefix+".metadata_unmarshal",
			fmt.Sprintf(
				"metadata for %s/%s v%d (schema v%d)",
				aggType,
				eventType,
				version,
				schemaVersion,
			),
		)
	}

	return reconstructEvent(eventID, eventType, aggType, aggID, version, schemaVersion,
		payload, metaOpts, occurredAt, encoding, errCodePrefix)
}

// ReconstructEventWithMetadata rebuilds an Event from an already-decoded
// Metadata value, skipping the marshal-to-JSON/unmarshal round-trip that
// [ReconstructEventFromFields] performs. Engines that decode the whole event
// envelope (including metadata) in one step (CBOR envelopes in Pebble, bbolt)
// should use this variant: it avoids two serializations and an intermediate
// buffer per event read.
//
// The metadata value is merged into the event as-is; the caller must not
// retain or mutate it afterwards (treat it as transferred).
func ReconstructEventWithMetadata(
	eventID id.EventID,
	eventType Type,
	aggType id.StreamType,
	aggID id.StreamID,
	version, schemaVersion int,
	payload []byte,
	metadata Metadata,
	occurredAt time.Time,
	encoding codec.Encoding,
	errCodePrefix string,
) (Event, error) {
	return reconstructEvent(eventID, eventType, aggType, aggID, version, schemaVersion,
		payload, []Option{WithMetadata(metadata)}, occurredAt, encoding, errCodePrefix)
}

func reconstructEvent(
	eventID id.EventID,
	eventType Type,
	aggType id.StreamType,
	aggID id.StreamID,
	version, schemaVersion int,
	payload []byte,
	metaOpts []Option,
	occurredAt time.Time,
	encoding codec.Encoding,
	errCodePrefix string,
) (Event, error) {
	opts := make([]Option, 0, 3+len(metaOpts))

	opts = append(opts, WithEventID(eventID), WithOccurredAt(occurredAt))
	if schemaVersion > 0 {
		opts = append(opts, WithSchemaVersion(SchemaVersion(schemaVersion)))
	}

	opts = append(opts, metaOpts...)

	if encoding != "" {
		opts = append(opts, WithEncoding(encoding))
	}

	evt, err := NewEvent(
		eventType,
		aggID,
		aggType,
		Version(version),
		payload,
		opts...,
	)
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, errCodePrefix+".reconstruct_event",
			"reconstruct event "+string(eventType))
	}

	return evt, nil
}
