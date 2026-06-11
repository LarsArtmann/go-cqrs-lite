package codec

// Encoding identifies the serialization format used for a payload.
type Encoding string

const (
	EncodingJSON Encoding = "json"
	EncodingCBOR Encoding = "cbor"
	EncodingRaw  Encoding = "raw"
)

// Codec serializes and deserializes values with a declared encoding.
type Codec interface {
	Encoding() Encoding
	Encode(v any) ([]byte, error)
	Decode(data []byte, v any) error
}
