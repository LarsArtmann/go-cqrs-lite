package event_test

import (
	"context"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

func TestHandlerToObserver_Success(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var received []event.Event

	handler := func(ctx context.Context, e event.Event) error {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()

		return nil
	}

	observer := event.HandlerToObserver(handler)

	observer.Next(newTestEvent(t, "user.created"))

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 1 {
		t.Fatalf("expected 1 event, got %d", len(received))
	}

	if observer.HasThrown() {
		t.Fatal("expected observer to not have thrown")
	}
}

func TestHandlerToObserver_ErrorTerminatesObserver(t *testing.T) {
	t.Parallel()

	handlerErr := errTestHandler("handler failed")

	handler := func(_ context.Context, _ event.Event) error {
		return handlerErr
	}

	observer := event.HandlerToObserver(handler)

	observer.Next(newTestEvent(t, "user.created"))

	if !observer.HasThrown() {
		t.Fatal("expected observer to have thrown after handler error")
	}

	if !observer.IsClosed() {
		t.Fatal("expected observer to be closed after error")
	}
}

func TestHandlerToObserver_SubsequentEventsDroppedAfterError(t *testing.T) {
	t.Parallel()

	var callCount int

	handler := func(_ context.Context, _ event.Event) error {
		callCount++
		if callCount > 1 {
			return errTestHandler("should not be called")
		}

		return errTestHandler("first error")
	}

	observer := event.HandlerToObserver(handler)

	observer.Next(newTestEvent(t, "first"))
	observer.Next(newTestEvent(t, "second"))
	observer.Next(newTestEvent(t, "third"))

	if callCount != 1 {
		t.Fatalf("expected 1 handler call, got %d (subsequent events should be dropped)", callCount)
	}
}

func TestHandlerToObserver_UsesStreamContext(t *testing.T) {
	t.Parallel()

	key := struct{}{}
	streamCtx := context.WithValue(context.Background(), key, "from-stream")

	var gotCtx context.Context

	handler := func(ctx context.Context, _ event.Event) error {
		gotCtx = ctx
		return nil
	}

	observer := event.HandlerToObserver(handler)

	observer.NextWithContext(streamCtx, newTestEvent(t, "ctx.test"))

	if gotCtx == nil {
		t.Fatal("expected context from stream")
	}

	if gotCtx.Value(key) != "from-stream" {
		t.Error("expected stream context, not event or background context")
	}
}

func TestHandlerToObserverWithContext_OverridesStreamContext(t *testing.T) {
	t.Parallel()

	key := struct{}{}
	overrideCtx := context.WithValue(context.Background(), key, "override")

	var gotCtx context.Context

	handler := func(ctx context.Context, _ event.Event) error {
		gotCtx = ctx
		return nil
	}

	observer := event.HandlerToObserverWithContext(overrideCtx, handler)

	streamCtx := context.WithValue(context.Background(), key, "stream")
	observer.NextWithContext(streamCtx, newTestEvent(t, "ctx.test"))

	if gotCtx == nil {
		t.Fatal("expected context")
	}

	if gotCtx.Value(key) != "override" {
		t.Error("expected override context, not stream context")
	}
}

func TestHandlerToObserverWithContext_ErrorTerminatesObserver(t *testing.T) {
	t.Parallel()

	handlerErr := errTestHandler("ctx handler failed")

	handler := func(_ context.Context, _ event.Event) error {
		return handlerErr
	}

	observer := event.HandlerToObserverWithContext(context.Background(), handler)

	observer.Next(newTestEvent(t, "user.created"))

	if !observer.HasThrown() {
		t.Fatal("expected observer to have thrown after handler error")
	}
}
