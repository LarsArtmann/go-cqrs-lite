package event

import (
	"encoding/base64"

	errorfamily "github.com/larsartmann/go-error-family"
)

// DecodeBase64String decodes a base64-encoded string, trying URL-safe
// encoding first, then falling back to standard base64 for backward
// compatibility with legacy consumers.
//
// Exported so that downstream modules (signing, encryption) can share
// a single implementation of the URL-safe→standard fallback pattern.
func DecodeBase64String(encoded string) ([]byte, error) {
	decoded, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		decoded, err = base64.StdEncoding.DecodeString(encoded)
	}

	if err != nil {
		return decoded, errorfamily.Wrapf(
			err,
			Corruption,
			"event.base64_decode",
			"encoded=%v",
			encoded,
		)
	}

	return decoded, nil
}
