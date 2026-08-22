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
		name string
		stamp record.Stamp
	}{
		{"known", record.NewStamp(at)},
		{"unknown", record.Stamp{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			for _, marshal := range []struct {
				name   string
				marsh  func(v any) ([]byte, error)
				unmar  func(data []byte, v any) error
			}{
				{"json v1", json.Marshal, json.Unmarshal},
				{"json v2", jsonv2.Marshal, jsonv2.Unmarshal},
			} {
				data, err := marshal.marsh(tc.stamp)
				if err != nil {
					t.Fatalf("%s Marshal: %v", marshal.name, err)
				}

				var got record.Stamp
				if err := marshal.unmar(data, &got); err != nil {
					t.Fatalf("%s Unmarshal(%s): %v", marshal.name, data, err)
				}

				if got.IsZero() != tc.stamp.IsZero() {
					t.Errorf("%s round trip changed presence: got IsZero=%v, want %v",
						marshal.name, got.IsZero(), tc.stamp.IsZero())
				}

				if !got.IsZero() && !got.Time().Equal(tc.stamp.Time()) {
					t.Errorf("%s round trip changed time: got %v, want %v",
						marshal.name, got.Time(), tc.stamp.Time())
				}
			}
		})
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
		{"known at", `{"at":"2026-08-22T05:30:01Z"}`, true},
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
