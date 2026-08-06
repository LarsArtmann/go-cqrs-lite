package bbolt

import (
	"encoding/json/v2"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func marshalCBOR(v any) ([]byte, error) {
	return codec.CBOREncMode().Marshal(v)
}

func unmarshalCBOR(data []byte, v any) error {
	return codec.CBORDecMode().Unmarshal(data, v)
}

func isCBOR(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	return data[0] >= 0xa0 && data[0] <= 0xbf
}

func unmarshalCBOROrJSON(data []byte, target any, code, msg string) error {
	var err error

	if isCBOR(data) {
		err = unmarshalCBOR(data, target)
	} else {
		err = json.Unmarshal(data, target)
	}

	if err != nil {
		return errorfamily.WrapCorruption(err, code, msg)
	}

	return nil
}

func serializeEvent(evt event.Event) ([]byte, error) {
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

	data, err := marshalCBOR(s)
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, "bbolt.serialize_event", "marshal event")
	}

	return data, nil
}

func deserializeEvent(data []byte) (event.Event, error) {
	var s serializableEvent
	if err := unmarshalCBOROrJSON(data, &s, "bbolt.unmarshal_event",
		"failed to unmarshal event"); err != nil {
		return nil, err
	}

	metadataJSON, err := event.MarshalMetadataJSON(s.Metadata, "bbolt.marshal_metadata")
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, "bbolt.marshal_metadata",
			"failed to marshal metadata for deserialization")
	}

	evt, err := event.ReconstructEventFromFields(
		s.ID, event.Type(s.Type), id.StreamType(s.StreamType), s.StreamID,
		s.Version, s.SchemaVersion,
		s.Payload, metadataJSON,
		time.Unix(0, s.OccurredAt).UTC(),
		codec.Encoding(s.Encoding), "bbolt",
	)
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, "bbolt.reconstruct_event",
			"failed to reconstruct event from fields")
	}

	return evt, nil
}

type serializableEvent struct {
	ID            id.EventID      `json:"id"`
	Type          string          `json:"type"`
	StreamID      id.StreamID     `json:"aggregate_id"`
	StreamType    string          `json:"aggregate_type"`
	Version       int             `json:"version"`
	SchemaVersion int             `json:"schema_version,omitempty"`
	Payload       []byte          `json:"payload"`
	OccurredAt    int64           `json:"occurred_at"`
	Metadata      event.Metadata  `json:"metadata"`
	Encoding      string          `json:"encoding,omitempty"`
}
