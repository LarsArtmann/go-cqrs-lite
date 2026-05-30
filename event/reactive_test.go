package event_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/id"

	ro "github.com/samber/ro"
)

func newTestEvent(t *testing.T, eventType event.Type) event.Event {
	t.Helper()

	evt, err := event.New(
		eventType,
		id.NewAggregateID(),
		"TestAggregate",
		1,
		[]byte(`{"test":true}`),
	)
	if err != nil {
		t.Fatalf("create test event: %v", err)
	}

	return evt
}

func TestNewEventBus_PublishSubscribe(t *testing.T) {
	t.Parallel()

	bus := event.NewEventBus()

	var mu sync.Mutex
	var received []event.Event

	bus.Subscribe(ro.OnNext(func(e event.Event) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	}))

	evt := newTestEvent(t, "user.created")
	bus.Next(evt)
	bus.Complete()

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 1 {
		t.Fatalf("expected 1 event, got %d", len(received))
	}

	if received[0].Type() != event.Type("user.created") {
		t.Errorf("expected type user.created, got %s", received[0].Type())
	}
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

func TestFilterEventType(t *testing.T) {
	t.Parallel()

	bus := event.NewEventBus()

	filtered := ro.Pipe1(bus, event.FilterEventType("user.created"))

	var mu sync.Mutex
	var received []event.Event

	filtered.Subscribe(ro.OnNext(func(e event.Event) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	}))

	bus.Next(newTestEvent(t, "user.created"))
	bus.Next(newTestEvent(t, "user.changed"))
	bus.Next(newTestEvent(t, "user.created"))
	bus.Complete()

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 2 {
		t.Fatalf("expected 2 filtered events, got %d", len(received))
	}

	for _, e := range received {
		if e.Type() != event.Type("user.created") {
			t.Errorf("expected type user.created, got %s", e.Type())
		}
	}
}

func TestFilterEventTypes(t *testing.T) {
	t.Parallel()

	bus := event.NewEventBus()

	filtered := ro.Pipe1(bus, event.FilterEventTypes("user.created", "user.deleted"))

	var mu sync.Mutex
	var received []event.Event

	filtered.Subscribe(ro.OnNext(func(e event.Event) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	}))

	bus.Next(newTestEvent(t, "user.created"))
	bus.Next(newTestEvent(t, "user.changed"))
	bus.Next(newTestEvent(t, "user.deleted"))
	bus.Next(newTestEvent(t, "user.loggedIn"))
	bus.Complete()

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 2 {
		t.Fatalf("expected 2 filtered events, got %d", len(received))
	}

	if received[0].Type() != event.Type("user.created") {
		t.Errorf("expected user.created, got %s", received[0].Type())
	}

	if received[1].Type() != event.Type("user.deleted") {
		t.Errorf("expected user.deleted, got %s", received[1].Type())
	}
}

func TestHandlerToObserver_Success(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var received []event.Event
	var errors []error

	handler := func(ctx context.Context, e event.Event) error {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()

		return nil
	}

	observer := event.HandlerToObserver(handler, func(err error) {
		mu.Lock()
		errors = append(errors, err)
		mu.Unlock()
	})

	observer.Next(newTestEvent(t, "user.created"))

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 1 {
		t.Fatalf("expected 1 event, got %d", len(received))
	}

	if len(errors) != 0 {
		t.Fatalf("expected 0 errors, got %d", len(errors))
	}
}

func TestHandlerToObserver_ErrorPropagation(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var errors []error

	handlerErr := errTestHandler("handler failed")

	handler := func(_ context.Context, _ event.Event) error {
		return handlerErr
	}

	observer := event.HandlerToObserver(handler, func(err error) {
		mu.Lock()
		errors = append(errors, err)
		mu.Unlock()
	})

	observer.Next(newTestEvent(t, "user.created"))

	mu.Lock()
	defer mu.Unlock()

	if len(errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errors))
	}

	if errors[0] != handlerErr {
		t.Errorf("expected handlerErr, got %v", errors[0])
	}
}

func TestHandlerToObserver_StreamError(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var errors []error

	handler := func(_ context.Context, _ event.Event) error {
		return nil
	}

	observer := event.HandlerToObserver(handler, func(err error) {
		mu.Lock()
		errors = append(errors, err)
		mu.Unlock()
	})

	streamErr := errTestHandler("stream broken")
	observer.Error(streamErr)

	mu.Lock()
	defer mu.Unlock()

	if len(errors) != 1 {
		t.Fatalf("expected 1 stream error, got %d", len(errors))
	}

	if errors[0] != streamErr {
		t.Errorf("expected streamErr, got %v", errors[0])
	}
}

func TestHandlerToObserver_UsesEventContext(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var gotDeadline time.Time
	var hasDeadline bool

	deadline := time.Now().Add(30 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	handler := func(ctx context.Context, _ event.Event) error {
		mu.Lock()
		gotDeadline, hasDeadline = ctx.Deadline()
		mu.Unlock()

		return nil
	}

	observer := event.HandlerToObserver(handler, func(_ error) {})

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

	observer.Next(evt)

	mu.Lock()
	defer mu.Unlock()

	if !hasDeadline {
		t.Fatal("expected deadline from event context, got none")
	}

	if !gotDeadline.Equal(deadline) {
		t.Errorf("expected deadline %v from event.Context(), got %v", deadline, gotDeadline)
	}
}

func TestHandlerToObserverWithContext_OverridesContext(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var gotCtx context.Context

	overrideKey := struct{}{}
	overrideCtx := context.WithValue(context.Background(), overrideKey, "override")

	handler := func(ctx context.Context, _ event.Event) error {
		mu.Lock()
		gotCtx = ctx
		mu.Unlock()

		return nil
	}

	observer := event.HandlerToObserverWithContext(overrideCtx, handler, func(_ error) {})

	observer.Next(newTestEvent(t, "ctx.test"))

	mu.Lock()
	defer mu.Unlock()

	if gotCtx == nil {
		t.Fatal("expected context, got nil")
	}

	if gotCtx.Value(overrideKey) != "override" {
		t.Error("expected override context, not event context")
	}
}

func TestHandlerToObserverWithContext_ErrorPropagation(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var errors []error

	handlerErr := errTestHandler("ctx handler failed")

	handler := func(_ context.Context, _ event.Event) error {
		return handlerErr
	}

	observer := event.HandlerToObserverWithContext(context.Background(), handler, func(err error) {
		mu.Lock()
		errors = append(errors, err)
		mu.Unlock()
	})

	observer.Next(newTestEvent(t, "user.created"))

	mu.Lock()
	defer mu.Unlock()

	if len(errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errors))
	}

	if errors[0] != handlerErr {
		t.Errorf("expected handlerErr, got %v", errors[0])
	}
}

type errTestHandler string

func (e errTestHandler) Error() string { return string(e) }
