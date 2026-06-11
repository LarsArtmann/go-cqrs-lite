package codec

import "github.com/fxamacker/cbor/v2"

// CBORCodec implements Codec using fxamacker/cbor with canonical encoding
// (RFC 7049: sorted map keys, shortest floats). Canonical mode is deterministic,
// making CBORCodec safe for content-addressed storage and cryptographic signing.
type CBORCodec struct{}

var _ Codec = CBORCodec{}

// cborEncMode provides canonical (deterministic) CBOR encoding with sorted map keys.
//
//nolint:gochecknoglobals // concurrency-safe EncMode, created once at package init
var cborEncMode = func() cbor.EncMode {
	em, err := cbor.CanonicalEncOptions().EncMode()
	if err != nil {
		panic("codec: failed to create CBOR canonical encoding mode: " + err.Error())
	}

	return em
}()

// cborDecMode provides the default CBOR decoding mode with duplicate map key
// enforcement. Decode uses a configured DecMode for explicit control over
// decoding behavior, mirroring the encode-side EncMode pattern.
//
//nolint:gochecknoglobals // concurrency-safe DecMode, created once at package init
var cborDecMode = func() cbor.DecMode {
	//nolint:exhaustruct // only DupMapKey is intentional; all other fields use library defaults
	opts := cbor.DecOptions{DupMapKey: cbor.DupMapKeyEnforcedAPF}

	dm, err := opts.DecMode()
	if err != nil {
		panic("codec: failed to create CBOR decoding mode: " + err.Error())
	}

	return dm
}()

func (CBORCodec) Encoding() Encoding { return EncodingCBOR }

// Encode marshals a value to canonical CBOR bytes with deterministic map ordering.
func (CBORCodec) Encode(v any) ([]byte, error) {
	//nolint:wrapcheck // thin wrapper over cbor EncMode.Marshal
	return cborEncMode.Marshal(v)
}

// Decode unmarshals CBOR bytes into a value.
func (CBORCodec) Decode(data []byte, v any) error {
	//nolint:wrapcheck // thin wrapper over cbor DecMode.Unmarshal
	return cborDecMode.Unmarshal(data, v)
}
