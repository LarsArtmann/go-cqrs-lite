package event

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

func TestSliceFromVersion(t *testing.T) {
	t.Parallel()

	events := makeSliceTestEvents(t, 5)

	tests := []struct {
		name    string
		version int
		want    int
	}{
		{"from_0", 0, 5},
		{"from_2", 2, 3},
		{"from_4", 4, 1},
		{"from_5_equals_len", 5, 0},
		{"from_10_exceeds_len", 10, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := SliceFromVersion(events, Version(tt.version))
			if len(result) != tt.want {
				t.Errorf(
					"SliceFromVersion(events, %d) = %d events, want %d",
					tt.version,
					len(result),
					tt.want,
				)
			}
		})
	}
}

func TestSliceFromVersion_Empty(t *testing.T) {
	t.Parallel()

	result := SliceFromVersion(nil, Version(0))
	if len(result) != 0 {
		t.Errorf("SliceFromVersion(nil, 0) = %d, want 0", len(result))
	}
}

func TestSliceToVersion(t *testing.T) {
	t.Parallel()

	events := makeSliceTestEvents(t, 5)

	tests := []struct {
		name    string
		version int
		want    int
	}{
		{"to_0", 0, 0},
		{"to_2", 2, 2},
		{"to_5", 5, 5},
		{"to_10_exceeds_len", 10, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := SliceToVersion(events, Version(tt.version))
			if len(result) != tt.want {
				t.Errorf(
					"SliceToVersion(events, %d) = %d events, want %d",
					tt.version,
					len(result),
					tt.want,
				)
			}
		})
	}
}

func TestFilterByTimestamp(t *testing.T) {
	t.Parallel()

	now := time.Now()
	cutoff := now
	events := []Event{
		mustNewTimestampedEvent(t, now.Add(-2*time.Hour)),
		mustNewTimestampedEvent(t, now.Add(-1*time.Hour)),
		mustNewTimestampedEvent(t, now),
		mustNewTimestampedEvent(t, now.Add(1*time.Hour)),
	}

	result := FilterByTimestamp(events, cutoff)
	if len(result) != 3 {
		t.Errorf("FilterByTimestamp with cutoff=now = %d events, want 3", len(result))
	}
}

func TestFilterByTimestamp_Empty(t *testing.T) {
	t.Parallel()

	result := FilterByTimestamp(nil, time.Now())
	if len(result) != 0 {
		t.Errorf("FilterByTimestamp(nil, _) = %d, want 0", len(result))
	}
}

func TestFilterByTimestamp_NoneMatch(t *testing.T) {
	t.Parallel()

	now := time.Now()
	events := []Event{
		mustNewTimestampedEvent(t, now.Add(1*time.Hour)),
		mustNewTimestampedEvent(t, now.Add(2*time.Hour)),
	}

	result := FilterByTimestamp(events, now.Add(-1*time.Hour))
	if len(result) != 0 {
		t.Errorf("FilterByTimestamp with past cutoff = %d, want 0", len(result))
	}
}

func TestFilterByTimestamp_AllMatch(t *testing.T) {
	t.Parallel()

	now := time.Now()
	events := []Event{
		mustNewTimestampedEvent(t, now.Add(-2*time.Hour)),
		mustNewTimestampedEvent(t, now.Add(-1*time.Hour)),
	}

	result := FilterByTimestamp(events, now.Add(1*time.Hour))
	if len(result) != 2 {
		t.Errorf("FilterByTimestamp with future cutoff = %d, want 2", len(result))
	}
}

func makeSliceTestEvents(t *testing.T, n int) []Event {
	t.Helper()

	events := make([]Event, n)
	for i := range n {
		var err error
		events[i], err = NewEvent("test.evt", id.NewAggregateID(), "Test", Version(i+1), nil)
		if err != nil {
			t.Fatalf("create event %d: %v", i, err)
		}
	}

	return events
}

func mustNewTimestampedEvent(t *testing.T, ts time.Time) Event {
	t.Helper()

	evt, err := NewEvent(
		"test.evt",
		id.NewAggregateID(),
		"Test",
		Version(1),
		nil,
		WithOccurredAt(ts),
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	return evt
}
