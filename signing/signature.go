package signing

import (
	"crypto/hmac"
	"encoding/base64"
	"slices"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
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

// compareSig returns ErrInvalidSignature when expected and actual differ,
// nil when they match. Shared verify logic across all signer implementations.
func compareSig(expected, actual Signature) error {
	if !expected.Equal(actual) {
		return ErrInvalidSignature
	}

	return nil
}

// String returns the URL-safe base64 encoding of the signature.
func (s Signature) String() string {
	return base64.URLEncoding.EncodeToString(s)
}

// MarshalJSON encodes the signature as a URL-safe base64 JSON string.
func (s Signature) MarshalJSON() ([]byte, error) {
	return codec.MarshalBase64JSONWithModule(s, "signing", "signature")
}

// UnmarshalJSON decodes a URL-safe base64 JSON string into the signature.
// Falls back to standard base64 for backward compatibility.
func (s *Signature) UnmarshalJSON(data []byte) error {
	if err := codec.AssignBase64JSON(data, "signing", "signature", (*[]byte)(s)); err != nil {
		return errorfamily.WrapInfrastructure(
			err,
			"signing.unmarshal_signature",
			"unmarshal signature",
		)
	}

	return nil
}
