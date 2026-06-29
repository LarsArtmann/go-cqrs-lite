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

// DeriveCommandID creates a deterministic CommandID from a namespace and one or
// more key strings using SHA-256. Same inputs always produce the same ID.
//
// This is the idempotency primitive for command derivation: re-deriving a
// command from the same source inputs yields the same CommandID, so an
// idempotency store keyed on the command ID (see
// idempotency.CommandIDKey) deduplicates at-least-once redeliveries.
//
// Unlike [DeriveAggregateID] (which yields a string-backed hex ID), the result
// is a valid ULID-encoded CommandID — the first 16 bytes of the SHA-256 digest
// are packed into a [ulid.ULID]. The timestamp portion is therefore NOT
// wall-clock derived: the ID trades time-ordering for determinism. It is safe
// wherever a CommandID is used as an opaque identifier or idempotency key.
func DeriveCommandID(namespace string, keys ...string) CommandID {
	h := sha256.New()
	_, _ = h.Write([]byte(namespace))

	for _, k := range keys {
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(k))
	}

	var u ulid.ULID
	copy(u[:], h.Sum(nil)[:16])

	return cbid.NewID[CommandMarker](u)
}
