package event

import (
	"fmt"
	"net/netip"
	"strconv"
	"strings"
)

// Source identifies where an event originated (e.g., "api", "scheduler", "cli").
// Using a phantom type prevents accidental mixing with other string fields.
type Source string

// ParseSource validates and creates a Source from a string.
// Returns an error if the source is empty or contains invalid characters.
func ParseSource(s string) (Source, error) {
	original := s

	s = strings.TrimSpace(s)
	if s == "" {
		//nolint:err113 // dynamic error required to include original input
		return "", fmt.Errorf("source cannot be empty (input: %q)", original)
	}

	return Source(s), nil
}

// String returns the underlying string value.
func (s Source) String() string { return string(s) }

// IsEmpty returns true if the source is empty.
func (s Source) IsEmpty() bool { return s == "" }

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
		return "", fmt.Errorf("invalid IP address %q: %w", s, err)
	}

	return IPAddress(addr.String()), nil
}

// String returns the underlying string value.
func (ip IPAddress) String() string { return string(ip) }

// IsEmpty returns true if the IP address is empty.
func (ip IPAddress) IsEmpty() bool { return ip == "" }

// UserAgent represents an HTTP User-Agent string.
// Using a phantom type prevents accidental mixing with other string fields.
type UserAgent string

// ParseUserAgent creates a UserAgent from a string.
// Empty user agents are allowed (optional field).
func ParseUserAgent(s string) UserAgent {
	return UserAgent(strings.TrimSpace(s))
}

// String returns the underlying string value.
func (ua UserAgent) String() string { return string(ua) }

// IsEmpty returns true if the user agent is empty.
func (ua UserAgent) IsEmpty() bool { return ua == "" }

// Version represents an event/aggregate version number.
// Using a phantom type ensures type safety and prevents mixing with other ints.
type Version int

// ParseVersion validates and creates a Version from an int.
// Returns an error if the version is negative.
func ParseVersion(v int) (Version, error) {
	if v < 0 {
		//nolint:err113 // dynamic error required to include the invalid version number
		return 0, fmt.Errorf("version cannot be negative: %d", v)
	}

	return Version(v), nil
}

// Int returns the underlying int value.
func (v Version) Int() int { return int(v) }

// IsZero returns true if the version is zero.
func (v Version) IsZero() bool { return v == 0 }

// Increment returns a new Version incremented by 1.
func (v Version) Increment() Version { return v + 1 }

// String returns the version as a decimal string.
func (v Version) String() string { return strconv.Itoa(int(v)) }

// IsPositive returns true if the version is greater than zero.
func (v Version) IsPositive() bool { return v > 0 }

// Add returns a new Version incremented by n.
func (v Version) Add(n int) Version { return v + Version(n) }

// Sub returns a new Version decremented by n.
func (v Version) Sub(n int) Version { return v - Version(n) }

// Mod returns v modulo n.
func (v Version) Mod(n int) int { return int(v) % n }

// Cmp compares two Versions. Returns -1, 0, or +1.
func (v Version) Cmp(other Version) int {
	if v < other {
		return -1
	}

	if v > other {
		return 1
	}

	return 0
}

// SchemaVersion represents the schema version of an event payload.
// Distinct from Version (stream position) to prevent accidental mixing.
type SchemaVersion int

// ParseSchemaVersion validates and creates a SchemaVersion from an int.
// Returns an error if the schema version is negative or zero.
func ParseSchemaVersion(v int) (SchemaVersion, error) {
	if v < 1 {
		//nolint:err113 // dynamic error required to include the invalid version number
		return 0, fmt.Errorf("schema version must be positive: %d", v)
	}

	return SchemaVersion(v), nil
}

// Int returns the underlying int value.
func (sv SchemaVersion) Int() int { return int(sv) }

// String returns the schema version as a decimal string.
func (sv SchemaVersion) String() string { return strconv.Itoa(int(sv)) }

// IsZero returns true if the schema version is zero.
func (sv SchemaVersion) IsZero() bool { return sv == 0 }
