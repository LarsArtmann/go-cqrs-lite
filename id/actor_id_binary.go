package id

import (
	"fmt"
)

// MarshalBinary implements encoding.BinaryMarshaler. The binary form is the
// self-describing "kind:raw" prefixed string, or empty for the zero ActorID.
//
// Codecs with binary-marshaling support (fxamacker/cbor, used by go-codec's
// CBORCodec default in the typed stores) delegate to this method. Without it,
// ActorID's unexported fields are silently dropped and the decoded value comes
// back zero — an audit-trail data loss.
func (a ActorID) MarshalBinary() ([]byte, error) {
	return []byte(a.PrefixedString()), nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler, reversing
// [ActorID.MarshalBinary]. Empty data yields the zero ActorID.
func (a *ActorID) UnmarshalBinary(data []byte) error {
	if len(data) == 0 {
		*a = ActorID{}
		return nil
	}

	parsed, err := ParseActorID(string(data))
	if err != nil {
		return fmt.Errorf("ActorID binary: %w", err)
	}

	*a = parsed

	return nil
}
