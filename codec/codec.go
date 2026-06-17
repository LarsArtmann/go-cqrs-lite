package codec

import "bytes"

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

// BufferEncoder is an optional interface that Codecs can implement for
// zero-allocation encoding. Instead of allocating a new []byte on every
// Encode call, EncodeToBuffer writes directly into a caller-provided buffer.
// Callers can reuse the buffer across calls to eliminate GC pressure in
// hot paths (batch event publishing, bulk snapshot saving).
type BufferEncoder interface {
	EncodeToBuffer(v any, buf *bytes.Buffer) error
}
