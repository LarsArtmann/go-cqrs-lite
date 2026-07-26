package codec

import (
	"encoding/base64"
	"encoding/json/v2"

	errorfamily "github.com/larsartmann/go-error-family"
)

// DecodeBase64String decodes a base64-encoded string, trying URL-safe
// encoding first, then falling back to standard base64 for backward
// compatibility with legacy consumers.
func DecodeBase64String(encoded string) ([]byte, error) {
	decoded, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		decoded, err = base64.StdEncoding.DecodeString(encoded)
	}

	if err != nil {
		return decoded, errorfamily.Wrapf(
			err,
			errorfamily.Corruption,
			"codec.base64_decode",
			"encoded=%v",
			encoded,
		)
	}

	return decoded, nil
}

// MarshalBase64JSON encodes raw bytes as a URL-safe base64 JSON string.
// Used by types that wrap []byte and need deterministic JSON marshalling
// (encryption.Ciphertext, signing.Signature, event.EventID, etc.).
func MarshalBase64JSON(raw []byte) ([]byte, error) {
	encoded := base64.URLEncoding.EncodeToString(raw)

	return json.Marshal(encoded, json.Deterministic(true)) //nolint:wrapcheck // base64 encoding
}

// MarshalBase64JSONWithModule encodes raw bytes as base64 JSON and wraps any
// error with the given module and noun (e.g. "encryption", "ciphertext" →
// "encryption.marshal_ciphertext"). Symmetric with [AssignBase64JSON] —
// collapses the standard MarshalJSON body to one call.
func MarshalBase64JSONWithModule(raw []byte, module, noun string) ([]byte, error) {
	b, err := MarshalBase64JSON(raw)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err,
			module+".marshal_"+noun, "marshal "+noun)
	}

	return b, nil
}

// UnmarshalBase64JSON unmarshals a JSON string field and decodes it from
// URL-safe (or standard) base64. The module and noun parameters produce
// meaningful error locations: e.g. module="signing", noun="signature" yields
// "signing.unmarshal_signature" and "signing.decode_signature".
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

// AssignBase64JSON unmarshals data and assigns the decoded bytes to *target.
// Convenience for types whose underlying type is []byte (encryption.Ciphertext,
// signing.Signature, etc.) — collapses the standard UnmarshalJSON body to one
// call so each []byte wrapper type does not repeat the decode+assign boilerplate.
func AssignBase64JSON(data []byte, module, noun string, target *[]byte) error {
	decoded, err := UnmarshalBase64JSON(data, module, noun)
	if err != nil {
		return err
	}

	*target = decoded

	return nil
}
