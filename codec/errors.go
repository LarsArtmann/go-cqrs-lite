package codec

import errorfamily "github.com/larsartmann/go-error-family"

var (
	ErrEncodeRawType = errorfamily.NewRejection("codec.raw_encode_type", "raw codec: expected []byte")
	ErrDecodeRawType = errorfamily.NewRejection("codec.raw_decode_type", "raw codec: expected *[]byte target")
)
