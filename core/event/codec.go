package event

import (
	jsonv2 "github.com/go-json-experiment/json"
)

// Codec serializes and deserializes event payloads.
type Codec interface {
	// Encode marshals a value to bytes.
	Encode(v any) ([]byte, error)
	// Decode unmarshals bytes into a value.
	Decode(data []byte, v any) error
}

// JSONCodec implements Codec using go-json-experiment/json (JSON v2).
type JSONCodec struct{}

// Encode marshals a value to JSON bytes.
func (JSONCodec) Encode(v any) ([]byte, error) {
	//nolint:wrapcheck // thin wrapper over json.Marshal
	return jsonv2.Marshal(v)
}

// Decode unmarshals JSON bytes into a value.
func (JSONCodec) Decode(data []byte, v any) error {
	//nolint:wrapcheck // thin wrapper over json.Unmarshal
	return jsonv2.Unmarshal(data, v)
}
