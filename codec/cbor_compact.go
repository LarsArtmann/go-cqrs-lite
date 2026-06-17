package codec

import "github.com/fxamacker/cbor/v2"

// CBORCompactCodec is an opt-in Codec that produces smaller CBOR output by
// encoding structs as positional arrays (toarray) instead of maps.
//
// This codec is NOT compatible with data written by CBORCodec (which uses
// map encoding). Use it only for new event stores where backward compatibility
// with CBORCodec-encoded data is not required.
//
// For existing data, continue using CBORCodec. For new stores where payload
// size matters (e.g., high-volume event stores), CBORCompactCodec can reduce
// encoded size by 30-40% by eliminating field-name string keys.
//
// Encoding mode: CoreDetEncOptions (RFC 8949 "Core Deterministic").
// Decoding mode: DupMapKey enforcement + unknown field error (strict).
//
//	type MyEvent struct {
//	    _           struct{} `cbor:",toarray"`
//	    ID          string
//	    Payload     []byte
//	    OccurredAt  int64
//	}
//	// Encodes as [3]("event-id", bytes, 1234567890) instead of
//	// {1: "event-id", 2: bytes, 3: 1234567890}
type CBORCompactCodec struct{}

var _ Codec = CBORCompactCodec{}

//nolint:gochecknoglobals // concurrency-safe EncMode, created once at package init
var compactEncMode = func() cbor.EncMode {
	opts := cbor.CoreDetEncOptions()
	em, err := opts.EncMode()
	if err != nil {
		panic("codec: failed to create CBOR compact encoding mode: " + err.Error())
	}

	return em
}()

//nolint:gochecknoglobals // concurrency-safe DecMode, created once at package init
var compactDecMode = func() cbor.DecMode {
	opts := cbor.DecOptions{
		DupMapKey:         cbor.DupMapKeyEnforcedAPF,
		ExtraReturnErrors: cbor.ExtraDecErrorUnknownField,
	}
	dm, err := opts.DecMode()
	if err != nil {
		panic("codec: failed to create CBOR compact decoding mode: " + err.Error())
	}

	return dm
}()

func (CBORCompactCodec) Encoding() Encoding { return EncodingCBOR }

// Encode marshals a value to compact CBOR bytes with deterministic ordering.
func (CBORCompactCodec) Encode(v any) ([]byte, error) {
	//nolint:wrapcheck // thin wrapper over cbor EncMode.Marshal
	return compactEncMode.Marshal(v)
}

// Decode unmarshals compact CBOR bytes into a value.
// Returns an error if the data contains unknown fields (schema drift detection).
func (CBORCompactCodec) Decode(data []byte, v any) error {
	//nolint:wrapcheck // thin wrapper over cbor DecMode.Unmarshal
	return compactDecMode.Unmarshal(data, v)
}
