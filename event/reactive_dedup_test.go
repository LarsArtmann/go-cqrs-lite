package event_test

import (
	"testing"

	ro "github.com/samber/ro"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func TestDistinctByEventIDBounded_DeduplicatesWithinCapacity(t *testing.T) {
	t.Parallel()

	bus := event.NewEventBus()
	deduped := ro.Pipe1(bus, event.DistinctByEventIDBounded(100))

	mu, received := subscribeAndCollect(deduped)

	evt1 := newTestEvent(t, "user.created")
	evt2 := newTestEvent(t, "user.updated")

	bus.Next(evt1)
	bus.Next(evt2)
	bus.Next(evt1) // duplicate within capacity → suppressed
	bus.Next(evt2) // duplicate within capacity → suppressed
	bus.Complete()

	mu.Lock()
	defer mu.Unlock()

	if len(*received) != 2 {
		t.Fatalf("expected 2 deduped events, got %d", len(*received))
	}
}

func TestDistinctByEventIDBounded_AllowsReprocessingAfterEviction(t *testing.T) {
	t.Parallel()

	bus := event.NewEventBus()
	deduped := ro.Pipe1(bus, event.DistinctByEventIDBounded(2))

	mu, received := subscribeAndCollect(deduped)

	evt1 := newTestEvent(t, "user.created")
	evt2 := newTestEvent(t, "user.updated")
	evt3 := newTestEvent(t, "user.deleted")

	bus.Next(evt1) // added (ring: [evt1, _])
	bus.Next(evt2) // added (ring: [evt1, evt2])
	bus.Next(evt3) // added, evicts evt1 (ring: [evt3, evt2])
	bus.Next(evt1) // evt1 evicted → passes through again
	bus.Complete()

	mu.Lock()
	defer mu.Unlock()

	if len(*received) != 4 {
		t.Fatalf("expected 4 events (evt1 evicted then reprocessed), got %d", len(*received))
	}
}

func TestDistinctByEventIDBoundedWith_SeedsFromPriorPhase(t *testing.T) {
	t.Parallel()

	bus := event.NewEventBus()

	evt1 := newTestEvent(t, "user.created")
	evt2 := newTestEvent(t, "user.updated")
	evt3 := newTestEvent(t, "user.deleted")

	seen := map[id.EventID]struct{}{
		evt1.ID(): {},
	}

	deduped := ro.Pipe1(bus, event.DistinctByEventIDBoundedWith(100, seen))

	mu, received := subscribeAndCollect(deduped)

	bus.Next(evt1) // seeded → suppressed
	bus.Next(evt2) // new → passes
	bus.Next(evt3) // new → passes
	bus.Complete()

	mu.Lock()
	defer mu.Unlock()

	if len(*received) != 2 {
		t.Fatalf("expected 2 events (evt2, evt3), got %d", len(*received))
	}
}
