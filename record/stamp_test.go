package record_test

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

func TestStamp_ZeroValueIsUnknown(t *testing.T) {
	t.Parallel()

	var s record.Stamp
	if !s.IsZero() {
		t.Error("zero Stamp must IsZero() (unknown, not epoch)")
	}

	if !s.Time().IsZero() {
		t.Errorf("zero Stamp Time() = %v, want zero time", s.Time())
	}

	if s.String() != "unknown" {
		t.Errorf("zero Stamp String() = %q, want %q", s.String(), "unknown")
	}
}

func TestNewStamp(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 8, 22, 5, 30, 0, 0, time.UTC)
	s := record.NewStamp(at)

	if s.IsZero() {
		t.Error("NewStamp must be known")
	}

	if !s.Time().Equal(at) {
		t.Errorf("Time() = %v, want %v", s.Time(), at)
	}

	if s.String() != at.Format(time.RFC3339Nano) {
		t.Errorf("String() = %q, want %q", s.String(), at.Format(time.RFC3339Nano))
	}
}

func TestStamp_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 8, 22, 5, 30, 1, 500, time.UTC)

	cases := []struct {
		name  string
		stamp record.Stamp
	}{
		{"known", record.NewStamp(at)},
		{"known zero time", record.NewStamp(time.Time{})},
		{"unknown", record.Stamp{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// encoding/json v1
			data, err := json.Marshal(tc.stamp)
			if err != nil {
				t.Fatalf("v1 Marshal: %v", err)
			}

			var got record.Stamp
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("v1 Unmarshal(%s): %v", data, err)
			}

			assertStampRoundTrip(t, "v1", got, tc.stamp)

			// encoding/json v2 (goexperiment.jsonv2)
			data2, err := jsonv2.Marshal(tc.stamp)
			if err != nil {
				t.Fatalf("v2 Marshal: %v", err)
			}

			var got2 record.Stamp
			if err := jsonv2.Unmarshal(data2, &got2); err != nil {
				t.Fatalf("v2 Unmarshal(%s): %v", data2, err)
			}

			assertStampRoundTrip(t, "v2", got2, tc.stamp)
		})
	}
}

func assertStampRoundTrip(t *testing.T, flavor string, got, want record.Stamp) {
	t.Helper()

	if got.IsZero() != want.IsZero() {
		t.Errorf("%s round trip changed presence: got IsZero=%v, want %v",
			flavor, got.IsZero(), want.IsZero())
	}

	if !got.IsZero() && !got.Time().Equal(want.Time()) {
		t.Errorf("%s round trip changed time: got %v, want %v",
			flavor, got.Time(), want.Time())
	}
}

func TestStamp_UnmarshalForms(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		data  string
		known bool
	}{
		{"null", `null`, false},
		{"empty object", `{}`, false},
		{"null at", `{"at":null}`, false},
		{"known at", `{"at":"2026-08-22T05:30:01Z"}`, true},
		{"known zero at", `{"at":"0001-01-01T00:00:00Z"}`, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var s record.Stamp
			if err := json.Unmarshal([]byte(tc.data), &s); err != nil {
				t.Fatalf("Unmarshal(%s): %v", tc.data, err)
			}

			if s.IsZero() == tc.known {
				t.Errorf("Unmarshal(%s) IsZero() = %v, want known=%v",
					tc.data, s.IsZero(), tc.known)
			}
		})
	}
}
