package codec

import (
	"io"

	"github.com/fxamacker/cbor/v2"
)

// NewCBOREncoder creates a streaming CBOR encoder that writes to w.
// Use for encoding large event batches without materializing the full
// byte slice in memory. The encoder uses the same canonical encoding mode
// as CBORCodec.
func NewCBOREncoder(w io.Writer) *cbor.Encoder {
	return cborEncMode.NewEncoder(w)
}

// NewCBORDecoder creates a streaming CBOR decoder that reads from r.
// Use for decoding large event batches from a stream without loading all
// bytes into memory at once. The decoder uses the same decoding mode as
// CBORCodec.
func NewCBORDecoder(r io.Reader) *cbor.Decoder {
	return cborDecMode.NewDecoder(r)
}
