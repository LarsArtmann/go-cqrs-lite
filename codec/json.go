package codec

import (
	"bytes"
	"encoding/json/v2"
)

// JSONCodec implements Codec using encoding/json.
type JSONCodec struct{}

var _ Codec = JSONCodec{}

func (JSONCodec) Encoding() Encoding { return EncodingJSON }

// Encode marshals a value to JSON bytes.
func (JSONCodec) Encode(v any) ([]byte, error) {
	//nolint:wrapcheck // thin wrapper over json.Marshal
	return json.Marshal(v)
}

// Decode unmarshals JSON bytes into a value.
func (JSONCodec) Decode(data []byte, v any) error {
	//nolint:wrapcheck // thin wrapper over json.Unmarshal
	return json.Unmarshal(data, v, json.MatchCaseInsensitiveNames(true))
}

// EncodeToBuffer writes JSON encoding of v directly into buf,
// avoiding the allocation that Encode returns. Implements BufferEncoder.
func (JSONCodec) EncodeToBuffer(v any, buf *bytes.Buffer) error {
	//nolint:wrapcheck // thin wrapper over json.Encoder.Encode
	return json.MarshalWrite(buf, v)
}
