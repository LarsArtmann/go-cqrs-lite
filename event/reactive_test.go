package event_test

import (
	"context"
	"testing"

	ro "github.com/samber/ro"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/id"
)

func TestNewEventBus_PublishSubscribe(t *testing.T) {
	t.Parallel()

	bus := event.NewEventBus()

	var received event.Event
	aggID := id.NewAggregateID()

	filtered := ro.Pipe1(
		bus.AsObservable(),
		event.FilterEventType("UserCreated"),
	)

	filtered.Subscribe(ro.NewObserver[event.Event](
		func(e event.Event) { received = e },
		func(err error) {},
		func() {},
	))

	evt, _ := event.NewEvent("UserCreated", aggID, "User", 1, []byte(`{}`))
	bus.Next(evt)

	if received == nil {
		t.Fatal("expected to receive event")
	}

	if received.Type() != "UserCreated" {
		t.Errorf("type = %q, want UserCreated", received.Type())
	}
}

func TestNewEventBus_FilterMultipleTypes(t *testing.T) {
	t.Parallel()

	bus := event.NewEventBus()

	var count int

	filtered := ro.Pipe1(
		bus.AsObservable(),
		event.FilterEventTypes("UserCreated", "UserUpdated"),
	)

	filtered.Subscribe(ro.NewObserver[event.Event](
		func(e event.Event) { count++ },
		func(err error) {},
		func() {},
	))

	aggID := id.NewAggregateID()
	evt1, _ := event.NewEvent("UserCreated", aggID, "User", 1, []byte(`{}`))
	evt2, _ := event.NewEvent("UserDeleted", aggID, "User", 2, []byte(`{}`))
	evt3, _ := event.NewEvent("UserUpdated", aggID, "User", 3, []byte(`{}`))

	bus.Next(evt1)
	bus.Next(evt2)
	bus.Next(evt3)

	if count != 2 {
		t.Errorf("received %d events, want 2 (UserCreated + UserUpdated)", count)
	}
}

func TestNewEventBus_MultipleSubscribers(t *testing.T) {
	t.Parallel()

	bus := event.NewEventBus()

	var count1, count2 int

	bus.Subscribe(ro.NewObserver[event.Event](
		func(e event.Event) { count1++ },
		func(_ error) {},
		func() {},
	))

	bus.Subscribe(ro.NewObserver[event.Event](
		func(e event.Event) { count2++ },
		func(_ error) {},
		func() {},
	))

	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent("TestEvent", aggID, "Test", 1, []byte(`{}`))
	bus.Next(evt)

	if count1 != 1 || count2 != 1 {
		t.Errorf("subscribers: %d, %d — want both 1", count1, count2)
	}
}

func TestHandlerToObserver_invokes_handler(t *testing.T) {
	t.Parallel()

	var called bool
	handler := event.Handler(func(_ context.Context, _ event.Event) error {
		called = true

		return nil
	})

	observer := event.HandlerToObserver(handler)
	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent("TestEvent", aggID, "Test", 1, []byte(`{}`))
	observer.Next(evt)

	if !called {
		t.Fatal("expected handler to be called via observer")
	}
}

func TestHandlerToObserverWithContext_passes_context(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), "key", "value")

	var gotCtx context.Context
	handler := event.Handler(func(ctx context.Context, _ event.Event) error {
		gotCtx = ctx

		return nil
	})

	observer := event.HandlerToObserverWithContext(ctx, handler)
	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent("TestEvent", aggID, "Test", 1, []byte(`{}`))
	observer.Next(evt)

	if gotCtx == nil {
		t.Fatal("expected handler to receive context")
	}

	if gotCtx.Value("key") != "value" {
		t.Error("expected context value to be passed through")
	}
}
