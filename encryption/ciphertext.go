package encryption

import (
	"crypto/subtle"
	"encoding/base64"
	"slices"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
)

type Ciphertext []byte //nolint:recvcheck // value receiver for immutable type

func (c Ciphertext) IsZero() bool { return len(c) == 0 }

func (c Ciphertext) Equal(other Ciphertext) bool {
	return subtle.ConstantTimeCompare(c, other) == 1
}

func (c Ciphertext) Bytes() []byte { return slices.Clone(c) }

func (c Ciphertext) String() string {
	return base64.URLEncoding.EncodeToString(c)
}

func (c Ciphertext) MarshalJSON() ([]byte, error) {
	b, err := codec.MarshalBase64JSON(c)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"encryption.marshal_ciphertext",
			"marshal ciphertext",
		)
	}

	return b, nil
}

func (c *Ciphertext) UnmarshalJSON(data []byte) error {
	if err := codec.AssignBase64JSON(data, "encryption", "ciphertext", (*[]byte)(c)); err != nil {
		return errorfamily.WrapInfrastructure(
			err,
			"encryption.unmarshal_ciphertext",
			"unmarshal ciphertext",
		)
	}

	return nil
}
