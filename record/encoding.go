package record

import (
	"errors"
	"fmt"
)

// Encoding is the payload codec stamp on a Record (ADR-0111): how the
// Payload bytes are encoded. It is the compact form of the canonical codec
// names ("json", "cbor") that the codec layers and the ADR-0044 envelope
// use, so Records stay cheap to carry and compare.
//
// The zero value (EncodingUnknown) means the payload carries no codec
// stamp: it is absent, opaque, or envelope-wrapped — the envelope stamps
// itself.
type Encoding uint8

const (
	// EncodingUnknown means the payload carries no codec stamp: absent,
	// opaque, or envelope-wrapped (the envelope carries its own stamp,
	// ADR-0044).
	EncodingUnknown Encoding = iota
	// EncodingJSON is the human-readable codec stamp ("json").
	EncodingJSON
	// EncodingCBOR is the deterministic binary codec stamp ("cbor").
	EncodingCBOR
)

// String returns the canonical codec name, or "" for EncodingUnknown — the
// same self-describing string form the codec layers use.
func (e Encoding) String() string {
	switch e {
	case EncodingJSON:
		return "json"
	case EncodingCBOR:
		return "cbor"
	case EncodingUnknown:
		return ""
	default:
		return ""
	}
}

// ErrUnknownEncoding is returned by ParseEncoding for names outside the
// canonical codec vocabulary.
var ErrUnknownEncoding = errors.New("record: unknown encoding")

// ParseEncoding maps a canonical codec name onto an Encoding. Unrecognized
// names return EncodingUnknown and an error wrapping ErrUnknownEncoding, so
// callers decide how to degrade (bridges stamp EncodingUnknown and keep
// carrying the payload).
func ParseEncoding(name string) (Encoding, error) {
	switch name {
	case "json":
		return EncodingJSON, nil
	case "cbor":
		return EncodingCBOR, nil
	default:
		return EncodingUnknown, fmt.Errorf("%w: %q", ErrUnknownEncoding, name)
	}
}
