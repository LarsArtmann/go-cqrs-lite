package event_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"

	ro "github.com/samber/ro"
)

func mustNewTestEventForAggregate(t *testing.T, eventType event.Type, aggID id.AggregateID) event.Event {
	t.Helper()

	evt, err := event.NewEvent(eventType, aggID, "Test", 1, nil)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	return evt
}

func TestFilterEventType(t *testing.T) {
	t.Parallel()

	bus := event.NewEventBus()

	filtered := ro.Pipe1(bus, event.FilterEventType("user.created"))

	mu, received := subscribeAndCollect(filtered)

	bus.Next(newTestEvent(t, "user.created"))
	bus.Next(newTestEvent(t, "user.changed"))
	bus.Next(newTestEvent(t, "user.created"))
	bus.Complete()

	mu.Lock()
	defer mu.Unlock()

	if len(*received) != 2 {
		t.Fatalf("expected 2 filtered events, got %d", len(*received))
	}

	for _, e := range *received {
		if e.Type() != event.Type("user.created") {
			t.Errorf("expected type user.created, got %s", e.Type())
		}
	}
}

func TestFilterEventTypes(t *testing.T) {
	t.Parallel()

	bus := event.NewEventBus()

	filtered := ro.Pipe1(bus, event.FilterEventTypes("user.created", "user.deleted"))

	eventMu, events := subscribeAndCollect(filtered)

	bus.Next(newTestEvent(t, "user.created"))
	bus.Next(newTestEvent(t, "user.changed"))
	bus.Next(newTestEvent(t, "user.deleted"))
	bus.Next(newTestEvent(t, "user.loggedIn"))
	bus.Complete()

	eventMu.Lock()
	defer eventMu.Unlock()

	if len(*events) != 2 {
		t.Fatalf("expected 2 filtered events, got %d", len(*events))
	}

	assertEventType(t, (*events)[0], "user.created")
	assertEventType(t, (*events)[1], "user.deleted")
}

func TestDistinctByEventID(t *testing.T) {
	t.Parallel()

	bus := event.NewEventBus()

	deduped := ro.Pipe1(bus, event.DistinctByEventID())

	mu, received := subscribeAndCollect(deduped)

	// Send 3 events, then re-send event 1 and event 2 (duplicates by ID).
	evt1 := newTestEvent(t, "user.created")
	evt2 := newTestEvent(t, "user.updated")
	evt3 := newTestEvent(t, "user.deleted")

	bus.Next(evt1)
	bus.Next(evt2)
	bus.Next(evt3)
	bus.Next(evt1) // duplicate
	bus.Next(evt2) // duplicate
	bus.Complete()

	mu.Lock()
	defer mu.Unlock()

	if len(*received) != 3 {
		t.Fatalf("expected 3 deduped events, got %d", len(*received))
	}
}

func TestDistinctByAggregateID(t *testing.T) {
	t.Parallel()

	bus := event.NewEventBus()

	deduped := ro.Pipe1(bus, event.DistinctByAggregateID())

	mu, received := subscribeAndCollect(deduped)

	// Two events from same aggregate, two from another.
	// Only first per aggregate should pass.
	agg1 := id.NewAggregateID()
	agg2 := id.NewAggregateID()

	evtA1 := mustNewTestEventForAggregate(t, "user.created", agg1)
	evtA2 := mustNewTestEventForAggregate(t, "user.updated", agg1) // same aggregate, suppressed
	evtB1 := mustNewTestEventForAggregate(t, "user.created", agg2)

	bus.Next(evtA1)
	bus.Next(evtA2) // suppressed (same aggregate as evtA1)
	bus.Next(evtB1)
	bus.Complete()

	mu.Lock()
	defer mu.Unlock()

	if len(*received) != 2 {
		t.Fatalf("expected 2 deduped events (one per aggregate), got %d", len(*received))
	}
}
