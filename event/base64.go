package event

import (
	"encoding/base64"
	"encoding/json/v2"

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

// UnmarshalBase64JSON unmarshals a JSON string field and decodes it from
// URL-safe (or standard) base64. The module and noun parameters produce
// meaningful error locations: e.g. module="signing", noun="signature" yields
// "signing.unmarshal_signature" and "signing.decode_signature".
//
// Shared by encryption.Ciphertext and signing.Signature so both follow the
// same URL-safe→standard fallback decode path.
func UnmarshalBase64JSON(data []byte, module, noun string) ([]byte, error) {
	var encoded string

	err := json.Unmarshal(data, &encoded, json.MatchCaseInsensitiveNames(true))
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err,
			module+".unmarshal_"+noun, "unmarshal "+noun+" string")
	}

	decoded, err := DecodeBase64String(encoded)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err,
			module+".decode_"+noun, "decode "+noun+" base64")
	}

	return decoded, nil
}
