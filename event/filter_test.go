package event_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"

	ro "github.com/samber/ro"
)

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
