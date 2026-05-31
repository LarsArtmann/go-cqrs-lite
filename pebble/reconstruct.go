package pebble

import (
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec"
	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/id"
)

func reconstructEvent(
	eventID id.EventID,
	eventType, aggType string,
	aggID id.AggregateID,
	version, schemaVersion int,
	payload, metadataJSON []byte,
	occurredAt time.Time,
	encoding codec.Encoding,
) (event.Event, error) {
	metaOpts, err := unmarshalEventMetadata(metadataJSON, eventType)
	if err != nil {
		return nil, event.WrapCorruption(
			err,
			"pebble.metadata_unmarshal",
			fmt.Sprintf(
				"metadata for %s/%s v%d (schema v%d)",
				aggType,
				eventType,
				version,
				schemaVersion,
			),
		)
	}

	const reconstructOptionCapacity = 3 // event ID + occurred-at + schema-version

	opts := make([]event.Option, 0, reconstructOptionCapacity+len(metaOpts))

	opts = append(opts, event.WithEventID(eventID), event.WithOccurredAt(occurredAt))
	if schemaVersion > 0 {
		opts = append(opts, event.WithSchemaVersion(event.SchemaVersion(schemaVersion)))
	}

	opts = append(opts, metaOpts...)

	if encoding != "" {
		opts = append(opts, event.WithEncoding(encoding))
	}

	evt, err := event.NewEvent(
		event.Type(eventType),
		aggID,
		event.AggregateType(aggType),
		event.Version(version),
		payload,
		opts...,
	)
	if err != nil {
		return nil, event.WrapCorruption(err, "pebble.reconstruct_event",
			"reconstruct event "+eventType)
	}

	return evt, nil
}

func unmarshalEventMetadata(data []byte, eventType string) ([]event.Option, error) {
	return event.UnmarshalMetadataJSON(data, "pebble.unmarshal_metadata", eventType)
}

func marshalMetadata(m event.Metadata) ([]byte, error) {
	return event.MarshalMetadataJSON(m, "pebble.marshal_metadata")
}
