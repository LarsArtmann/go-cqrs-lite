package event

import (
	"encoding/json"
	"strconv"
)

// Codec serializes and deserializes event payloads.
type Codec interface {
	// Encode marshals a value to bytes.
	Encode(v any) ([]byte, error)
	// Decode unmarshals bytes into a value.
	Decode(data []byte, v any) error
}

// JSONCodec implements Codec using encoding/json.
type JSONCodec struct{}

var _ Codec = JSONCodec{}

// Encode marshals a value to JSON bytes.
func (JSONCodec) Encode(v any) ([]byte, error) {
	//nolint:wrapcheck // thin wrapper over json.Marshal
	return json.Marshal(v)
}

// Decode unmarshals JSON bytes into a value.
func (JSONCodec) Decode(data []byte, v any) error {
	//nolint:wrapcheck // thin wrapper over json.Unmarshal
	return json.Unmarshal(data, v)
}

// DecodePayload decodes an event's payload bytes into a typed value using
// the provided codec. This is the standard way to deserialize event data
// in event handlers and projectors.
func DecodePayload[T any](evt Event, codec Codec) (T, error) {
	var zero T

	payload := evt.Payload()
	if len(payload) == 0 {
		return zero, nil
	}

	var target T

	err := codec.Decode(payload, &target)
	if err != nil {
		return zero, WrapCorruption(
			err,
			"event.decode_payload_failed",
			"decode payload for event "+string(evt.Type()),
		)
	}

	return target, nil
}

// DecodePayloads decodes multiple events' payloads into a slice of typed values.
// Returns an error at the first decode failure, indicating the index.
func DecodePayloads[T any](events []Event, codec Codec) ([]T, error) {
	result := make([]T, 0, len(events))

	for i, evt := range events {
		v, err := DecodePayload[T](evt, codec)
		if err != nil {
			return nil, WrapCorruption(
				err,
				"event.decode_payload_failed",
				"decode payload ["+strconv.Itoa(i)+"] for event "+string(evt.Type()),
			)
		}

		result = append(result, v)
	}

	return result, nil
}
