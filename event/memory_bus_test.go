package event_test

import (
	"context"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/event/internal/evtest"
	"github.com/larsartmann/go-cqrs-lite/internal/testhelpers"
)

func TestMemoryBus_Publish(t *testing.T) {
	t.Parallel()

	bus := event.NewMemoryBus()
	ctx := context.Background()

	var received []event.Event

	handler := evtest.CollectingHandler(t, &received)

	err := bus.Subscribe("UserCreated", handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	evt, _ := event.NewEvent("UserCreated", "user-1", "User", 0, nil)

	err = bus.Publish(ctx, evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(received) != 1 {
		t.Errorf("expected 1 event, got %d", len(received))
	}
}

func TestMemoryBus_SubscribeAll(t *testing.T) {
	t.Parallel()

	bus := event.NewMemoryBus()
	ctx := context.Background()

	var received []event.Event

	handler := evtest.CollectingHandler(t, &received)

	err := bus.SubscribeAll(handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	evt1, _ := event.NewEvent("UserCreated", "user-1", "User", 0, nil)
	evt2, _ := event.NewEvent("OrderPlaced", "order-1", "Order", 0, nil)

	err = bus.Publish(ctx, evt1, evt2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(received) != 2 {
		t.Errorf("expected 2 events, got %d", len(received))
	}
}

// testMiddleware creates a middleware that records its name in callOrder.
func testMiddleware(callOrder *[]string, name string) func(h event.Handler) event.Handler {
	return func(h event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			*callOrder = append(*callOrder, name)

			return h(ctx, evt)
		}
	}
}

func TestMemoryBus_Middleware(t *testing.T) {
	t.Parallel()

	bus := event.NewMemoryBus()
	ctx := context.Background()

	var callOrder []string

	bus.Use(
		testMiddleware(&callOrder, "middleware1"),
		testMiddleware(&callOrder, "middleware2"),
	)

	_ = bus.Subscribe("TestEvent", func(_ context.Context, _ event.Event) error {
		callOrder = append(callOrder, "handler")

		return nil
	})

	evt, _ := event.NewEvent("TestEvent", "test-1", "Test", 0, nil)
	_ = bus.Publish(ctx, evt)

	testhelpers.AssertCallOrder(t, callOrder, []string{"middleware1", "middleware2", "handler"})
}

func TestMemoryBus_Closed(t *testing.T) {
	t.Parallel()

	bus := event.NewMemoryBus()
	_ = bus.Close()

	handler := func(_ context.Context, _ event.Event) error { return nil }

	err := bus.Subscribe("TestEvent", handler)
	if err == nil {
		t.Error("expected bus closed error")
	}

	err = bus.SubscribeAll(handler)
	if err == nil {
		t.Error("expected bus closed error")
	}

	evt, _ := event.NewEvent("TestEvent", "test-1", "Test", 0, nil)

	err = bus.Publish(context.Background(), evt)
	if err == nil {
		t.Error("expected bus closed error")
	}
}

func TestMemoryBus_HandlerError(t *testing.T) {
	t.Parallel()

	bus := event.NewMemoryBus()
	ctx := context.Background()

	_ = bus.Subscribe("TestEvent", testErrorHandler("handler failed"))

	evt, _ := event.NewEvent("TestEvent", "test-1", "Test", 0, nil)

	err := bus.Publish(ctx, evt)
	if err == nil {
		t.Error("expected handler error to propagate")
	}
}

func TestMemoryBus_SubscribeAllHandlerError(t *testing.T) {
	t.Parallel()

	bus := event.NewMemoryBus()
	ctx := context.Background()

	_ = bus.SubscribeAll(testErrorHandler("all-handler failed"))

	evt, _ := event.NewEvent("TestEvent", "test-1", "Test", 0, nil)

	err := bus.Publish(ctx, evt)
	if err == nil {
		t.Error("expected all-handler error to propagate")
	}
}

func testErrorHandler(msg string) event.Handler {
	return func(_ context.Context, _ event.Event) error {
		return errors.New(msg)
	}
}

func TestMemoryBus_PublishMultipleEvents_SecondFails(t *testing.T) {
	t.Parallel()

	bus := event.NewMemoryBus()
	ctx := context.Background()

	callCount := 0
	failingHandler := func(ctx context.Context, evt event.Event) error {
		_ = ctx
		_ = evt
		return errors.New("subscriber failure")
	}
	_ = bus.Subscribe("FailEvent", failingHandler)

	evt1, _ := event.NewEvent("OKEvent", "test-1", "Test", 0, nil)
	evt2, _ := event.NewEvent("FailEvent", "test-1", "Test", 1, nil)

	err := bus.Publish(ctx, evt1, evt2)
	if err == nil {
		t.Error("expected error when second event fails")
	}

	_ = callCount
}
