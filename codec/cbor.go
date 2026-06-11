package codec

import "github.com/fxamacker/cbor/v2"

// CBORCodec implements Codec using fxamacker/cbor (IETF STD 94 / RFC 8949).
// Encoding uses canonical mode with sorted map keys for deterministic output.
type CBORCodec struct{}

var _ Codec = CBORCodec{}

// cborEncMode provides canonical (deterministic) CBOR encoding with sorted map keys.
//
//nolint:gochecknoglobals // concurrency-safe EncMode, created once at package init
var cborEncMode, _ = cbor.CanonicalEncOptions().EncMode()

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
