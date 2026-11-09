package event

import (
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
)

func TestNewDate(t *testing.T) {
	tests := []struct {
		name    string
		year    int
		month   time.Month
		day     int
		wantErr bool
	}{
		{"valid date", 2024, time.March, 15, false},
		{"january 1", 2000, time.January, 1, false},
		{"december 31", 1999, time.December, 31, false},
		{"leap day", 2024, time.February, 29, false},
		{"feb 30 invalid", 2024, time.February, 30, true},
		{"feb 29 non-leap", 2023, time.February, 29, true},
		{"month too high", 2024, 13, 1, true},
		{"day too high", 2024, 1, 32, true},
		{"zero year", 0, time.January, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := NewDate(tt.year, tt.month, tt.day)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", d)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if d.Year != tt.year || d.Month != tt.month || d.Day != tt.day {
				t.Errorf("got %v, want %d-%d-%d", d, tt.year, int(tt.month), tt.day)
			}
		})
	}
}

func TestDateString(t *testing.T) {
	d := Date{Year: 2024, Month: time.March, Day: 15}
	want := "2024-03-15"
	if got := d.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestDateFromString(t *testing.T) {
	d, err := NewDateFromString("2024-03-15")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Year != 2024 || d.Month != time.March || d.Day != 15 {
		t.Errorf("got %v, want 2024-03-15", d)
	}

	if _, err := NewDateFromString("not-a-date"); err == nil {
		t.Error("expected error for invalid string")
	}
}

func TestDateComparison(t *testing.T) {
	a := Date{Year: 2024, Month: time.January, Day: 1}
	b := Date{Year: 2024, Month: time.January, Day: 2}
	c := Date{Year: 2024, Month: time.January, Day: 1}

	if !a.Before(b) {
		t.Error("Jan 1 should be before Jan 2")
	}
	if !b.After(a) {
		t.Error("Jan 2 should be after Jan 1")
	}
	if !a.Equal(c) {
		t.Error("same dates should be equal")
	}
}

func TestDateJSONRoundTrip(t *testing.T) {
	original := Date{Year: 2024, Month: time.December, Day: 25}

	data, err := original.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	var decoded Date
	if err := decoded.UnmarshalJSON(data); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}

	if !original.Equal(decoded) {
		t.Errorf("round-trip mismatch: got %v, want %v", decoded, original)
	}
}

func TestDateCBORRoundTrip(t *testing.T) {
	original := Date{Year: 2024, Month: time.July, Day: 4}

	data, err := original.MarshalCBOR()
	if err != nil {
		t.Fatalf("MarshalCBOR: %v", err)
	}

	var decoded Date
	if err := decoded.UnmarshalCBOR(data); err != nil {
		t.Fatalf("UnmarshalCBOR: %v", err)
	}

	if !original.Equal(decoded) {
		t.Errorf("round-trip mismatch: got %v, want %v", decoded, original)
	}
}

func TestDateToTime(t *testing.T) {
	d := Date{Year: 2024, Month: time.March, Day: 15}
	tm := d.ToTime()
	if tm.Year() != 2024 || tm.Month() != time.March || tm.Day() != 15 {
		t.Errorf("ToTime() = %v, want 2024-03-15", tm)
	}
	if tm.Location() != time.UTC {
		t.Errorf("ToTime() location = %v, want UTC", tm.Location())
	}
}

func TestInstantSub(t *testing.T) {
	a := NewInstantUnix(1_000_000_000) // 1 second
	b := NewInstantUnix(1_500_000_000) // 1.5 seconds
	dur := b.Sub(a)
	if dur.Milliseconds() != 500 {
		t.Errorf("Sub() = %v, want 500ms", dur)
	}
}

func TestInstantAdd(t *testing.T) {
	a := NewInstantUnix(1_000_000_000) // 1 second
	b := a.Add(500_000_000)            // +0.5 seconds
	if b.UnixNano() != 1_500_000_000 {
		t.Errorf("Add() = %v, want 1500000000", b.UnixNano())
	}
}

func TestInstantZeroConstant(t *testing.T) {
	if !Zero.IsZero() {
		t.Error("Zero should be the zero Instant")
	}
}

func TestWallTimeIsValid(t *testing.T) {
	valid := WallTime{Hour: 9, Minute: 30, Location: "America/New_York"}
	if !valid.IsValid() {
		t.Error("valid WallTime should pass IsValid")
	}

	invalidHour := WallTime{Hour: 25, Minute: 0, Location: "UTC"}
	if invalidHour.IsValid() {
		t.Error("hour 25 should fail IsValid")
	}

	invalidLoc := WallTime{Hour: 9, Minute: 0, Location: "NotARealZone"}
	if invalidLoc.IsValid() {
		t.Error("invalid location should fail IsValid")
	}

	emptyLoc := WallTime{Hour: 9, Minute: 0, Location: ""}
	if emptyLoc.IsValid() {
		t.Error("empty location should fail IsValid")
	}
}

func TestWallTimePreviousOccurrence(t *testing.T) {
	wt := WallTime{Hour: 9, Minute: 0, Location: "America/New_York"}

	// At 10am ET, the previous 9am was today.
	tenAM := time.Date(2024, 7, 15, 14, 0, 0, 0, time.UTC) // 10am EDT
	prev := wt.PreviousOccurrence(tenAM)
	if prev.Hour() != 9 || prev.Minute() != 0 {
		t.Errorf(
			"PreviousOccurrence at 10am: got %02d:%02d, want 09:00",
			prev.Hour(),
			prev.Minute(),
		)
	}

	// At 8am ET, the previous 9am was yesterday.
	eightAM := time.Date(2024, 7, 15, 12, 0, 0, 0, time.UTC) // 8am EDT
	prev = wt.PreviousOccurrence(eightAM)
	if prev.Hour() != 9 || prev.Minute() != 0 {
		t.Errorf("PreviousOccurrence at 8am: got %02d:%02d, want 09:00", prev.Hour(), prev.Minute())
	}
	// Should be yesterday
	if prev.Day() != 14 {
		t.Errorf("PreviousOccurrence at 8am: day = %d, want 14 (yesterday)", prev.Day())
	}
}

func TestWallTimeCBORRoundTrip(t *testing.T) {
	original := WallTime{Hour: 14, Minute: 30, Location: "Europe/Berlin"}

	data, err := original.MarshalCBOR()
	if err != nil {
		t.Fatalf("MarshalCBOR: %v", err)
	}

	var decoded WallTime
	if err := decoded.UnmarshalCBOR(data); err != nil {
		t.Fatalf("UnmarshalCBOR: %v", err)
	}

	if !original.Equal(decoded) {
		t.Errorf("round-trip mismatch: got %v, want %v", decoded, original)
	}
}

func TestDateCBORWithCborLib(t *testing.T) {
	original := Date{Year: 2024, Month: time.June, Day: 15}

	data, err := cbor.Marshal(original)
	if err != nil {
		t.Fatalf("cbor.Marshal: %v", err)
	}

	var decoded Date
	if err := cbor.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("cbor.Unmarshal: %v", err)
	}

	if !original.Equal(decoded) {
		t.Errorf("round-trip mismatch: got %v, want %v", decoded, original)
	}
}
