package event_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
)

func TestTimeTravel_DeciderLoadAtVersion(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()

	t.Cleanup(func() { _ = store.Close() })

	bus := memory.NewMemoryBus()

	t.Cleanup(func() { _ = bus.Close() })

	ctx := context.Background()
	aggID := id.NewAggregateID()

	now := time.Now()

	evt1, _ := event.NewEvent(
		"CounterCreated", aggID, "Counter",
		1, []byte(`{"v":1}`),
		event.WithOccurredAt(now.Add(-2*time.Hour)),
	)
	evt2, _ := event.NewEvent(
		"CounterIncremented", aggID, "Counter",
		2, []byte(`{"v":2}`),
		event.WithOccurredAt(now.Add(-1*time.Hour)),
	)
	evt3, _ := event.NewEvent(
		"CounterIncremented", aggID, "Counter",
		3, []byte(`{"v":3}`),
		event.WithOccurredAt(now),
	)

	err := store.AppendBatch(ctx, "Counter", aggID, []event.Event{evt1, evt2, evt3})
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	events, err := store.LoadToVersion(ctx, "Counter", aggID, 2)
	if err != nil {
		t.Fatalf("LoadToVersion: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	if events[0].Version() != 1 {
		t.Errorf("first event version = %d, want 1", events[0].Version())
	}

	if events[1].Version() != 2 {
		t.Errorf("second event version = %d, want 2", events[1].Version())
	}
}

func TestTimeTravel_DeciderLoadAtTime(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()

	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	aggID := id.NewAggregateID()

	now := time.Now()

	evt1, _ := event.NewEvent(
		"CounterCreated", aggID, "Counter",
		1, []byte(`{"v":1}`),
		event.WithOccurredAt(now.Add(-2*time.Hour)),
	)
	evt2, _ := event.NewEvent(
		"CounterIncremented", aggID, "Counter",
		2, []byte(`{"v":2}`),
		event.WithOccurredAt(now.Add(-30*time.Minute)),
	)
	evt3, _ := event.NewEvent(
		"CounterIncremented", aggID, "Counter",
		3, []byte(`{"v":3}`),
		event.WithOccurredAt(now),
	)

	err := store.AppendBatch(ctx, "Counter", aggID, []event.Event{evt1, evt2, evt3})
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	events, err := store.LoadToTimestamp(ctx, "Counter", aggID, now.Add(-15*time.Minute))
	if err != nil {
		t.Fatalf("LoadToTimestamp: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestTimeTravel_PositionalLoader(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()

	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()

	aggID1 := id.NewAggregateID()
	aggID2 := id.NewAggregateID()

	evt1, _ := event.NewEvent("Created", aggID1, "Issue", 1, nil)
	evt2, _ := event.NewEvent("Created", aggID2, "Issue", 1, nil)
	evt3, _ := event.NewEvent("Updated", aggID1, "Issue", 2, nil)

	err := store.AppendBatch(ctx, "Issue", aggID1, []event.Event{evt1, evt3})
	if err != nil {
		t.Fatalf("AppendBatch agg1: %v", err)
	}

	err = store.AppendBatch(ctx, "Issue", aggID2, []event.Event{evt2})
	if err != nil {
		t.Fatalf("AppendBatch agg2: %v", err)
	}

	all, err := store.LoadAll(ctx)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	if len(all) != 3 {
		t.Fatalf("expected 3 total events, got %d", len(all))
	}

	remaining, err := store.LoadAllFromPosition(ctx, all[0].ID(), 0)
	if err != nil {
		t.Fatalf("LoadAllFromPosition: %v", err)
	}

	if len(remaining) != 2 {
		t.Fatalf("expected 2 events after position, got %d", len(remaining))
	}

	limited, err := store.LoadAllFromPosition(ctx, all[0].ID(), 1)
	if err != nil {
		t.Fatalf("LoadAllFromPosition limited: %v", err)
	}

	if len(limited) != 1 {
		t.Fatalf("expected 1 limited event, got %d", len(limited))
	}
}
