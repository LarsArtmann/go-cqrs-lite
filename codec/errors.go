package codec

import errorfamily "github.com/larsartmann/go-error-family"

var (
	ErrEncodeRawType = errorfamily.NewRejection(
		"codec.raw_encode_type",
		"raw codec: expected []byte",
	)
	ErrDecodeRawType = errorfamily.NewRejection(
		"codec.raw_decode_type",
		"raw codec: expected *[]byte target",
	)
	ErrInvalidCOSESign1 = errorfamily.NewRejection(
		"codec.invalid_cose_sign1",
		"COSE_Sign1 structure has an invalid number of elements",
	)
	ErrInvalidCOSEEncrypt0 = errorfamily.NewRejection(
		"codec.invalid_cose_encrypt0",
		"COSE_Encrypt0 structure has an invalid number of elements",
	)
)
