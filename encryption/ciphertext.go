package encryption

import (
	"crypto/subtle"
	"encoding/base64"
	"slices"

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
	return codec.MarshalBase64JSON(c)
}

func (c *Ciphertext) UnmarshalJSON(data []byte) error {
	return codec.AssignBase64JSON(data, "encryption", "ciphertext", (*[]byte)(c))
}
