package encryption

import (
	"encoding/base64"
	"encoding/json"
	"slices"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

const gcmNonceSize = 12 //nolint:unused // referenced by Nonce/Data accessors

type Ciphertext []byte

func (c Ciphertext) Nonce() []byte {
	if len(c) < gcmNonceSize {
		return nil
	}

	return slices.Clone(c[:gcmNonceSize])
}

func (c Ciphertext) Data() []byte {
	if len(c) < gcmNonceSize {
		return nil
	}

	return slices.Clone(c[gcmNonceSize:])
}

func (c Ciphertext) IsZero() bool { return len(c) == 0 }

func (c Ciphertext) Equal(other Ciphertext) bool {
	if len(c) != len(other) {
		return false
	}

	for i := range c {
		if c[i] != other[i] {
			return false
		}
	}

	return true
}

func (c Ciphertext) Bytes() []byte { return slices.Clone(c) }

func (c Ciphertext) String() string {
	return base64.URLEncoding.EncodeToString(c)
}

func (c Ciphertext) MarshalJSON() ([]byte, error) {
	encoded := base64.URLEncoding.EncodeToString(c)

	return json.Marshal(encoded) //nolint:wrapcheck // ciphertext encoding, no wrapping needed
}

func (c *Ciphertext) UnmarshalJSON(data []byte) error {
	var encoded string

	err := json.Unmarshal(data, &encoded)
	if err != nil {
		return event.WrapInfrastructure(
			err,
			"encryption.unmarshal_ciphertext",
			"unmarshal ciphertext string",
		)
	}

	decoded, decodeErr := base64.URLEncoding.DecodeString(encoded)
	if decodeErr != nil {
		var fallbackErr error

		decoded, fallbackErr = base64.StdEncoding.DecodeString(encoded)
		if fallbackErr != nil {
			return event.Newf(
				event.Infrastructure,
				"encryption.decode_ciphertext",
				"decode ciphertext: URL-safe: %v, standard: %v",
				decodeErr,
				fallbackErr,
			)
		}
	}

	*c = decoded

	return nil
}
