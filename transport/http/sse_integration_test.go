package http

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// TestSSEHandler_ReplayWithRealMemoryStore runs the full replay path against
// the production storage/memory.MemoryStore instead of eventtest.FakeStore.
// The real store has index-based ReadFrom semantics and shares the single
// globalLog copy across Load/ReadAll/ReadFrom — this test catches any bug
// specific to that implementation (e.g. off-by-one on the eventID index, or
// mutation of events shared across callers).
func TestSSEHandler_ReplayWithRealMemoryStore(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	defer store.Close()

	bus := eventtest.NewFakeBus()
	defer bus.Close()

	aggID := id.NewStreamID()
	ref := id.NewStreamRef("Account", aggID)

	// Seed three events across two Save calls (version 1, then 2..3).
	evt1, err := event.NewEvent("AccountOpened", aggID, "Account", 1, []byte(`{"seq":1}`))
	if err != nil {
		t.Fatalf("create evt1: %v", err)
	}

	if err := store.Save(context.Background(), ref, []event.Event{evt1}, 0); err != nil {
		t.Fatalf("save evt1: %v", err)
	}

	evt2, err := event.NewEvent("AccountCredited", aggID, "Account", 2, []byte(`{"seq":2}`))
	if err != nil {
		t.Fatalf("create evt2: %v", err)
	}

	evt3, err := event.NewEvent("AccountDebited", aggID, "Account", 3, []byte(`{"seq":3}`))
	if err != nil {
		t.Fatalf("create evt3: %v", err)
	}

	if err := store.Save(context.Background(), ref, []event.Event{evt2, evt3}, 1); err != nil {
		t.Fatalf("save evt2,evt3: %v", err)
	}

	broker, err := NewSSEBroker(bus, WithReconnectJournal(store, 100))
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	// Reconnect with Last-Event-ID = evt1 → must replay evt2 AND evt3 only.
	rec, stop := startSSE(broker, "memory-replay", evt1.ID().String())
	time.Sleep(150 * time.Millisecond)
	stop()

	body := rec.Body.String()

	if !strings.Contains(body, `{"seq":2}`) {
		t.Errorf("expected evt2 replayed, got: %q", body)
	}

	if !strings.Contains(body, `{"seq":3}`) {
		t.Errorf("expected evt3 replayed, got: %q", body)
	}

	if strings.Contains(body, `{"seq":1}`) {
		t.Errorf("evt1 must not be replayed (it was Last-Event-ID), got: %q", body)
	}
}

// TestSSEHandler_UnlimitedReplayWithRealMemoryStore verifies the batched
// unlimited-replay path (>1 batch) against the production MemoryStore.
func TestSSEHandler_UnlimitedReplayWithRealMemoryStore(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	defer store.Close()

	bus := eventtest.NewFakeBus()
	defer bus.Close()

	aggID := id.NewStreamID()
	ref := id.NewStreamRef("Bulk", aggID)

	// Seed more events than sseReplayBatchSize to force multiple batch reads.
	const total = sseReplayBatchSize + 100
	events := make([]event.Event, 0, total)

	for range total {
		evt, err := event.NewEvent("BulkEvent", aggID, "Bulk", 1, []byte(`{}`))
		if err != nil {
			t.Fatalf("create event: %v", err)
		}

		events = append(events, evt)
	}

	if err := store.AppendBatch(context.Background(), ref, events); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	// Use unlimited replay (limit <= 0 → batched streaming).
	broker, err := NewSSEBroker(bus, WithReconnectJournal(store, 0))
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	// Last-Event-ID = first event → replays events[1..total-1] via batched streaming.
	rec, stop := startSSE(broker, "bulk-memory", events[0].ID().String())
	time.Sleep(500 * time.Millisecond)
	stop()

	body := rec.Body.String()
	got := strings.Count(body, "event: BulkEvent")

	// First event is the cursor (not replayed), so expect total-1.
	if got != total-1 {
		t.Errorf("expected %d replayed events, got %d (body len=%d)", total-1, got, len(body))
	}
}
