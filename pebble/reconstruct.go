package pebble

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
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

	opts := make([]event.Option, 0, 3+len(metaOpts))

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
	if len(data) == 0 {
		return nil, nil
	}

	var meta event.Metadata

	err := json.Unmarshal(data, &meta)
	if err != nil {
		return nil, event.WrapCorruption(err, "pebble.unmarshal_metadata",
			"unmarshal metadata for event "+eventType)
	}

	return []event.Option{event.WithMetadata(meta)}, nil
}

func marshalMetadata(m event.Metadata) ([]byte, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, event.WrapCorruption(err, "pebble.marshal_metadata",
			"marshal metadata")
	}

	return data, nil
}
