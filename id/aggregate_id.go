package id

import (
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
//
// # Why string-backed, not ULID-backed
//
// All other branded IDs (EventID, UserID, CommandID, etc.) are backed by
// ulid.ULID via Of[T]. AggregateID is the exception: it is backed by plain
// string. This is a deliberate design decision, not an oversight.
//
// Aggregate IDs have a richer lifecycle than other IDs:
//
//   - NewAggregateID() generates a ULID — the common case for new aggregates.
//   - DeriveAggregateID() creates a deterministic SHA-256 hash — for stable
//     IDs in idempotent workflows (e.g., "lock:user1:resource2").
//   - AggregateIDFrom() accepts any Stringer — for consumer-side branded IDs.
//   - ParseAggregateID() accepts any non-empty string — for legacy data,
//     migration imports, and domain-specific naming schemes.
//
// Forcing ULID-backing would break DeriveAggregateID (SHA-256 hashes are not
// ULIDs) and prevent consumers from using meaningful domain identifiers.
//
// # When you need ULID guarantees
//
// Use ParseAggregateIDStrict to validate that an AggregateID IS a ULID.
// Use AggregateTimestamp to extract the embedded timestamp.
// Use IsAggregateIDULID to check at runtime.
//
// String comparison of ULID-formatted AggregateIDs preserves chronological
// order (Crockford base32 is designed for this), so standard sorting works
// correctly for ULID-generated IDs.
type AggregateID = cbid.ID[AggregateMarker, string]

// NewAggregateID generates a new AggregateID backed by a ULID string.
// The ID is chronologically sortable and embeds a timestamp (via AggregateTimestamp).
func NewAggregateID() AggregateID {
	ulidStr := newULID().String()

	return cbid.NewID[AggregateMarker](ulidStr)
}

// ParseAggregateID converts a string to an AggregateID.
// Accepts any non-empty string — not limited to ULID format.
// This supports ULID-based IDs, SHA-256 derived IDs, and domain-specific IDs.
//
// For ULID validation, use ParseAggregateIDStrict.
func ParseAggregateID(s string) (AggregateID, error) {
	if s == "" {
		var zero AggregateID

		return zero, errorfamily.Wrapf(
			ErrEmptyString,
			errorfamily.Rejection,
			"id.parse_aggregate_empty",
			"cannot parse empty string as AggregateID",
		)
	}

	return cbid.NewID[AggregateMarker](s), nil
}

// ParseAggregateIDStrict converts a ULID string to an AggregateID, validating
// that the string is a well-formed ULID.
//
// Use this when you need ULID guarantees: chronological sortability, timestamp
// extraction (via AggregateTimestamp), or interop with ULID-backed ID types.
//
// For a lenient parse that accepts any non-empty string, use ParseAggregateID.
func ParseAggregateIDStrict(s string) (AggregateID, error) {
	if s == "" {
		var zero AggregateID

		return zero, errorfamily.Wrapf(
			ErrEmptyString,
			errorfamily.Rejection,
			"id.parse_aggregate_strict_empty",
			"cannot parse empty string as AggregateID",
		)
	}

	ulidVal, err := ulid.Parse(s)
	if err != nil {
		var zero AggregateID

		return zero, errorfamily.Wrapf(
			err,
			errorfamily.Rejection,
			"id.parse_aggregate_strict_not_ulid",
			"AggregateID %q is not a valid ULID",
			s,
		)
	}

	return cbid.NewID[AggregateMarker](ulidVal.String()), nil
}

// IsAggregateIDULID reports whether the AggregateID is a valid ULID.
// Returns false for SHA-256 derived IDs, domain-specific IDs, and empty IDs.
func IsAggregateIDULID(id AggregateID) bool {
	_, err := ulid.Parse(id.Get())

	return err == nil
}

// AggregateTimestamp extracts the embedded timestamp from a ULID-formatted
// AggregateID. Returns an error if the ID is not a valid ULID (e.g., a
// SHA-256 derived ID or a domain-specific string).
//
// For a predicate check, use IsAggregateIDULID.
func AggregateTimestamp(id AggregateID) (time.Time, error) {
	ulidVal, err := ulid.Parse(id.Get())
	if err != nil {
		return time.Time{}, errorfamily.Wrapf(
			err,
			errorfamily.Rejection,
			"id.aggregate_timestamp_not_ulid",
			"AggregateID %q is not a valid ULID, cannot extract timestamp",
			id.Get(),
		)
	}

	return ulid.Time(ulidVal.Time()), nil
}

// DeriveAggregateID creates a deterministic AggregateID from a namespace and
// one or more key strings using SHA-256. Same inputs always produce the same ID.
// Useful for stable IDs in idempotent workflows (e.g., "lock:" + userID + ":" + resourceID).
//
// The resulting ID is NOT a ULID — IsAggregateIDULID returns false, and
// AggregateTimestamp returns an error.
func DeriveAggregateID(namespace string, keys ...string) AggregateID {
	return cbid.NewID[AggregateMarker](hex.EncodeToString(hashNamespacedKeys(namespace, keys...)))
}

// AggregateIDFrom creates an AggregateID from any fmt.Stringer.
// Useful for interop with consumer-side branded IDs that implement String().
func AggregateIDFrom(s fmt.Stringer) AggregateID {
	return cbid.NewID[AggregateMarker](s.String())
}
