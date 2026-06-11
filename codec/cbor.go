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

func (CBORCodec) Encoding() Encoding { return EncodingCBOR }

// Encode marshals a value to canonical CBOR bytes with deterministic map ordering.
func (CBORCodec) Encode(v any) ([]byte, error) {
	//nolint:wrapcheck // thin wrapper over cbor EncMode.Marshal
	return cborEncMode.Marshal(v)
}

// Decode unmarshals CBOR bytes into a value.
func (CBORCodec) Decode(data []byte, v any) error {
	//nolint:wrapcheck // thin wrapper over cbor.Unmarshal
	return cbor.Unmarshal(data, v)
}
