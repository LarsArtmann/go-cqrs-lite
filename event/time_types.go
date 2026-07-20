package event

import (
	"encoding/json/v2"
	"fmt"
	"time"

	"github.com/fxamacker/cbor/v2"
)

// Instant is a UTC-normalized timestamp for use in event payloads.
//
// It wraps time.Time and enforces UTC at construction, preventing the silent
// timezone corruption that occurs when time.Time values with local timezone
// are encoded via CBOR's epoch format (which discards timezone information).
//
// Use Instant for all event payload fields that represent a unique physical
// moment ("when did this happen?"): created_at, occurred_at, updated_at, etc.
//
// For wall-clock times ("9am, for whom?"), use WallTime instead.
type Instant struct {
	t time.Time
}

// NewInstant creates an Instant from any time.Time, normalizing to UTC.
// This is the primary constructor — always use this instead of constructing
// Instant literals directly.
func NewInstant(t time.Time) Instant {
	return Instant{t: t.UTC()}
}

// NewInstantNow creates an Instant representing the current moment in UTC.
func NewInstantNow() Instant {
	return Instant{t: time.Now().UTC()}
}

// NewInstantUnix creates an Instant from Unix nanoseconds.
func NewInstantUnix(nanos int64) Instant {
	return Instant{t: time.Unix(0, nanos).UTC()}
}

// Time returns the underlying time.Time value (always UTC).
func (i Instant) Time() time.Time { return i.t }

// UnixNano returns the timestamp as Unix nanoseconds.
func (i Instant) UnixNano() int64 { return i.t.UnixNano() }

// IsZero reports whether the Instant represents the zero time (January 1, year 1).
func (i Instant) IsZero() bool { return i.t.IsZero() }

// Equal reports whether two Instants represent the same moment.
func (i Instant) Equal(other Instant) bool { return i.t.Equal(other.t) }

// Before reports whether the Instant is before another.
func (i Instant) Before(other Instant) bool { return i.t.Before(other.t) }

// After reports whether the Instant is after another.
func (i Instant) After(other Instant) bool { return i.t.After(other.t) }

// Sub returns the duration between this Instant and another.
func (i Instant) Sub(other Instant) time.Duration { return i.t.Sub(other.t) }

// Add returns a new Instant offset by the given duration, preserving UTC.
func (i Instant) Add(d time.Duration) Instant { return Instant{t: i.t.Add(d)} }

// Zero is the zero-value Instant (January 1, year 1).
// Use IsZero() to check for it rather than comparing equality.
var Zero = Instant{}

// Format returns a textual representation of the Instant formatted according
// to the layout defined by the argument.
func (i Instant) Format(layout string) string { return i.t.Format(layout) }

// String returns the Instant in RFC3339Nano format.
func (i Instant) String() string { return i.t.Format(time.RFC3339Nano) }

// MarshalJSON implements json.Marshaler, encoding as an RFC3339Nano string.
// This matches the default encoding/json behavior for time.Time.
func (i Instant) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.t.Format(time.RFC3339Nano))
}

// UnmarshalJSON implements json.Unmarshaler, parsing an RFC3339Nano string
// and normalizing to UTC.
func (i *Instant) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("instant: failed to unmarshal JSON: %w", err)
	}

	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return fmt.Errorf("instant: failed to parse time %q: %w", s, err)
	}

	i.t = t.UTC()
	return nil
}

// MarshalCBOR implements cbor.Marshaler, encoding the Instant as int64 UnixNano.
// This guarantees exact nanosecond precision regardless of the encoder's time mode.
//
// Design decision: we use bare int64 (UnixNano) instead of CBOR tag 1 (standard
// time encoding). Tag 1 encodes as float64 seconds (lossy) or requires fractional
// encoding. Our internal event storage format prioritizes exact precision over
// standard CBOR time interop. This format is never exchanged with external CBOR
// consumers.
func (i Instant) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal(i.t.UnixNano())
}

// UnmarshalCBOR implements cbor.Unmarshaler, decoding from int64 UnixNano
// and normalizing to UTC.
func (i *Instant) UnmarshalCBOR(data []byte) error {
	var nanos int64
	if err := cbor.Unmarshal(data, &nanos); err != nil {
		return fmt.Errorf("instant: failed to unmarshal CBOR: %w", err)
	}

	i.t = time.Unix(0, nanos).UTC()
	return nil
}

// WallTime represents a time of day tied to a specific timezone location.
//
// Use WallTime for event payload fields that represent a local time of day
// ("9am, for whom?"): recurring schedules, reminders, business hours, etc.
//
// Unlike time.Time, WallTime survives DST transitions correctly because it
// stores the wall time components and IANA timezone name, not an absolute
// instant. The actual instant is resolved on demand via NextOccurrence.
//
// Example:
//
//	schedule := event.NewWallTimeMust(9, 0, "America/New_York")
//	next := schedule.NextOccurrence(time.Now()) // 9am ET, DST-aware
type WallTime struct {
	Hour     int    `json:"hour"`
	Minute   int    `json:"minute"`
	Location string `json:"location"` // IANA timezone name, e.g., "America/New_York"
}

// NewWallTime creates a WallTime with validation.
// Returns an error if the hour/minute are out of range or the location is invalid.
func NewWallTime(hour, minute int, location string) (WallTime, error) {
	if hour < 0 || hour > 23 {
		return WallTime{}, fmt.Errorf("wall_time: hour %d out of range [0, 23]", hour)
	}

	if minute < 0 || minute > 59 {
		return WallTime{}, fmt.Errorf("wall_time: minute %d out of range [0, 59]", minute)
	}

	if _, err := time.LoadLocation(location); err != nil {
		return WallTime{}, fmt.Errorf("wall_time: invalid IANA timezone %q: %w", location, err)
	}

	return WallTime{Hour: hour, Minute: minute, Location: location}, nil
}

// NewWallTimeMust creates a WallTime, panicking on invalid input.
// Use only for hardcoded constants where invalid input is a programming error.
func NewWallTimeMust(hour, minute int, location string) WallTime {
	wt, err := NewWallTime(hour, minute, location)
	if err != nil {
		panic(err)
	}

	return wt
}

// loadLocationSafe returns the WallTime's IANA location, falling back to UTC
// if the location cannot be loaded. The location is validated at construction
// time (NewWallTime), so this fallback only triggers for WallTimes built
// directly from struct literals (e.g. decoded from JSON without validation).
// We fall back rather than panic so that a corrupted Location does not crash
// production callers.
func (w WallTime) loadLocationSafe() *time.Location {
	loc, err := time.LoadLocation(w.Location)
	if err != nil {
		return time.UTC
	}

	return loc
}

// NextOccurrence returns the next time this wall time occurs in its timezone,
// at or after the given reference time. If the wall time has already passed
// today in the target timezone, it returns tomorrow's occurrence.
//
// This method is DST-aware: the IANA timezone database handles offset changes
// automatically. "9am America/New_York" will be 14:00Z in winter (EST, UTC-5)
// and 13:00Z in summer (EDT, UTC-4).
func (w WallTime) NextOccurrence(after time.Time) time.Time {
	loc := w.loadLocationSafe()

	// Convert "after" to the target timezone to determine the local date.
	afterLocal := after.In(loc)

	// Construct today's occurrence.
	next := time.Date(
		afterLocal.Year(), afterLocal.Month(), afterLocal.Day(),
		w.Hour, w.Minute, 0, 0,
		loc,
	)

	// If it already passed today, advance to tomorrow.
	if !next.After(after) {
		next = next.AddDate(0, 0, 1)
	}

	return next
}

// String returns a human-readable representation of the WallTime.
func (w WallTime) String() string {
	return fmt.Sprintf("%02d:%02d %s", w.Hour, w.Minute, w.Location)
}

// Equal reports whether two WallTimes represent the same wall time
// in the same timezone.
func (w WallTime) Equal(other WallTime) bool {
	return w.Hour == other.Hour &&
		w.Minute == other.Minute &&
		w.Location == other.Location
}

// IsValid reports whether the WallTime has valid hour, minute, and location.
// Use this to check a WallTime that was not constructed via NewWallTime
// (e.g., decoded from JSON without validation).
func (w WallTime) IsValid() bool {
	if w.Hour < 0 || w.Hour > 23 {
		return false
	}
	if w.Minute < 0 || w.Minute > 59 {
		return false
	}
	if w.Location == "" {
		return false
	}
	_, err := time.LoadLocation(w.Location)
	return err == nil
}

// PreviousOccurrence returns the most recent time this wall time occurred
// in its timezone, before the given reference time. If the wall time has not
// yet occurred today in the target timezone, it returns yesterday's occurrence.
//
// This method is DST-aware, just like NextOccurrence.
func (w WallTime) PreviousOccurrence(before time.Time) time.Time {
	loc := w.loadLocationSafe()

	beforeLocal := before.In(loc)

	prev := time.Date(
		beforeLocal.Year(), beforeLocal.Month(), beforeLocal.Day(),
		w.Hour, w.Minute, 0, 0,
		loc,
	)

	if !prev.Before(before) {
		prev = prev.AddDate(0, 0, -1)
	}

	return prev
}

// MarshalCBOR implements cbor.Marshaler, encoding WallTime as a CBOR map
// with hour, minute, and location fields. This preserves the wall-clock
// semantics (no timezone information is lost).
func (w WallTime) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal(struct {
		Hour     int    `cbor:"hour"`
		Minute   int    `cbor:"minute"`
		Location string `cbor:"location"`
	}{
		Hour:     w.Hour,
		Minute:   w.Minute,
		Location: w.Location,
	})
}

// UnmarshalCBOR implements cbor.Unmarshaler, decoding from a CBOR map.
func (w *WallTime) UnmarshalCBOR(data []byte) error {
	var raw struct {
		Hour     int    `cbor:"hour"`
		Minute   int    `cbor:"minute"`
		Location string `cbor:"location"`
	}
	if err := cbor.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("wall_time: failed to unmarshal CBOR: %w", err)
	}
	w.Hour = raw.Hour
	w.Minute = raw.Minute
	w.Location = raw.Location
	return nil
}
