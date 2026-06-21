package event

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

// Clock returns the current time. Override for deterministic testing.
// The default is time.Now.
type Clock func() time.Time

// DefaultClock is the clock used when no WithClock option is provided.
var defaultClock Clock = time.Now //nolint:gochecknoglobals // package-level default, intentionally mutable via tests

// Source identifies where an event originated (e.g., "api", "scheduler", "cli").
// Using a phantom type prevents accidental mixing with other string fields.
type Source string

// ParseSource validates and creates a Source from a string.
// Returns an error if the source is empty or contains invalid characters.
func ParseSource(s string) (Source, error) {
	original := s

	s = strings.TrimSpace(s)
	if s == "" {
		return "", NewRejection(
			"event.empty_source",
			fmt.Sprintf("source cannot be empty (input: %q)", original),
		)
	}

	return Source(s), nil
}

// String returns the underlying string value.
func (s Source) String() string { return string(s) }

// IsZero returns true if the source is zero-valued.
func (s Source) IsZero() bool { return s == "" }

// IPAddress represents a validated IP address.
// Using a phantom type ensures type safety and validation.
type IPAddress string

// ParseIPAddress validates and creates an IPAddress from a string.
// Returns an error if the address is not a valid IP (v4 or v6).
func ParseIPAddress(s string) (IPAddress, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil // Empty is allowed (optional field)
	}

	addr, err := netip.ParseAddr(s)
	if err != nil {
		return "", WrapRejection(
			err,
			"event.invalid_ip_address",
			"invalid IP address "+s,
		)
	}

	return IPAddress(addr.String()), nil
}

// String returns the underlying string value.
func (ip IPAddress) String() string { return string(ip) }

// IsZero returns true if the IP address is zero-valued.
func (ip IPAddress) IsZero() bool { return ip == "" }

// UserAgent represents an HTTP User-Agent string.
// Using a phantom type prevents accidental mixing with other string fields.
type UserAgent string

// NewUserAgent creates a UserAgent from a string.
// Empty user agents are allowed (optional field).
func NewUserAgent(s string) UserAgent {
	return UserAgent(strings.TrimSpace(s))
}

// String returns the underlying string value.
func (ua UserAgent) String() string { return string(ua) }

// IsZero returns true if the user agent is zero-valued.
func (ua UserAgent) IsZero() bool { return ua == "" }

// Version represents an event/aggregate version number.
// Using a phantom type ensures type safety and prevents mixing with other integers.
type Version uint64

// ParseVersion validates and creates a Version from a uint64.
func ParseVersion(v uint64) (Version, error) {
	return Version(v), nil
}

// Int returns the underlying int value.
func (v Version) Int() int { return int(v) }

// UInt64 returns the underlying uint64 value.
func (v Version) UInt64() uint64 { return uint64(v) }

// IsZero returns true if the version is zero.
func (v Version) IsZero() bool { return v == 0 }

// Increment returns a new Version incremented by 1.
func (v Version) Increment() Version { return v + 1 }

// ErrVersionUnderflow is returned when a Version operation would result in a negative value.
var ErrVersionUnderflow = errors.New("event: version underflow")

// Decrement returns a new Version decremented by 1.
// Returns ErrVersionUnderflow if v is 0.
func (v Version) Decrement() (Version, error) {
	if v == 0 {
		return 0, ErrVersionUnderflow
	}

	return v - 1, nil
}

// String returns the version as a decimal string.
func (v Version) String() string { return strconv.FormatUint(uint64(v), 10) }

// IsPositive returns true if the version is greater than zero.
func (v Version) IsPositive() bool { return v > 0 }

// Add returns a new Version incremented by n.
// Panics if n is negative (use Sub for decrements).
func (v Version) Add(n uint) Version { return v + Version(n) }

// Sub returns a new Version decremented by n.
// Returns ErrVersionUnderflow if the result would be negative.
func (v Version) Sub(n int) (Version, error) {
	if Version(n) > v {
		return 0, fmt.Errorf("%w: %d - %d", ErrVersionUnderflow, v, n)
	}

	return v - Version(n), nil
}

// Mod returns v modulo n.
func (v Version) Mod(n int) int { return int(v) % n }

// Cmp compares two Versions. Returns -1, 0, or +1.
func (v Version) Cmp(other Version) int {
	return cmp.Compare(v, other)
}

// CheckVersionConflict verifies that existingLen matches the expected version.
// Returns ErrVersionConflict if they differ, nil otherwise.
// Useful for optimistic concurrency checks in event stores.
func CheckVersionConflict(existingLen int, expected Version) error {
	if existingLen != expected.Int() {
		return WrapConflict(
			ErrVersionConflict,
			"event.version_conflict",
			"expected version "+expected.String()+", got "+strconv.Itoa(existingLen),
		)
	}

	return nil
}

// SchemaVersion represents the schema version of an event payload.
// Distinct from Version (stream position) to prevent accidental mixing.
type SchemaVersion int

// ParseSchemaVersion validates and creates a SchemaVersion from an int.
// Returns an error if the schema version is negative or zero.
func ParseSchemaVersion(v int) (SchemaVersion, error) {
	if v < 1 {
		return 0, NewRejection(
			"event.invalid_schema_version",
			fmt.Sprintf("schema version must be positive: %d", v),
		)
	}

	return SchemaVersion(v), nil
}

// ErrSchemaVersionUnderflow is returned when a SchemaVersion operation would result in a non-positive value.
var ErrSchemaVersionUnderflow = errors.New("event: schema version underflow")

// Decrement returns the previous schema version.
// Returns ErrSchemaVersionUnderflow if sv is 0 or 1 (minimum schema version is 1).
func (sv SchemaVersion) Decrement() (SchemaVersion, error) {
	if sv <= 1 {
		return 0, ErrSchemaVersionUnderflow
	}

	return sv - 1, nil
}

// Int returns the underlying int value.
func (sv SchemaVersion) Int() int { return int(sv) }

// String returns the schema version as a decimal string.
func (sv SchemaVersion) String() string { return strconv.Itoa(int(sv)) }

// IsZero returns true if the schema version is zero.
func (sv SchemaVersion) IsZero() bool { return sv == 0 }

// IsPositive returns true if the schema version is greater than zero.
func (sv SchemaVersion) IsPositive() bool { return sv > 0 }

// Increment returns the next schema version.
func (sv SchemaVersion) Increment() SchemaVersion { return sv + 1 }

// Add returns a new SchemaVersion incremented by n.
// Returns ErrSchemaVersionUnderflow if the result would be non-positive.
func (sv SchemaVersion) Add(n int) (SchemaVersion, error) {
	result := sv + SchemaVersion(n)
	if result < 1 {
		return 0, fmt.Errorf("%w: %d + %d < 1", ErrSchemaVersionUnderflow, sv, n)
	}

	return result, nil
}

// Sub returns a new SchemaVersion decremented by n.
// Returns ErrSchemaVersionUnderflow if the result would be non-positive.
func (sv SchemaVersion) Sub(n int) (SchemaVersion, error) {
	result := sv - SchemaVersion(n)
	if result < 1 {
		return 0, fmt.Errorf("%w: %d - %d < 1", ErrSchemaVersionUnderflow, sv, n)
	}

	return result, nil
}

// Cmp compares two SchemaVersions. Returns -1, 0, or +1.
func (sv SchemaVersion) Cmp(other SchemaVersion) int {
	return cmp.Compare(sv, other)
}

func (v Version) MarshalJSON() ([]byte, error) { return json.Marshal(v.Int()) }

func (v *Version) UnmarshalJSON(b []byte) error {
	var n int
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*v = Version(n)
	return nil
}

func (sv SchemaVersion) MarshalJSON() ([]byte, error) { return json.Marshal(sv.Int()) }

func (sv *SchemaVersion) UnmarshalJSON(b []byte) error {
	var n int
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*sv = SchemaVersion(n)
	return nil
}
