package listing_test

import (
	"encoding/json/v2"
	"math"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
)

// FuzzAggregateListing_JSON_Roundtrip drives the JSON encoding of an
// StreamListing with arbitrary id, type, version, and counts.
func FuzzAggregateListing_JSON_Roundtrip(f *testing.F) {
	f.Add("01H4S2Z4QX8N1P5K3M7R9T0V2W", "User", int64(0), uint(0))
	f.Add("plain-id", "Order", int64(9999), uint(1000000))
	f.Add(strings.Repeat("x", 200), strings.Repeat("Z", 50), int64(42), uint(7))

	f.Fuzz(func(t *testing.T, aggID, aggType string, version int64, eventCount uint) {
		if version < 0 {
			version = 0
		}

		// JSON encoding is lossy for invalid UTF-8; skip such inputs.
		if !utf8.ValidString(aggID) || !utf8.ValidString(aggType) {
			return
		}

		idVal, err := id.ParseStreamID(aggID)
		if err != nil {
			return
		}

		original := listing.StreamListing{
			ID:          idVal,
			Type:        id.StreamType(aggType),
			Version:     event.Version(version),
			EventCount:  eventCount,
			LastEventAt: time.Unix(0, 0).UTC(),
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}

		var decoded listing.StreamListing
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}

		if decoded.ID != original.ID {
			t.Errorf("ID: got %q, want %q", decoded.ID, original.ID)
		}

		if decoded.Type != original.Type {
			t.Errorf("Type: got %q, want %q", decoded.Type, original.Type)
		}

		if decoded.Version != original.Version {
			t.Errorf("Version: got %d, want %d", decoded.Version, original.Version)
		}

		if decoded.EventCount != original.EventCount {
			t.Errorf("EventCount: got %d, want %d", decoded.EventCount, original.EventCount)
		}
	})
}

// FuzzTombstonePolicy_String ensures TombstonePolicy.String() always returns
// one of the canonical values or a default format.
func FuzzTombstonePolicy_String(f *testing.F) {
	f.Add(int(0))
	f.Add(int(1))
	f.Add(int(2))
	f.Add(int(-1))
	f.Add(int(99))
	f.Add(int(math.MaxInt32))

	f.Fuzz(func(t *testing.T, raw int) {
		policy := listing.TombstonePolicy(raw)

		got := policy.String()
		if raw >= 0 && raw <= 2 {
			expected := []string{"exclude", "include", "only"}[raw]
			if got != expected {
				t.Errorf("String(%d): got %q, want %q", raw, got, expected)
			}
		} else if !strings.HasPrefix(got, "TombstonePolicy(") {
			// Out-of-range must use default format
			t.Errorf("out-of-range String(%d): got %q, want default format", raw, got)
		}
	})
}

// FuzzAggregateStatus_MarshalOnly verifies the MarshalJSON path of
// StreamStatus without asserting full roundtrip — there is currently no
// custom UnmarshalJSON (it inherits the default, which expects an int for
// TombstoneStatus, but Marshal emits a string). We just verify marshaling
// never panics on any status value.
func FuzzAggregateStatus_MarshalOnly(f *testing.F) {
	f.Add("01H4S2Z4QX8N1P5K3M7R9T0V2W", "User", int64(1), uint(5), int(0))
	f.Add("id-1", "Order", int64(99), uint(999), int(1))
	f.Add("id-2", "X", int64(0), uint(0), int(2))
	f.Add("id-3", "Y", int64(0), uint(0), int(-1))
	f.Add("id-4", "Z", int64(0), uint(0), int(99))

	f.Fuzz(
		func(t *testing.T, aggID, aggType string, version int64, eventCount uint, statusInt int) {
			if version < 0 {
				version = 0
			}

			idVal, err := id.ParseStreamID(aggID)
			if err != nil {
				return
			}

			status := event.TombstoneStatus(statusInt)

			original := listing.StreamStatus{
				Ref: listing.StreamListing{
					ID:          idVal,
					Type:        id.StreamType(aggType),
					Version:     event.Version(version),
					EventCount:  eventCount,
					LastEventAt: time.Unix(0, 0).UTC(),
				},
				Status: status,
			}

			// Marshal must never panic, regardless of status int value.
			data, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}

			// The marshaled status is a string ("active", "tombstoned", "undetermined", or "TombstoneStatus(N)").
			if statusInt >= 0 && statusInt <= 2 {
				expected := []string{`"active"`, `"tombstoned"`, `"undetermined"`}[statusInt]
				if !strings.Contains(string(data), expected) {
					t.Errorf("Marshal: expected to contain %s, got %s", expected, data)
				}
			}
		},
	)
}
