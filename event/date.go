package event

import (
	"encoding/json/v2"
	"fmt"
	"time"

	"github.com/fxamacker/cbor/v2"
)

// Date represents a calendar date (year, month, day) without a time or timezone.
//
// Use Date for event payload fields that represent a calendar date with no
// specific time of day: birth dates, employment start dates, contract dates, etc.
//
// Unlike Instant (which represents a specific physical moment), Date is
// timezone-agnostic. "2024-03-15" means the same calendar day regardless of
// the observer's timezone. This prevents the common bug where a birth date
// stored as time.Time shifts by a day when viewed from a different timezone.
//
// The string format is RFC 3339 date-only: "2006-01-02".
type Date struct {
	Year  int
	Month time.Month
	Day   int
}

// NewDate creates a Date from year, month, and day components.
// Returns an error if the date does not exist (e.g., February 30).
func NewDate(year int, month time.Month, day int) (Date, error) {
	// Validate by constructing a time.Time and checking round-trip.
	t := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	if t.Year() != year || t.Month() != month || t.Day() != day {
		return Date{}, fmt.Errorf("%w: %04d-%02d-%02d", ErrInvalidDate, year, int(month), day)
	}
	return Date{Year: year, Month: month, Day: day}, nil
}

// NewDateMust creates a Date, panicking on invalid input.
// Use only for hardcoded constants.
func NewDateMust(year int, month time.Month, day int) Date {
	d, err := NewDate(year, month, day)
	if err != nil {
		panic(err)
	}
	return d
}

// NewDateFromString parses a date in RFC 3339 date format ("2006-01-02").
func NewDateFromString(s string) (Date, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return Date{}, fmt.Errorf("date: failed to parse %q: %w", s, err)
	}
	return Date{Year: t.Year(), Month: t.Month(), Day: t.Day()}, nil
}

// String returns the date in "2006-01-02" format.
func (d Date) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, int(d.Month), d.Day)
}

// Equal reports whether two Dates represent the same calendar day.
func (d Date) Equal(other Date) bool {
	return d.Year == other.Year && d.Month == other.Month && d.Day == other.Day
}

// Before reports whether the Date is before another chronologically.
func (d Date) Before(other Date) bool {
	if d.Year != other.Year {
		return d.Year < other.Year
	}
	if d.Month != other.Month {
		return d.Month < other.Month
	}
	return d.Day < other.Day
}

// After reports whether the Date is after another chronologically.
func (d Date) After(other Date) bool { return other.Before(d) }

// IsZero reports whether the Date is the zero value.
func (d Date) IsZero() bool { return d.Year == 0 && d.Month == 0 && d.Day == 0 }

// ToTime returns the Date as a time.Time at midnight UTC.
func (d Date) ToTime() time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC)
}

// MarshalJSON implements json.Marshaler, encoding as a "2006-01-02" string.
func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// parseDateInto parses s and stores the result in d. Used by both UnmarshalJSON
// and UnmarshalCBOR so the two decoders stay in lockstep — without this helper,
// adding a new validation rule would require remembering to touch both.
func (d *Date) parseDateInto(s string) error {
	parsed, err := NewDateFromString(s)
	if err != nil {
		return err
	}

	*d = parsed

	return nil
}

// UnmarshalJSON implements json.Unmarshaler, parsing a "2006-01-02" string.
func (d *Date) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("date: failed to unmarshal JSON: %w", err)
	}

	return d.parseDateInto(s)
}

// MarshalCBOR implements cbor.Marshaler, encoding as a "2006-01-02" string.
func (d Date) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal(d.String())
}

// UnmarshalCBOR implements cbor.Unmarshaler, decoding from a "2006-01-02" string.
func (d *Date) UnmarshalCBOR(data []byte) error {
	var s string
	if err := cbor.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("date: failed to unmarshal CBOR: %w", err)
	}

	return d.parseDateInto(s)
}
