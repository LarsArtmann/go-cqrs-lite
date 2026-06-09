package signing

import (
	"crypto/hmac"
	"encoding/base64"
	"encoding/json"
	"slices"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// Signature is an opaque, serializable event signature.
// Use Bytes() to get the raw signature data.
// Use String() for a human-readable encoding (URL-safe base64).
//
//nolint:recvcheck // UnmarshalJSON must use pointer receiver to mutate; Signature is []byte (reference type)
type Signature []byte

// Bytes returns a copy of the raw signature bytes.
// The clone is retained for defensive consistency: callers cannot mutate the
// signature through the returned slice, matching the pattern used by
// [event.Event.Payload] and [event.Metadata.Clone].
func (s Signature) Bytes() []byte {
	return slices.Clone(s)
}

// IsZero reports whether the signature is empty.
func (s Signature) IsZero() bool { return len(s) == 0 }

// Equal reports whether two signatures are identical (constant-time).
func (s Signature) Equal(other Signature) bool {
	return hmac.Equal(s, other)
}

// String returns the URL-safe base64 encoding of the signature.
func (s Signature) String() string {
	return base64.URLEncoding.EncodeToString(s)
}

// MarshalJSON encodes the signature as a URL-safe base64 JSON string.
func (s Signature) MarshalJSON() ([]byte, error) {
	encoded := base64.URLEncoding.EncodeToString(s)

	return json.Marshal(encoded) //nolint:wrapcheck // signature encoding, no wrapping needed
}

// UnmarshalJSON decodes a URL-safe base64 JSON string into the signature.
// Falls back to standard base64 for backward compatibility.
func (s *Signature) UnmarshalJSON(data []byte) error {
	var encoded string

	err := json.Unmarshal(data, &encoded)
	if err != nil {
		return event.WrapInfrastructure(
			err,
			"signing.unmarshal_signature",
			"unmarshal signature string",
		)
	}

	decoded, decodeErr := base64.URLEncoding.DecodeString(encoded)
	if decodeErr != nil {
		var fallbackErr error

		decoded, fallbackErr = base64.StdEncoding.DecodeString(encoded)
		if fallbackErr != nil {
			return event.Newf(
				event.Infrastructure,
				"signing.decode_signature",
				"decode signature: URL-safe: %v, standard: %v",
				decodeErr,
				fallbackErr,
			)
		}
	}

	*s = decoded

	return nil
}
