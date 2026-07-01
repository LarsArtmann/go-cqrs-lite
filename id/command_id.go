package id

import (
	"crypto/sha256"

	cbid "github.com/larsartmann/go-branded-id"
	"github.com/oklog/ulid/v2"
)

// CommandMarker is a phantom type for branding CommandIDs.
type CommandMarker struct{}

// CommandID is a branded unique identifier for command messages.
type CommandID = Of[CommandMarker]

// NewCommandID generates a new unique CommandID.
func NewCommandID() CommandID {
	return New[CommandMarker]()
}

// ParseCommandID parses a string into a CommandID.
// Returns an error if the string is not a valid ULID.
func ParseCommandID(s string) (CommandID, error) {
	return Parse[CommandMarker](s)
}

// ULID layout: 6-byte timestamp (48-bit ms precision) + 10-byte randomness.
// See https://github.com/ulid/spec
const (
	ulidTimestampLen = 6
	ulidRandomLen    = 10
)

// DeriveCommandID creates a deterministic CommandID from a namespace and one or
// more key strings using SHA-256. Same inputs always produce the same ID.
//
// This is the idempotency primitive for command derivation: re-deriving a
// command from the same source inputs yields the same CommandID, so an
// idempotency store keyed on the command ID (see
// idempotency.CommandIDKey) deduplicates at-least-once redeliveries.
//
// The result is a valid ULID-encoded CommandID, but the timestamp portion is
// zeroed (epoch 1970-01-01) as a sentinel: [ULID] on a derived ID returns the
// zero time, signalling "not a wall-clock timestamp". The 80-bit randomness
// portion carries 10 bytes of the SHA-256 digest — 2^80 entropy is far beyond
// any realistic collision risk for idempotency keys.
//
// Use [IsDerivedCommandID] to check whether a CommandID was produced by
// DeriveCommandID (timestamp is zero) vs [NewCommandID] (real timestamp).
func DeriveCommandID(namespace string, keys ...string) CommandID {
	h := sha256.New()
	_, _ = h.Write([]byte(namespace))

	for _, k := range keys {
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(k))
	}

	var derived ulid.ULID
	copy(
		derived[ulidTimestampLen:],
		h.Sum(nil)[:ulidRandomLen],
	) // entropy into randomness; timestamp stays zero

	return cbid.NewID[CommandMarker](derived)
}

// IsDerivedCommandID reports whether the CommandID was produced by
// [DeriveCommandID] (timestamp zeroed) rather than [NewCommandID] (real
// wall-clock timestamp). Derived IDs are deterministic idempotency keys, not
// time-ordered identifiers.
func IsDerivedCommandID(id CommandID) bool {
	u := id.Get()

	for _, b := range u[:ulidTimestampLen] {
		if b != 0 {
			return false
		}
	}

	return true
}
