package id

import (
	"crypto/rand"
	"fmt"
	"time"

	cbid "github.com/larsartmann/go-branded-id"
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

		return zero, fmt.Errorf("cannot parse empty string as AggregateID: %w", errEmptyString)
	}

	return cbid.NewID[AggregateMarker](s), nil
}

// MustParseAggregateID converts a string to an AggregateID, panicking on error.
func MustParseAggregateID(s string) AggregateID {
	parsed, err := ParseAggregateID(s)
	if err != nil {
		panic(fmt.Sprintf("id.MustParseAggregateID: %v (input: %q)", err, s))
	}

	return parsed
}
