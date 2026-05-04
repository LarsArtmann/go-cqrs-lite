package event

import (
	"encoding/json"
	"fmt"
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
		return zero, fmt.Errorf("decode payload for event %s: %w", evt.Type(), err)
	}

	return target, nil
}
