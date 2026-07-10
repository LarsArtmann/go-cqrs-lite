package pebble

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

func TestEventStore_ReadAll_Empty(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	events, err := store.ReadAll(context.Background())
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(events) != 0 {
		t.Fatalf("got %d events, want 0", len(events))
	}
}

func TestEventStore_ReadAll_SingleAggregate(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	cfg := issueStoreConfig()
	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Issue", aggID)

	now := time.Now()
	evt1 := cfg.NewTestEvent(t, aggID, 1, event.WithOccurredAt(now))
	evt2 := cfg.NewTestEvent(t, aggID, 2, event.WithOccurredAt(now.Add(time.Nanosecond)))

	err := store.Save(context.Background(), ref, []event.Event{evt1, evt2}, event.Version(0))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	events, err := store.ReadAll(context.Background())
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}

	if events[0].ID() != evt1.ID() {
		t.Errorf("event 0 ID = %s, want %s", events[0].ID(), evt1.ID())
	}

	if events[1].ID() != evt2.ID() {
		t.Errorf("event 1 ID = %s, want %s", events[1].ID(), evt2.ID())
	}
}

func TestEventStore_ReadAll_MultipleAggregates(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	cfg := issueStoreConfig()

	aggID1 := id.NewAggregateID()
	aggID2 := id.NewAggregateID()

	evt1 := cfg.NewTestEvent(t, aggID1, 1)
	evt2 := cfg.NewTestEvent(t, aggID2, 1)

	err := store.Save(
		context.Background(),
		id.NewAggregateRef("Issue", aggID1),
		[]event.Event{evt1},
		event.Version(0),
	)
	if err != nil {
		t.Fatalf("Save agg1: %v", err)
	}

	err = store.Save(
		context.Background(),
		id.NewAggregateRef("Issue", aggID2),
		[]event.Event{evt2},
		event.Version(0),
	)
	if err != nil {
		t.Fatalf("Save agg2: %v", err)
	}

	events, err := store.ReadAll(context.Background())
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}

	ids := []id.EventID{events[0].ID(), events[1].ID()}
	if !slices.Contains(ids, evt1.ID()) || !slices.Contains(ids, evt2.ID()) {
		t.Errorf("ReadAll IDs = %v, want both %s and %s", ids, evt1.ID(), evt2.ID())
	}
}

func TestEventStore_ReadFrom_AfterFirstEvent(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	cfg := issueStoreConfig()
	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Issue", aggID)

	now := time.Now()
	evt1 := cfg.NewTestEvent(t, aggID, 1, event.WithOccurredAt(now))
	evt2 := cfg.NewTestEvent(t, aggID, 2, event.WithOccurredAt(now.Add(time.Nanosecond)))
	evt3 := cfg.NewTestEvent(t, aggID, 3, event.WithOccurredAt(now.Add(2*time.Nanosecond)))

	err := store.Save(context.Background(), ref, []event.Event{evt1, evt2, evt3}, event.Version(0))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	events, err := store.ReadFrom(context.Background(), evt1.ID(), 0)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}

	if events[0].ID() != evt2.ID() {
		t.Errorf("event 0 ID = %s, want %s", events[0].ID(), evt2.ID())
	}

	if events[1].ID() != evt3.ID() {
		t.Errorf("event 1 ID = %s, want %s", events[1].ID(), evt3.ID())
	}
}

func TestEventStore_ReadFrom_WithLimit(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	cfg := issueStoreConfig()
	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Issue", aggID)

	now := time.Now()
	evt1 := cfg.NewTestEvent(t, aggID, 1, event.WithOccurredAt(now))
	evt2 := cfg.NewTestEvent(t, aggID, 2, event.WithOccurredAt(now.Add(time.Nanosecond)))
	evt3 := cfg.NewTestEvent(t, aggID, 3, event.WithOccurredAt(now.Add(2*time.Nanosecond)))

	err := store.Save(context.Background(), ref, []event.Event{evt1, evt2, evt3}, event.Version(0))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	events, err := store.ReadFrom(context.Background(), evt1.ID(), 1)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1 (limit)", len(events))
	}

	if events[0].ID() != evt2.ID() {
		t.Errorf("event 0 ID = %s, want %s", events[0].ID(), evt2.ID())
	}
}

func TestEventStore_ReadFrom_ZeroEventID(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	cfg := issueStoreConfig()
	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Issue", aggID)

	evt := cfg.NewTestEvent(t, aggID, 1)

	err := store.Save(context.Background(), ref, []event.Event{evt}, event.Version(0))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	events, err := store.ReadFrom(context.Background(), id.EventID{}, 0)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
}

func TestEventStore_ReadFrom_UnknownEventID(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	cfg := issueStoreConfig()
	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Issue", aggID)

	evt := cfg.NewTestEvent(t, aggID, 1)

	err := store.Save(context.Background(), ref, []event.Event{evt}, event.Version(0))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	unknownID := id.NewEventID()
	events, err := store.ReadFrom(context.Background(), unknownID, 0)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(events) != 0 {
		t.Fatalf("got %d events, want 0 (unknown afterEventID)", len(events))
	}
}

func TestEventStore_Journal_AppendBatch(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	cfg := issueStoreConfig()
	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Issue", aggID)

	evt := cfg.NewTestEvent(t, aggID, 1)

	err := store.AppendBatch(context.Background(), ref, []event.Event{evt})
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	events, err := store.ReadAll(context.Background())
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}

	if events[0].ID() != evt.ID() {
		t.Errorf("event ID = %s, want %s", events[0].ID(), evt.ID())
	}
}
