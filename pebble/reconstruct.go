package pebble

import (
	"github.com/larsartmann/go-cqrs-lite/event"
)

func unmarshalEventMetadata(data []byte, eventType string) ([]event.Option, error) {
	return event.UnmarshalMetadataJSON(data, "pebble.unmarshal_metadata", eventType)
}

func marshalMetadata(m event.Metadata) ([]byte, error) {
	return event.MarshalMetadataJSON(m, "pebble.marshal_metadata")
}
