package event_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func TestTimeTravel_DeciderLoadAtVersion(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()

	t.Cleanup(func() { _ = store.Close() })

	bus := eventtest.NewFakeBus()

	t.Cleanup(func() { _ = bus.Close() })

	ctx := context.Background()
	streamID := id.NewStreamID()

	now := time.Now()

	evt1 := eventtest.QuickEventOpts(
		"CounterCreated", streamID, "Counter",
		1, []byte(`{"v":1}`),
		event.WithOccurredAt(now.Add(-2*time.Hour)),
	)
	evt2 := eventtest.QuickEventOpts(
		"CounterIncremented", streamID, "Counter",
		2, []byte(`{"v":2}`),
		event.WithOccurredAt(now.Add(-1*time.Hour)),
	)
	evt3 := eventtest.QuickEventOpts(
		"CounterIncremented", streamID, "Counter",
		3, []byte(`{"v":3}`),
		event.WithOccurredAt(now),
	)

	err := store.AppendBatch(
		ctx,
		id.NewStreamRef(id.StreamType("Counter"), streamID),
		[]event.Event{evt1, evt2, evt3},
	)
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	events, err := store.LoadToVersion(
		ctx,
		id.NewStreamRef(id.StreamType("Counter"), streamID),
		2,
	)
	if err != nil {
		t.Fatalf("LoadToVersion: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	eventtest.AssertEventVersion(t, events, 0, 1)
	eventtest.AssertEventVersion(t, events, 1, 2)
}

func TestTimeTravel_DeciderLoadAtTime(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()

	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	streamID := id.NewStreamID()

	now := time.Now()

	evt1 := eventtest.QuickEventOpts(
		"CounterCreated", streamID, "Counter",
		1, []byte(`{"v":1}`),
		event.WithOccurredAt(now.Add(-2*time.Hour)),
	)
	evt2 := eventtest.QuickEventOpts(
		"CounterIncremented", streamID, "Counter",
		2, []byte(`{"v":2}`),
		event.WithOccurredAt(now.Add(-30*time.Minute)),
	)
	evt3 := eventtest.QuickEventOpts(
		"CounterIncremented", streamID, "Counter",
		3, []byte(`{"v":3}`),
		event.WithOccurredAt(now),
	)

	err := store.AppendBatch(
		ctx,
		id.NewStreamRef(id.StreamType("Counter"), streamID),
		[]event.Event{evt1, evt2, evt3},
	)
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	events, err := store.LoadToTimestamp(
		ctx,
		id.NewStreamRef(id.StreamType("Counter"), streamID),
		now.Add(-15*time.Minute),
	)
	if err != nil {
		t.Fatalf("LoadToTimestamp: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestTimeTravel_SeekableJournal(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()

	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()

	aggID1 := id.NewStreamID()
	aggID2 := id.NewStreamID()

	evt1, _ := event.NewEvent("Created", aggID1, "Issue", 1, nil)
	evt2, _ := event.NewEvent("Created", aggID2, "Issue", 1, nil)
	evt3, _ := event.NewEvent("Updated", aggID1, "Issue", 2, nil)

	err := store.AppendBatch(
		ctx,
		id.NewStreamRef(id.StreamType("Issue"), aggID1),
		[]event.Event{evt1, evt3},
	)
	if err != nil {
		t.Fatalf("AppendBatch stream1: %v", err)
	}

	err = store.AppendBatch(
		ctx,
		id.NewStreamRef(id.StreamType("Issue"), aggID2),
		[]event.Event{evt2},
	)
	if err != nil {
		t.Fatalf("AppendBatch stream2: %v", err)
	}

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(all) != 3 {
		t.Fatalf("expected 3 total events, got %d", len(all))
	}

	remaining, err := store.ReadFrom(ctx, all[0].ID(), 0)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(remaining) != 2 {
		t.Fatalf("expected 2 events after position, got %d", len(remaining))
	}

	limited, err := store.ReadFrom(ctx, all[0].ID(), 1)
	if err != nil {
		t.Fatalf("ReadFrom limited: %v", err)
	}

	if len(limited) != 1 {
		t.Fatalf("expected 1 limited event, got %d", len(limited))
	}
}
