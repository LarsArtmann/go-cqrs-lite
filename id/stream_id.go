package id

import (
	"encoding/hex"
	"fmt"
	"time"

	cbid "github.com/larsartmann/go-branded-id"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/oklog/ulid/v2"
)

// StreamMarker is a phantom type for branding StreamIDs.
// Export it so domain packages can create domain-specific IDs interoperable with StreamID.
type StreamMarker struct{}

// StreamID is a strongly-typed identifier for an event stream.
//
// # Why string-backed, not ULID-backed
//
// All other branded IDs (EventID, UserID, CommandID, etc.) are backed by
// ulid.ULID via Of[T]. StreamID is the exception: it is backed by plain
// string. This is a deliberate design decision, not an oversight.
//
// Stream IDs have a richer lifecycle than other IDs:
//
//   - NewStreamID() generates a ULID — the common case for new streams.
//   - DeriveStreamID() creates a deterministic SHA-256 hash — for stable
//     IDs in idempotent workflows (e.g., "lock:user1:resource2").
//   - StreamIDFrom() accepts any Stringer — for consumer-side branded IDs.
//   - ParseStreamID() accepts any non-empty string — for legacy data,
//     migration imports, and domain-specific naming schemes.
//
// Forcing ULID-backing would break DeriveStreamID (SHA-256 hashes are not
// ULIDs) and prevent consumers from using meaningful domain identifiers.
//
// # When you need ULID guarantees
//
// Use ParseStreamIDStrict to validate that a StreamID IS a ULID.
// Use StreamTimestamp to extract the embedded timestamp.
// Use IsStreamIDULID to check at runtime.
//
// String comparison of ULID-formatted StreamIDs preserves chronological
// order (Crockford base32 is designed for this), so standard sorting works
// correctly for ULID-generated IDs.
type StreamID = cbid.ID[StreamMarker, string]

// NewStreamID generates a new StreamID backed by a ULID string.
// The ID is chronologically sortable and embeds a timestamp (via StreamTimestamp).
func NewStreamID() StreamID {
	ulidStr := newULID().String()

	return cbid.NewID[StreamMarker](ulidStr)
}

// ParseStreamID converts a string to a StreamID.
// Accepts any non-empty string — not limited to ULID format.
// This supports ULID-based IDs, SHA-256 derived IDs, and domain-specific IDs.
//
// For ULID validation, use ParseStreamIDStrict.
func ParseStreamID(s string) (StreamID, error) {
	if s == "" {
		var zero StreamID

		return zero, errorfamily.Wrapf(
			ErrEmptyString,
			errorfamily.Rejection,
			"id.parse_stream_empty",
			"cannot parse empty string as StreamID",
		)
	}

	return cbid.NewID[StreamMarker](s), nil
}

// ParseStreamIDStrict converts a ULID string to a StreamID, validating
// that the string is a well-formed ULID.
//
// Use this when you need ULID guarantees: chronological sortability, timestamp
// extraction (via StreamTimestamp), or interop with ULID-backed ID types.
//
// For a lenient parse that accepts any non-empty string, use ParseStreamID.
func ParseStreamIDStrict(s string) (StreamID, error) {
	if s == "" {
		var zero StreamID

		return zero, errorfamily.Wrapf(
			ErrEmptyString,
			errorfamily.Rejection,
			"id.parse_stream_strict_empty",
			"cannot parse empty string as StreamID",
		)
	}

	ulidVal, err := ulid.Parse(s)
	if err != nil {
		var zero StreamID

		return zero, errorfamily.Wrapf(
			err,
			errorfamily.Rejection,
			"id.parse_stream_strict_not_ulid",
			"StreamID %q is not a valid ULID",
			s,
		)
	}

	return cbid.NewID[StreamMarker](ulidVal.String()), nil
}

// IsStreamIDULID reports whether the StreamID is a valid ULID.
// Returns false for SHA-256 derived IDs, domain-specific IDs, and empty IDs.
func IsStreamIDULID(id StreamID) bool {
	_, err := ulid.Parse(id.Get())

	return err == nil
}

// StreamTimestamp extracts the embedded timestamp from a ULID-formatted
// StreamID. Returns an error if the ID is not a valid ULID (e.g., a
// SHA-256 derived ID or a domain-specific string).
//
// For a predicate check, use IsStreamIDULID.
func StreamTimestamp(id StreamID) (time.Time, error) {
	ulidVal, err := ulid.Parse(id.Get())
	if err != nil {
		return time.Time{}, errorfamily.Wrapf(
			err,
			errorfamily.Rejection,
			"id.stream_timestamp_not_ulid",
			"StreamID %q is not a valid ULID, cannot extract timestamp",
			id.Get(),
		)
	}

	return ulid.Time(ulidVal.Time()), nil
}

// DeriveStreamID creates a deterministic StreamID from a namespace and
// one or more key strings using SHA-256. Same inputs always produce the same ID.
// Useful for stable IDs in idempotent workflows (e.g., "lock:" + userID + ":" + resourceID).
//
// The resulting ID is NOT a ULID — IsStreamIDULID returns false, and
// StreamTimestamp returns an error.
func DeriveStreamID(namespace string, keys ...string) StreamID {
	return cbid.NewID[StreamMarker](hex.EncodeToString(hashNamespacedKeys(namespace, keys...)))
}

// StreamIDFrom creates a StreamID from any fmt.Stringer.
// Useful for interop with consumer-side branded IDs that implement String().
func StreamIDFrom(s fmt.Stringer) StreamID {
	return cbid.NewID[StreamMarker](s.String())
}
