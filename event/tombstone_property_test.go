package event_test

import (
	"testing"

	"pgregory.net/rapid"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func makeTombstoneEvent(t *rapid.T) event.Event {
	t.Helper()

	aggID := id.NewAggregateID()
	evt, err := event.NewEvent("test.event", aggID, "Test", event.Version(1), nil)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	return evt
}

// Property: DetectTombstone on an empty stream is always Undetermined.
func TestDetectTombstoneProperty_EmptyStream(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		status := event.DetectTombstone(nil)
		if status != event.TombstoneUndetermined {
			t.Fatalf("empty stream: got %s, want Undetermined", status)
		}

		status = event.DetectTombstone([]event.Event{})
		if status != event.TombstoneUndetermined {
			t.Fatalf("zero-length stream: got %s, want Undetermined", status)
		}
	})
}

// Property: only the LAST event's metadata matters — prefix events are irrelevant.
func TestDetectTombstoneProperty_LastEventWins(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		base := makeTombstoneEvent(t)

		// Generate the "decisive" last event with either tombstone or rebirth.
		isTombstone := rapid.Bool().Draw(t, "isTombstone")
		var lastEvent event.Event

		if isTombstone {
			marked, err := event.MarkTombstone(base)
			if err != nil {
				t.Fatalf("MarkTombstone: %v", err)
			}

			lastEvent = marked
		} else {
			marked, err := event.MarkRebirth(base)
			if err != nil {
				t.Fatalf("MarkRebirth: %v", err)
			}

			lastEvent = marked
		}

		// Build a stream with 0-10 random prefix events followed by the decisive one.
		prefixLen := rapid.IntRange(0, 10).Draw(t, "prefixLen")
		stream := make([]event.Event, 0, prefixLen+1)
		for range prefixLen {
			stream = append(stream, makeTombstoneEvent(t))
		}
		stream = append(stream, lastEvent)

		// Also build a stream with ONLY the last event.
		loneStream := []event.Event{lastEvent}

		statusFull := event.DetectTombstone(stream)
		statusLone := event.DetectTombstone(loneStream)

		if statusFull != statusLone {
			t.Fatalf("prefix events changed result: full=%s, lone=%s", statusFull, statusLone)
		}

		// Verify expected value.
		expected := event.TombstoneTombstoned
		if !isTombstone {
			expected = event.TombstoneActive
		}

		if statusFull != expected {
			t.Fatalf("got %s, want %s (isTombstone=%v)", statusFull, expected, isTombstone)
		}
	})
}

// Property: MarkTombstone/MarkRebirth never mutate the original event.
func TestTombstoneProperty_NoMutation(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		original := makeTombstoneEvent(t)
		originalStatus := event.DetectTombstone([]event.Event{original})

		// Mark it.
		marked, err := event.MarkTombstone(original)
		if err != nil {
			t.Fatalf("MarkTombstone: %v", err)
		}

		// Original must be unchanged.
		afterStatus := event.DetectTombstone([]event.Event{original})
		if afterStatus != originalStatus {
			t.Fatalf("original mutated: before=%s, after=%s", originalStatus, afterStatus)
		}

		// Marked must be different.
		markedStatus := event.DetectTombstone([]event.Event{marked})
		if markedStatus != event.TombstoneTombstoned {
			t.Fatalf("marked event: got %s, want Tombstoned", markedStatus)
		}
	})
}

// Property: tombstone then rebirth transitions correctly (and vice versa).
func TestTombstoneProperty_Transitions(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		base := makeTombstoneEvent(t)

		// Tombstone then rebirth = Active.
		tomb, err := event.MarkTombstone(base)
		if err != nil {
			t.Fatalf("MarkTombstone: %v", err)
		}

		reborn, err := event.MarkRebirth(tomb)
		if err != nil {
			t.Fatalf("MarkRebirth: %v", err)
		}

		status := event.DetectTombstone([]event.Event{base, tomb, reborn})
		if status != event.TombstoneActive {
			t.Fatalf("tombstone→rebirth: got %s, want Active", status)
		}

		// Rebirth then tombstone = Tombstoned.
		reborn2, err := event.MarkRebirth(base)
		if err != nil {
			t.Fatalf("MarkRebirth: %v", err)
		}

		tomb2, err := event.MarkTombstone(reborn2)
		if err != nil {
			t.Fatalf("MarkTombstone: %v", err)
		}

		status = event.DetectTombstone([]event.Event{base, reborn2, tomb2})
		if status != event.TombstoneTombstoned {
			t.Fatalf("rebirth→tombstone: got %s, want Tombstoned", status)
		}
	})
}

// Property: an unmarked event always yields Undetermined.
func TestDetectTombstoneProperty_UnmarkedEvent(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		evt := makeTombstoneEvent(t)
		status := event.DetectTombstone([]event.Event{evt})
		if status != event.TombstoneUndetermined {
			t.Fatalf("unmarked event: got %s, want Undetermined", status)
		}
	})
}

// Property: MarkTombstone/MarkRebirth on nil returns an error.
func TestTombstoneProperty_NilEvent(t *testing.T) {
	t.Parallel()

	_, err := event.MarkTombstone(nil)
	if err == nil {
		t.Fatal("MarkTombstone(nil): expected error")
	}

	_, err = event.MarkRebirth(nil)
	if err == nil {
		t.Fatal("MarkRebirth(nil): expected error")
	}
}
