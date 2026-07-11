package encryption

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json/v2"
	"slices"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
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
	encoded := base64.URLEncoding.EncodeToString(c)

	return json.Marshal(encoded, json.Deterministic(true)) //nolint:wrapcheck // ciphertext encoding
}

func (c *Ciphertext) UnmarshalJSON(data []byte) error {
	var encoded string

	err := json.Unmarshal(data, &encoded, json.MatchCaseInsensitiveNames(true))
	if err != nil {
		return errorfamily.WrapInfrastructure(
			err,
			"encryption.unmarshal_ciphertext",
			"unmarshal ciphertext string",
		)
	}

	decoded, err := event.DecodeBase64String(encoded)
	if err != nil {
		return errorfamily.WrapInfrastructure(
			err,
			"encryption.decode_ciphertext",
			"decode ciphertext base64",
		)
	}

	*c = decoded

	return nil
}
