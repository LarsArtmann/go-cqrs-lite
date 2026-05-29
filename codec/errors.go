package codec

import "errors"

var (
	ErrEncodeRawType = errors.New("raw codec: expected []byte")
	ErrDecodeRawType = errors.New("raw codec: expected *[]byte target")
)
