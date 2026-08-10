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
	//art-dupl:accept intentional cross-module duplicate — separate go.mod
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
	metadataJSON, err := event.MarshalMetadataJSON(
		evt.Metadata(), "bbolt.serialize_metadata",
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

	evt, err := event.ReconstructEventFromFields(
		s.ID, event.Type(s.Type), id.StreamType(s.StreamType), s.StreamID,
		s.Version, s.SchemaVersion,
		s.Payload, []byte(s.Metadata),
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
	Metadata      metadataPayload `json:"metadata"`
	Encoding      string          `json:"encoding,omitempty"`
}

// metadataPayload stores event.Metadata as JSON bytes within the CBOR envelope.
// This ensures types implementing json.Marshaler (e.g. id.ActorID, which has
// unexported fields) serialize correctly, since fxamacker/cbor does not invoke
// json.Marshaler. On decode, legacy CBOR data (where metadata was a CBOR map) is
// handled by falling back to struct reflection and re-marshaling to JSON.
// art-dupl:accept intentional cross-module duplicate — separate go.mod
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
