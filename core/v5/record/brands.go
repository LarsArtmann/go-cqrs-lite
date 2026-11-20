package record

import (
	"errors"
	"strconv"
)

// ErrEmptyInstanceID is returned by ParseInstanceID for an empty string.
var ErrEmptyInstanceID = errors.New("record: instance ID is required")

// ErrEmptyCorrelationID is returned by ParseCorrelationID for an empty string.
var ErrEmptyCorrelationID = errors.New("record: correlation ID is required")

// ErrNegativeVersion is returned by ParseVersion for a negative position.
// Stream positions are 1-indexed and cannot be negative; the only non-positive
// value is the zero Version, meaning "unversioned".
var ErrNegativeVersion = errors.New("record: version cannot be negative")

// InstanceID is the record instance's identifier: the EventID for events, the
// CommandID for commands, the RequestID for queries. Record is the meeting
// point of the three families, so it brands the identifier instead of
// accepting a bare string (the review R2 fix): ULID-ness stays a minting
// concern in id/; Record only promises "an identifier".
//
// The zero value means the source carried no instance ID.
type InstanceID string

// String returns the instance ID as a plain string.
func (i InstanceID) String() string { return string(i) }

// IsZero reports whether no instance ID was recorded.
func (i InstanceID) IsZero() bool { return i == "" }

// ParseInstanceID validates s and returns an InstanceID. Returns
// ErrEmptyInstanceID when s is empty — constructors are the only doors in.
func ParseInstanceID(s string) (InstanceID, error) {
	if s == "" {
		return "", ErrEmptyInstanceID
	}

	return InstanceID(s), nil
}

// Version is the 1-indexed position of a record within its stream. The first
// record in a stream has Version 1; the zero value means "unversioned" — the
// honest state for commands and queries, which are not stream members with
// positions (the review R3 fix: one unsigned vocabulary at the meeting point;
// the int64/sign split paid a conversion toll in every bridge).
type Version uint64

// IsZero reports whether the record is unversioned.
func (v Version) IsZero() bool { return v == 0 }

// UInt64 returns the underlying position. The named door for SQL and storage
// boundaries — never cast Record.Version fields directly.
func (v Version) UInt64() uint64 { return uint64(v) }

// String returns the position as a decimal string.
func (v Version) String() string { return strconv.FormatUint(uint64(v), 10) }

// ParseVersion converts a signed position (the storage/bridge vocabulary)
// into a Version, rejecting negative values with ErrNegativeVersion.
func ParseVersion(n int64) (Version, error) {
	if n < 0 {
		return 0, ErrNegativeVersion
	}

	return Version(n), nil
}

// CorrelationID links a chain of related records across the system: a user
// action that triggers a command, which emits events, which trigger sagas
// that emit more commands — all share one CorrelationID.
//
// This is the wire form of id.CorrelationID (which is ULID-backed): the
// metadata structure carries whatever the tracing chain carried, including
// foreign correlation strings, so the record brand is string-backed and
// conversion happens at the minting boundary (id/), not here.
//
// The zero value means no correlation was recorded.
type CorrelationID string

// String returns the correlation ID as a plain string.
func (c CorrelationID) String() string { return string(c) }

// IsZero reports whether no correlation ID was recorded.
func (c CorrelationID) IsZero() bool { return c == "" }

// ParseCorrelationID validates s and returns a CorrelationID. Returns
// ErrEmptyCorrelationID when s is empty.
func ParseCorrelationID(s string) (CorrelationID, error) {
	if s == "" {
		return "", ErrEmptyCorrelationID
	}

	return CorrelationID(s), nil
}
