package event

import (
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec"
	"github.com/larsartmann/go-cqrs-lite/id"
)

// ReconstructEventFromFields rebuilds an Event from persisted field values.
// This is the shared reconstruction logic for all event stores (SQL, Pebble, etc.).
// The errCodePrefix parameter is used for error attribution (e.g., "storage", "pebble").
func ReconstructEventFromFields(
	eventID id.EventID,
	eventType, aggType string,
	aggID id.AggregateID,
	version, schemaVersion int,
	payload, metadataJSON []byte,
	occurredAt time.Time,
	encoding codec.Encoding,
	errCodePrefix string,
) (Event, error) {
	metaOpts, err := UnmarshalMetadataJSON(
		metadataJSON,
		errCodePrefix+".unmarshal_metadata",
		eventType,
	)
	if err != nil {
		return nil, WrapCorruption(
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
		Type(eventType),
		aggID,
		AggregateType(aggType),
		Version(version),
		payload,
		opts...,
	)
	if err != nil {
		return nil, WrapCorruption(err, errCodePrefix+".reconstruct_event",
			"reconstruct event "+eventType)
	}

	return evt, nil
}
