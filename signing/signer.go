package signing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// Signer computes cryptographic signatures for events.
// Implementations are stateless and safe for concurrent use.
type Signer interface {
	// Sign computes a cryptographic signature for the given event.
	// The signature covers event ID, type, aggregate, version, payload, and occurredAt.
	Sign(event event.Event) (Signature, error)
}

// Verifier checks cryptographic signatures on events.
// Implementations are stateless and safe for concurrent use.
type Verifier interface {
	// Verify checks that the signature matches the event's content.
	// Returns ErrInvalidSignature if the signature does not match.
	// Returns ErrNilSignature if sig is nil.
	Verify(event event.Event, sig Signature) error
}

// SignerVerifier combines signing and verification capabilities.
// NewHMAC returns a type that implements this interface because the same key handles both.
type SignerVerifier interface {
	Signer
	Verifier
}

// Signature is an opaque, serializable event signature.
// Use Bytes() to get the raw signature data.
// Use String() for a human-readable encoding (URL-safe base64).
//
//nolint:recvcheck // UnmarshalJSON must use pointer receiver to mutate; Signature is []byte (reference type)
type Signature []byte

// Bytes returns a copy of the raw signature bytes.
func (s Signature) Bytes() []byte {
	cp := make([]byte, len(s))
	copy(cp, s)

	return cp
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
		return fmt.Errorf("unmarshal signature string: %w", err)
	}

	decoded, decodeErr := base64.URLEncoding.DecodeString(encoded)
	if decodeErr != nil {
		var fallbackErr error

		decoded, fallbackErr = base64.StdEncoding.DecodeString(encoded)
		if fallbackErr != nil {
			return fmt.Errorf(
				"decode signature: URL-safe: %w, standard: %w",
				decodeErr,
				fallbackErr,
			)
		}
	}

	*s = decoded

	return nil
}

// canonicalFormatVersion identifies the canonical payload format version.
// If the format changes, this constant must be incremented so that
// old and new signatures are distinguishable.
const canonicalFormatVersion = "go-cqrs-lite/signing/v1"

// canonicalPayload builds a deterministic byte representation of an event
// for signing. It excludes the signature itself and non-deterministic fields
// like metadata to prevent circular signing issues.
//
// Schema version is included because it semantically identifies the payload
// structure. Changing the schema version without changing payload content is
// a meaningful event transformation that must be reflected in the signature.
// The payload itself is SHA-256 hashed to keep the canonical representation
// bounded regardless of payload size.
//
// The output is prefixed with a format version tag so that future format
// changes produce different signatures, preventing cross-version collisions.
func canonicalPayload(evt event.Event) []byte {
	if evt == nil {
		return nil
	}

	id := evt.ID().String()
	typ := string(evt.Type())
	aggID := evt.AggregateID().String()
	aggType := string(evt.AggregateType())
	version := evt.Version().Int()
	schemaVer := evt.SchemaVersion().Int()
	occurred := evt.OccurredAt().Format(time.RFC3339Nano)
	payload := evt.Payload()

	// Deterministic string format: each field length-prefixed
	// Format: len(field):field (avoids delimiter collision)
	var buf []byte

	buf = appendLenPrefixed(buf, canonicalFormatVersion)
	buf = appendLenPrefixed(buf, id)
	buf = appendLenPrefixed(buf, typ)
	buf = appendLenPrefixed(buf, aggID)
	buf = appendLenPrefixed(buf, aggType)
	buf = appendLenPrefixed(buf, strconv.Itoa(version))
	buf = appendLenPrefixed(buf, strconv.Itoa(schemaVer))
	buf = appendLenPrefixed(buf, occurred)

	if len(payload) > 0 {
		// Hash payload to keep canonical representation bounded
		h := sha256.Sum256(payload)
		buf = append(buf, h[:]...)
	}

	return buf
}

const lengthPrefixSize = 4

func appendLenPrefixed(buf []byte, s string) []byte {
	b := []byte(s)
	lenBuf := make([]byte, lengthPrefixSize)
	binary.BigEndian.PutUint32(
		lenBuf,
		uint32(len(b)), //nolint:gosec // length fits in uint32 for any reasonable string
	)

	buf = append(buf, lenBuf...)
	buf = append(buf, b...)

	return buf
}
