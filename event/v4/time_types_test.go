package event

import (
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
)

func TestInstant_NewInstant_NormalizesToUTC(t *testing.T) {
	t.Parallel()

	loc, _ := time.LoadLocation("America/New_York")
	input := time.Date(2026, 7, 17, 9, 30, 45, 123456789, loc)

	inst := NewInstant(input)

	if inst.Time().Location() != time.UTC {
		t.Errorf("expected UTC, got %v", inst.Time().Location())
	}

	// The instant must represent the same physical moment.
	if !inst.Time().Equal(input) {
		t.Errorf("instant changed: input=%v, got=%v", input, inst.Time())
	}
}

func TestInstant_NewInstantNow_IsUTC(t *testing.T) {
	t.Parallel()

	inst := NewInstantNow()

	if inst.Time().Location() != time.UTC {
		t.Errorf("NewInstantNow should be UTC, got %v", inst.Time().Location())
	}
}

func TestInstant_NewInstantUnix(t *testing.T) {
	t.Parallel()

	nanos := time.Date(2026, 7, 17, 14, 30, 45, 987654321, time.UTC).UnixNano()
	inst := NewInstantUnix(nanos)

	if inst.UnixNano() != nanos {
		t.Errorf("UnixNano: want %d, got %d", nanos, inst.UnixNano())
	}

	if inst.Time().Location() != time.UTC {
		t.Errorf("expected UTC, got %v", inst.Time().Location())
	}
}

func TestInstant_Equal_Before_After(t *testing.T) {
	t.Parallel()

	a := NewInstant(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	b := NewInstant(time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC))
	c := NewInstant(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	if !a.Equal(c) {
		t.Error("a should equal c")
	}

	if !a.Before(b) {
		t.Error("a should be before b")
	}

	if !b.After(a) {
		t.Error("b should be after a")
	}
}

func TestInstant_IsZero(t *testing.T) {
	t.Parallel()

	var zero Instant
	if !zero.IsZero() {
		t.Error("zero Instant should be zero")
	}

	nonZero := NewInstant(time.Now())
	if nonZero.IsZero() {
		t.Error("current time Instant should not be zero")
	}
}

func TestInstant_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	original := NewInstant(time.Date(2026, 7, 17, 14, 30, 45, 123456789, time.UTC))

	data, err := original.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	var decoded Instant
	err = decoded.UnmarshalJSON(data)
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}

	if !decoded.Equal(original) {
		t.Errorf("round-trip mismatch: original=%v decoded=%v", original, decoded)
	}

	if decoded.Time().Location() != time.UTC {
		t.Errorf("decoded should be UTC, got %v", decoded.Time().Location())
	}
}

func TestInstant_CBORRoundTrip(t *testing.T) {
	t.Parallel()

	original := NewInstant(time.Date(2026, 7, 17, 14, 30, 45, 123456789, time.UTC))

	data, err := original.MarshalCBOR()
	if err != nil {
		t.Fatalf("MarshalCBOR: %v", err)
	}

	var decoded Instant
	err = decoded.UnmarshalCBOR(data)
	if err != nil {
		t.Fatalf("UnmarshalCBOR: %v", err)
	}

	if !decoded.Equal(original) {
		t.Errorf("round-trip mismatch: original=%v decoded=%v", original, decoded)
	}

	if decoded.Time().Location() != time.UTC {
		t.Errorf("decoded should be UTC, got %v", decoded.Time().Location())
	}
}

func TestInstant_CBORInStructRoundTrip(t *testing.T) {
	t.Parallel()

	type payload struct {
		Name      string  `json:"name"`
		CreatedAt Instant `json:"created_at"`
	}

	original := payload{
		Name:      "test",
		CreatedAt: NewInstant(time.Date(2026, 7, 17, 14, 30, 45, 987654321, time.UTC)),
	}

	data, err := cbor.Marshal(original)
	if err != nil {
		t.Fatalf("cbor.Marshal: %v", err)
	}

	var decoded payload
	err = cbor.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("cbor.Unmarshal: %v", err)
	}

	if decoded.Name != original.Name {
		t.Errorf("name: want %q, got %q", original.Name, decoded.Name)
	}

	if !decoded.CreatedAt.Equal(original.CreatedAt) {
		t.Errorf("CreatedAt: want %v, got %v", original.CreatedAt, decoded.CreatedAt)
	}
}

func TestInstant_String(t *testing.T) {
	t.Parallel()

	inst := NewInstant(time.Date(2026, 7, 17, 14, 30, 45, 0, time.UTC))
	want := "2026-07-17T14:30:45Z"

	if inst.String() != want {
		t.Errorf("String(): want %q, got %q", want, inst.String())
	}
}

func TestWallTime_NewWallTime_Valid(t *testing.T) {
	t.Parallel()

	wt, err := NewWallTime(9, 30, "America/New_York")
	if err != nil {
		t.Fatalf("NewWallTime error: %v", err)
	}

	if wt.Hour != 9 || wt.Minute != 30 || wt.Location != "America/New_York" {
		t.Errorf("unexpected WallTime: %+v", wt)
	}
}

func TestWallTime_NewWallTime_InvalidHour(t *testing.T) {
	t.Parallel()

	_, err := NewWallTime(25, 0, "UTC")
	if err == nil {
		t.Fatal("expected error for hour=25")
	}
}

func TestWallTime_NewWallTime_InvalidMinute(t *testing.T) {
	t.Parallel()

	_, err := NewWallTime(9, 61, "UTC")
	if err == nil {
		t.Fatal("expected error for minute=61")
	}
}

func TestWallTime_NewWallTime_InvalidLocation(t *testing.T) {
	t.Parallel()

	_, err := NewWallTime(9, 0, "Invalid/Timezone")
	if err == nil {
		t.Fatal("expected error for invalid timezone")
	}
}

func TestWallTime_NewWallTimeMust_PanicsOnInvalid(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid WallTime")
		}
	}()

	NewWallTimeMust(25, 0, "UTC")
}

func TestWallTime_NextOccurrence_SameDay(t *testing.T) {
	t.Parallel()

	wt := NewWallTimeMust(9, 0, "America/New_York")

	// 8am ET on July 17 (summer, EDT = UTC-4)
	now := time.Date(2026, 7, 17, 8, 0, 0, 0, time.UTC)

	next := wt.NextOccurrence(now)

	// Should be 9am ET = 13:00 UTC (EDT, UTC-4)
	want := time.Date(2026, 7, 17, 13, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("NextOccurrence: want %v, got %v", want, next)
	}
}

func TestWallTime_NextOccurrence_NextDay(t *testing.T) {
	t.Parallel()

	wt := NewWallTimeMust(9, 0, "America/New_York")

	// 10am ET on July 17 (already past 9am)
	now := time.Date(2026, 7, 17, 14, 0, 0, 0, time.UTC) // 10am EDT

	next := wt.NextOccurrence(now)

	// Should be 9am ET tomorrow = 13:00 UTC on July 18
	want := time.Date(2026, 7, 18, 13, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("NextOccurrence: want %v, got %v", want, next)
	}
}

func TestWallTime_NextOccurrence_DSTTransition(t *testing.T) {
	t.Parallel()

	wt := NewWallTimeMust(9, 0, "America/New_York")

	// Before DST spring-forward (early March, EST = UTC-5)
	winter := time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC) // 3am EST
	winterNext := wt.NextOccurrence(winter)

	// 9am EST = 14:00 UTC
	winterWant := time.Date(2026, 3, 1, 14, 0, 0, 0, time.UTC)
	if !winterNext.Equal(winterWant) {
		t.Errorf("winter NextOccurrence: want %v, got %v", winterWant, winterNext)
	}

	// After DST spring-forward (July, EDT = UTC-4)
	summer := time.Date(2026, 7, 17, 8, 0, 0, 0, time.UTC) // 4am EDT
	summerNext := wt.NextOccurrence(summer)

	// 9am EDT = 13:00 UTC
	summerWant := time.Date(2026, 7, 17, 13, 0, 0, 0, time.UTC)
	if !summerNext.Equal(summerWant) {
		t.Errorf("summer NextOccurrence: want %v, got %v", summerWant, summerNext)
	}
}

func TestWallTime_Equal(t *testing.T) {
	t.Parallel()

	a := NewWallTimeMust(9, 0, "America/New_York")
	b := NewWallTimeMust(9, 0, "America/New_York")
	c := NewWallTimeMust(10, 0, "America/New_York")

	if !a.Equal(b) {
		t.Error("a should equal b")
	}

	if a.Equal(c) {
		t.Error("a should not equal c")
	}
}

func TestWallTime_String(t *testing.T) {
	t.Parallel()

	wt := NewWallTimeMust(9, 5, "America/New_York")
	if wt.String() != "09:05 America/New_York" {
		t.Errorf("String(): want %q, got %q", "09:05 America/New_York", wt.String())
	}
}

func TestWallTime_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	original := NewWallTimeMust(9, 30, "America/New_York")

	// WallTime is a plain struct with json tags — encoding/json handles it.
	// This test verifies the struct can be serialized within a payload.
	type payload struct {
		Schedule WallTime `json:"schedule"`
	}

	p := payload{Schedule: original}

	// CBOR round-trip (since payloads go through CBOR codec)
	data, err := cbor.Marshal(p)
	if err != nil {
		t.Fatalf("cbor.Marshal: %v", err)
	}

	var decoded payload
	err = cbor.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("cbor.Unmarshal: %v", err)
	}

	if !decoded.Schedule.Equal(original) {
		t.Errorf("round-trip mismatch: want %+v, got %+v", original, decoded.Schedule)
	}
}
