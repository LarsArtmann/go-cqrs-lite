package codec

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/fxamacker/cbor/v2"
)

// CBORCompactCodec is an opt-in Codec that uses stricter decoding than CBORCodec.
// It uses CoreDetEncOptions (RFC 8949 "Core Deterministic") for encoding and
// ExtraReturnErrors(UnknownField) for decoding — rejecting unknown fields on
// decode as a schema drift detection mechanism.
//
// This codec is NOT compatible with data written by CBORCodec (which uses
// CanonicalEncOptions with length-first sort). Use it only for new event stores.
//
// For even smaller payloads, consumers should add `cbor:",toarray"` struct tags
// to their event payload types. This encodes structs as positional CBOR arrays
// instead of maps, eliminating field-name string keys (~30-40% reduction).
// The toarray tag is separate from this codec — the codec enables the strict
// encoding/decoding modes, and the consumer opts into per-type compactness.
//
//	type MyEvent struct {
//	    _           struct{} `cbor:",toarray"`
//	    ID          string
//	    Payload     []byte
//	    OccurredAt  int64
//	}
type CBORCompactCodec struct{}

var _ Codec = CBORCompactCodec{}

// compactEncMode and compactDecMode use sync.OnceValue for the same reason
// as canonicalEncMode/canonicalDecMode in cbor.go — the options are hardcoded
// valid constants, so mode creation cannot fail.
var compactEncMode = sync.OnceValue(func() cbor.EncMode {
	opts := cbor.CoreDetEncOptions()
	opts.Time = cbor.TimeUnixDynamic

	mode, err := opts.EncMode()
	if err != nil {
		panic(fmt.Errorf("codec: compact CBOR EncMode creation failed: %w", err))
	}

	return mode
})

var compactDecMode = sync.OnceValue(func() cbor.DecMode {
	opts := cbor.DecOptions{ //nolint:exhaustruct // only strict-mode fields needed
		DupMapKey:         cbor.DupMapKeyEnforcedAPF,
		ExtraReturnErrors: cbor.ExtraDecErrorUnknownField,
	}

	mode, err := opts.DecMode()
	if err != nil {
		panic(fmt.Errorf("codec: compact CBOR DecMode creation failed: %w", err))
	}

	return mode
})

func (CBORCompactCodec) Encoding() Encoding { return EncodingCBOR }

// Encode marshals a value to compact CBOR bytes with deterministic ordering.
func (CBORCompactCodec) Encode(v any) ([]byte, error) {
	//nolint:wrapcheck // thin wrapper over cbor EncMode.Marshal
	return compactEncMode().Marshal(v)
}

// Decode unmarshals compact CBOR bytes into a value.
// Returns an error if the data contains unknown fields (schema drift detection).
func (CBORCompactCodec) Decode(data []byte, v any) error {
	//nolint:wrapcheck // thin wrapper over cbor DecMode.Unmarshal
	return compactDecMode().Unmarshal(data, v)
}

// EncodeToBuffer writes compact CBOR encoding of v directly into buf,
// avoiding the allocation that Encode returns. Implements BufferEncoder.
func (CBORCompactCodec) EncodeToBuffer(v any, buf *bytes.Buffer) error {
	//nolint:wrapcheck // thin wrapper over cbor Encoder.Encode
	return compactEncMode().NewEncoder(buf).Encode(v)
}
