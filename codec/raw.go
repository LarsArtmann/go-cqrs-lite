package codec

import (
	"encoding/json/jsontext"

	errorfamily "github.com/larsartmann/go-error-family"
)

// RawCodec implements Codec as a passthrough for []byte.
// Encode accepts only []byte (or json.RawMessage). Decode copies data into a *[]byte target.
type RawCodec struct{}

var _ Codec = RawCodec{}

func (RawCodec) Encoding() Encoding { return EncodingRaw }

// Encode returns v as-is if it is []byte or json.RawMessage.
func (RawCodec) Encode(v any) ([]byte, error) {
	switch b := v.(type) {
	case []byte:
		return b, nil
	case jsontext.Value:
		return b, nil
	default:
		return nil, errorfamily.Wrapf(
			ErrEncodeRawType,
			errorfamily.Rejection,
			"codec.raw_encode_type",
			"got %T",
			v,
		)
	}
}

// Decode copies data into target, which must be *[]byte.
func (RawCodec) Decode(data []byte, v any) error {
	p, ok := v.(*[]byte)
	if !ok {
		return errorfamily.Wrapf(
			ErrDecodeRawType,
			errorfamily.Rejection,
			"codec.raw_decode_type",
			"got %T",
			v,
		)
	}

	cp := make([]byte, len(data))
	copy(cp, data)
	*p = cp

	return nil
}
