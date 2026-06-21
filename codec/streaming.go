package codec

import (
	"fmt"
	"io"

	"github.com/fxamacker/cbor/v2"
)

// NewCBOREncoder creates a streaming CBOR encoder that writes to w.
// Use for encoding large event batches without materializing the full
// byte slice in memory. The encoder uses the same canonical encoding mode
// as CBORCodec.
func NewCBOREncoder(w io.Writer) (*cbor.Encoder, error) {
	mode, err := canonicalEncMode()
	if err != nil {
		return nil, fmt.Errorf("codec: CBOR canonical encoding mode: %w", err)
	}

	return mode.NewEncoder(w), nil
}

// NewCBORDecoder creates a streaming CBOR decoder that reads from r.
// Use for decoding large event batches from a stream without loading all
// bytes into memory at once. The decoder uses the same decoding mode as
// CBORCodec.
func NewCBORDecoder(r io.Reader) (*cbor.Decoder, error) {
	mode, err := canonicalDecMode()
	if err != nil {
		return nil, fmt.Errorf("codec: CBOR decoding mode: %w", err)
	}

	return mode.NewDecoder(r), nil
}
