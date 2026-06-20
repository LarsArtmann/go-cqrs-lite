package pebble

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// writeJournalEvents writes n events to the store across multiple aggregates,
// each with a strictly increasing OccurredAt timestamp to guarantee journal
// ordering. Returns the saved events in insertion order.
func writeJournalEvents(
	t *testing.T,
	store *EventStore,
	n int,
) []event.Event {
	t.Helper()

	cfg := issueStoreConfig()
	ctx := context.Background()
	base := time.Now()

	events := make([]event.Event, n)

	for i := range n {
		aggID := id.NewAggregateID()
		evt := cfg.NewTestEvent(t, aggID, 1,
			event.WithOccurredAt(base.Add(time.Duration(i)*time.Millisecond)))
		events[i] = evt

		err := store.Save(ctx, event.NewAggregateRef("Issue", aggID),
			[]event.Event{evt}, event.Version(0))
		if err != nil {
			t.Fatalf("Save event %d: %v", i, err)
		}
	}

	return events
}

func TestEventStore_ReadFrom_NarrowedScan_Midpoint(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	events := writeJournalEvents(t, store, 100)

	// ReadFrom after events[50] with limit 10 → expect events[51:60].
	result, err := store.ReadFrom(context.Background(), events[50].ID(), 10)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(result) != 10 {
		t.Fatalf("got %d events, want 10", len(result))
	}

	for i, evt := range result {
		if evt.ID() != events[51+i].ID() {
			t.Errorf("result[%d] ID = %s, want %s", i, evt.ID(), events[51+i].ID())
		}
	}
}

func TestEventStore_ReadFrom_NarrowedScan_LastEvent(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	events := writeJournalEvents(t, store, 100)

	// ReadFrom after the last event → expect empty.
	result, err := store.ReadFrom(context.Background(), events[99].ID(), 10)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(result) != 0 {
		t.Fatalf("got %d events, want 0 (nothing after last event)", len(result))
	}
}

func TestEventStore_ReadFrom_NarrowedScan_ZeroID(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	events := writeJournalEvents(t, store, 100)

	// ReadFrom with zero ID → expect first 10 events.
	result, err := store.ReadFrom(context.Background(), id.EventID{}, 10)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(result) != 10 {
		t.Fatalf("got %d events, want 10", len(result))
	}

	for i, evt := range result {
		if evt.ID() != events[i].ID() {
			t.Errorf("result[%d] ID = %s, want %s", i, evt.ID(), events[i].ID())
		}
	}
}
