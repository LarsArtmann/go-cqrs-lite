package event_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"

	ro "github.com/samber/ro"
)

func newTestEvent(t *testing.T, eventType event.Type) event.Event {
	t.Helper()

	aggID := id.NewAggregateID()
	evt, err := event.NewEvent(eventType, aggID, "Test", 1, nil)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	return evt
}

func subscribeAndCollect(src ro.Observable[event.Event]) (*sync.Mutex, *[]event.Event) {
	var mu sync.Mutex
	var received []event.Event

	src.Subscribe(ro.OnNext(func(e event.Event) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	}))

	return &mu, &received
}

func assertEventType(t *testing.T, evt event.Event, want string) {
	t.Helper()
	if evt.Type() != event.Type(want) {
		t.Errorf("expected %s, got %s", want, evt.Type())
	}
}

type errTestHandler string

func (e errTestHandler) Error() string { return string(e) }

func TestNewEventBus_PublishSubscribe(t *testing.T) {
	t.Parallel()

	bus := event.NewEventBus()

	eventMu, events := subscribeAndCollect(bus)

	evt := newTestEvent(t, "user.created")
	bus.Next(evt)
	bus.Complete()

	eventMu.Lock()
	defer eventMu.Unlock()

	if len(*events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(*events))
	}

	assertEventType(t, (*events)[0], "user.created")
}

func TestNewEventBus_MultipleSubscribers(t *testing.T) {
	t.Parallel()

	bus := event.NewEventBus()

	var mu sync.Mutex
	count1, count2 := 0, 0

	bus.Subscribe(ro.OnNext(func(e event.Event) {
		mu.Lock()
		count1++
		mu.Unlock()
	}))

	bus.Subscribe(ro.OnNext(func(e event.Event) {
		mu.Lock()
		count2++
		mu.Unlock()
	}))

	bus.Next(newTestEvent(t, "user.created"))
	bus.Next(newTestEvent(t, "user.changed"))
	bus.Complete()

	mu.Lock()
	defer mu.Unlock()

	if count1 != 2 {
		t.Errorf("subscriber 1 expected 2 events, got %d", count1)
	}

	if count2 != 2 {
		t.Errorf("subscriber 2 expected 2 events, got %d", count2)
	}
}

func TestNewReplayEventBus(t *testing.T) {
	t.Parallel()

	bus := event.NewReplayEventBus(3)

	bus.Next(newTestEvent(t, "user.created"))
	bus.Next(newTestEvent(t, "user.changed"))
	bus.Next(newTestEvent(t, "user.deleted"))

	var received []event.Event

	bus.Subscribe(ro.OnNext(func(e event.Event) {
		received = append(received, e)
	}))
	bus.Complete()

	if len(received) != 3 {
		t.Fatalf("expected 3 replayed events, got %d", len(received))
	}

	types := make([]event.Type, len(received))
	for i, e := range received {
		types[i] = e.Type()
	}

	expected := []event.Type{"user.created", "user.changed", "user.deleted"}
	for i, exp := range expected {
		if types[i] != exp {
			t.Errorf("event %d: expected %s, got %s", i, exp, types[i])
		}
	}
}

func TestNewReplayEventBus_LateSubscriber(t *testing.T) {
	t.Parallel()

	bus := event.NewReplayEventBus(2)

	bus.Next(newTestEvent(t, "one"))
	bus.Next(newTestEvent(t, "two"))
	bus.Next(newTestEvent(t, "three"))

	var received []event.Event

	bus.Subscribe(ro.OnNext(func(e event.Event) {
		received = append(received, e)
	}))
	bus.Complete()

	if len(received) != 2 {
		t.Fatalf("expected 2 replayed events (buffer=2), got %d", len(received))
	}

	if received[0].Type() != event.Type("two") {
		t.Errorf("expected 'two' first, got %s", received[0].Type())
	}

	if received[1].Type() != event.Type("three") {
		t.Errorf("expected 'three' second, got %s", received[1].Type())
	}
}

func TestNewBehaviorEventBus(t *testing.T) {
	t.Parallel()

	initial := newTestEvent(t, "initial")
	bus := event.NewBehaviorEventBus(initial)

	var received event.Event

	bus.Subscribe(ro.OnNext(func(e event.Event) {
		received = e
	}))

	if received == nil {
		t.Fatal("expected immediate replay of initial value")
	}

	if received.Type() != event.Type("initial") {
		t.Errorf("expected initial, got %s", received.Type())
	}

	bus.Next(newTestEvent(t, "updated"))
	bus.Complete()

	if received.Type() != event.Type("updated") {
		t.Errorf("expected updated, got %s", received.Type())
	}
}

func TestHandlerToObserver_UsesEventContext(t *testing.T) {
	t.Parallel()

	var gotDeadline time.Time
	var hasDeadline bool

	deadline := time.Now().Add(30 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	handler := func(ctx context.Context, _ event.Event) error {
		gotDeadline, hasDeadline = ctx.Deadline()
		return nil
	}

	observer := event.HandlerToObserver(handler)

	evt, err := event.New(
		"ctx.test",
		id.NewAggregateID(),
		"TestAggregate",
		1,
		[]byte(`{}`),
		event.FromContext(ctx),
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	observer.NextWithContext(ctx, evt)

	if !hasDeadline {
		t.Fatal("expected deadline from event context")
	}

	if !gotDeadline.Equal(deadline) {
		t.Errorf("expected deadline %v, got %v", deadline, gotDeadline)
	}
}
