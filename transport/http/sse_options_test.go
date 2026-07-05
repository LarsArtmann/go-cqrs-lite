package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// TestSSEHandler_ByteBudget_StopsReplayEarly verifies that WithReplayByteBudget
// cuts replay short when cumulative payload bytes exceed the budget, sending an
// SSEReplayIncompleteEvent advisory.
func TestSSEHandler_ByteBudget_StopsReplayEarly(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	defer bus.Close()

	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("Big", aggID)

	// 10 events × ~50 bytes each = ~500 bytes total. Budget of 150 → ~3 events.
	const eventCount = 10
	events := make([]event.Event, 0, eventCount)

	for i := range eventCount {
		evt, err := event.NewEvent("BigEvent", aggID, "Big", 1,
			[]byte(`{"payload":"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}`))
		if err != nil {
			t.Fatalf("create evt %d: %v", i, err)
		}

		events = append(events, evt)
	}

	if err := store.Save(context.Background(), ref, events, 0); err != nil {
		t.Fatalf("save: %v", err)
	}

	broker, err := NewSSEBroker(
		bus,
		WithReconnectJournal(store, 0), // unlimited → batched streaming
		WithReplayByteBudget(150),      // very tight budget
	)
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/events?client=bytebudget", nil)
	req.Header.Set("Last-Event-ID", events[0].ID().String()) // cursor → replay events[1..]

	rec := httptest.NewRecorder()
	done := make(chan struct{})

	go func() {
		defer close(done)
		SSEHandler(broker).ServeHTTP(rec, req)
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	// Should have stopped early — NOT all 9 events replayed.
	got := strings.Count(body, "event: BigEvent")
	if got >= eventCount-1 {
		t.Errorf(
			"byte budget should have cut replay short; got %d events (expected < %d)",
			got,
			eventCount-1,
		)
	}

	if got == 0 {
		t.Errorf("expected at least one event before budget cut, got 0; body: %q", body)
	}

	if !strings.Contains(body, SSEReplayIncompleteEvent) {
		t.Errorf(
			"expected %q advisory when byte budget exceeded; body: %q",
			SSEReplayIncompleteEvent,
			body,
		)
	}
}

// TestSSEHandler_DedupRingCapacity_Custom verifies that a custom dedup ring
// capacity still correctly deduplicates replay→live overlap. Uses a tiny ring
// to confirm the option is wired through.
func TestSSEHandler_DedupRingCapacity_Custom(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	defer bus.Close()

	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("Test", aggID)

	evt0, err := event.NewEvent("TestEvent", aggID, "Test", 1, []byte(`{"seq":0}`))
	if err != nil {
		t.Fatalf("create evt0: %v", err)
	}

	evt1, err := event.NewEvent("TestEvent", aggID, "Test", 2, []byte(`{"seq":1}`))
	if err != nil {
		t.Fatalf("create evt1: %v", err)
	}

	if err := store.Save(context.Background(), ref, []event.Event{evt0, evt1}, 0); err != nil {
		t.Fatalf("save: %v", err)
	}

	broker, err := NewSSEBroker(
		bus,
		WithReconnectJournal(store, 100),
		WithDedupRingCapacity(2), // tiny ring, still holds both replayed IDs
	)
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	// Replay from evt0 → only evt1 is replayed and added to the ring.
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/events?client=tiny-ring", nil)
	req.Header.Set("Last-Event-ID", evt0.ID().String())

	rec := httptest.NewRecorder()
	done := make(chan struct{})

	go func() {
		defer close(done)
		SSEHandler(broker).ServeHTTP(rec, req)
	}()

	time.Sleep(100 * time.Millisecond)

	// Now publish evt1 live — it should be suppressed by the dedup ring.
	if err := bus.Publish(context.Background(), evt1); err != nil {
		t.Fatalf("publish evt1 live: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()
	count := strings.Count(body, `{"seq":1}`)

	// evt1 should appear exactly once (from replay, NOT again from live).
	if count != 1 {
		t.Errorf("expected evt1 delivered once (deduped), got %d; body: %q", count, body)
	}
}
