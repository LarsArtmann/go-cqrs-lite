package codec

import (
	"bytes"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"
)

// Encoding identifies the serialization format used for a payload.
type Encoding string

const (
	EncodingJSON Encoding = "json"
	EncodingCBOR Encoding = "cbor"
	EncodingRaw  Encoding = "raw"
)

// ErrUnknownEncoding is returned by [ForEncoding] when no built-in codec
// matches the requested encoding.
var ErrUnknownEncoding = errorfamily.NewRejection(
	"codec.unknown_encoding",
	"codec: unknown encoding (no built-in codec)",
)

// ForEncoding returns the built-in [Codec] for the given [Encoding].
// It resolves [EncodingJSON] → [JSONCodec] and [EncodingCBOR] → [CBORCodec].
//
// For unknown encodings (including [EncodingRaw] and custom values like
// "encrypted"), it returns [ErrUnknownEncoding]. Callers that need custom
// encoding support should build their own dispatch table.
//
// ForEncoding is the codec-level counterpart to [AutoDetect]: AutoDetect
// infers the encoding from raw bytes, ForEncoding resolves a known encoding
// stamp to its codec. Together they enable mixed-stream decoding — see
// [event.DecodePayloadAuto].
func ForEncoding(enc Encoding) (Codec, error) {
	switch enc { //nolint:exhaustive // Raw has no codec
	case EncodingJSON:
		return JSONCodec{}, nil
	case EncodingCBOR:
		return CBORCodec{}, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownEncoding, enc)
	}
}

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
