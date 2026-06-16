package id

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	cbid "github.com/larsartmann/go-branded-id"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/oklog/ulid/v2"
)

// AggregateMarker is a phantom type for branding AggregateIDs.
// Export it so domain packages can create domain-specific IDs interoperable with AggregateID.
type AggregateMarker struct{}

// AggregateID is a strongly-typed identifier for aggregate roots.
// Unlike other ID types (EventID, UserID), AggregateID is backed by a string
// rather than ulid.ULID. This allows it to represent both auto-generated ULIDs
// and domain-specific identifiers like "lock_user1_user2".
//
// NewAggregateID() still generates a ULID-based string for new aggregates,
// but ParseAggregateID() accepts any non-empty string for compatibility with
// existing data.
type AggregateID = cbid.ID[AggregateMarker, string]

// NewAggregateID generates a new AggregateID backed by a ULID string.
func NewAggregateID() AggregateID {
	ulidStr := ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()

	return cbid.NewID[AggregateMarker](ulidStr)
}

// ParseAggregateID converts a string to an AggregateID.
// Accepts any non-empty string — not limited to ULID format.
// This supports both new ULID-based IDs and legacy domain-specific IDs.
func ParseAggregateID(s string) (AggregateID, error) {
	if s == "" {
		var zero AggregateID

		return zero, errorfamily.Wrapf(
			errEmptyString,
			errorfamily.Rejection,
			"id.parse_aggregate_empty",
			"cannot parse empty string as AggregateID",
		)
	}

	return cbid.NewID[AggregateMarker](s), nil
}

// DeriveAggregateID creates a deterministic AggregateID from a namespace and
// one or more key strings using SHA-256. Same inputs always produce the same ID.
// Useful for stable IDs in idempotent workflows (e.g., "lock:" + userID + ":" + resourceID).
func DeriveAggregateID(namespace string, keys ...string) AggregateID {
	h := sha256.New()
	_, _ = h.Write([]byte(namespace))

	for _, k := range keys {
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(k))
	}

	return cbid.NewID[AggregateMarker](hex.EncodeToString(h.Sum(nil)))
}

// AggregateIDFrom creates an AggregateID from any fmt.Stringer.
// Useful for interop with consumer-side branded IDs that implement String().
func AggregateIDFrom(s fmt.Stringer) AggregateID {
	return cbid.NewID[AggregateMarker](s.String())
}
