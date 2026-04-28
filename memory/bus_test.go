package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

var errHandlerFailed = errors.New("handler failed")

var errAllHandlerFailed = errors.New("all-handler failed")

var errSubscriberFailure = errors.New("subscriber failure")

var testBusAggID = id.MustParseAggregateID("01HK154JFGAXYZMTS0FYGXF6RC") //nolint:gochecknoglobals

var testBusUserAggID = id.MustParseAggregateID( //nolint:gochecknoglobals
	"01HK1540X0841Y0A6BSX1VKR95",
)

var testBusOrderAggID = id.MustParseAggregateID( //nolint:gochecknoglobals
	"01HK1541W8PVV4E88DV993TP2A",
)

// busMiddleware returns middleware that tracks call order for event bus handlers.
func busMiddleware(callOrder *[]string, name string) func(next event.Handler) event.Handler {
	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			*callOrder = append(*callOrder, name)

			return next(ctx, evt)
		}
	}
}

func TestMemoryBus_Publish(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	ctx := context.Background()

	var received []event.Event

	err := bus.Subscribe("UserCreated", testhelpers.AppendEventsHandler(&received))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	evt, _ := event.NewEvent("UserCreated", testBusUserAggID, "User", 0, nil)

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

	bus := memory.NewMemoryBus()
	ctx := context.Background()

	var received []event.Event

	err := bus.SubscribeAll(testhelpers.AppendEventsHandler(&received))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	evt1, _ := event.NewEvent("UserCreated", testBusUserAggID, "User", 0, nil)
	evt2, _ := event.NewEvent("OrderPlaced", testBusOrderAggID, "Order", 0, nil)

	err = bus.Publish(ctx, evt1, evt2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(received) != 2 {
		t.Errorf("expected 2 events, got %d", len(received))
	}
}

func TestMemoryBus_Middleware(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	ctx := context.Background()

	var callOrder []string

	err := bus.Use(
		busMiddleware(&callOrder, "middleware1"),
		busMiddleware(&callOrder, "middleware2"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_ = bus.Subscribe("TestEvent", func(_ context.Context, _ event.Event) error {
		callOrder = append(callOrder, "handler")

		return nil
	})

	evt, _ := event.NewEvent("TestEvent", testBusAggID, "Test", 0, nil)
	_ = bus.Publish(ctx, evt)

	expected := []string{"middleware1", "middleware2", "handler"}
	if len(callOrder) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, callOrder)
	}

	for i, v := range expected {
		if callOrder[i] != v {
			t.Errorf("at index %d: expected %q, got %q", i, v, callOrder[i])
		}
	}
}

func TestMemoryBus_Closed(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
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

	evt, _ := event.NewEvent("TestEvent", testBusAggID, "Test", 0, nil)

	err = bus.Publish(context.Background(), evt)
	if err == nil {
		t.Error("expected bus closed error")
	}
}

func TestMemoryBus_HandlerError(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	ctx := context.Background()

	_ = bus.Subscribe("TestEvent", func(_ context.Context, _ event.Event) error {
		return errHandlerFailed
	})

	evt, _ := event.NewEvent("TestEvent", testBusAggID, "Test", 0, nil)

	err := bus.Publish(ctx, evt)
	if err == nil {
		t.Error("expected handler error to propagate")
	}
}

func TestMemoryBus_SubscribeAllHandlerError(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	ctx := context.Background()

	_ = bus.SubscribeAll(func(_ context.Context, _ event.Event) error {
		return errAllHandlerFailed
	})

	evt, _ := event.NewEvent("TestEvent", testBusAggID, "Test", 0, nil)

	err := bus.Publish(ctx, evt)
	if err == nil {
		t.Error("expected all-handler error to propagate")
	}
}

func TestMemoryBus_Use_Closed(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	_ = bus.Close()

	err := bus.Use(func(next event.Handler) event.Handler { return next })
	if err == nil {
		t.Error("expected bus closed error when calling Use on closed bus")
	}
}

func TestMemoryBus_Subscribe_NilHandler(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()

	err := bus.Subscribe("TestEvent", nil)
	if err == nil {
		t.Error("expected error for nil handler")
	}
}

func TestMemoryBus_SubscribeAll_NilHandler(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()

	err := bus.SubscribeAll(nil)
	if err == nil {
		t.Error("expected error for nil handler")
	}
}

func TestMemoryBus_PublishMultipleEvents_SecondFails(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	ctx := context.Background()

	_ = bus.Subscribe("FailEvent", func(_ context.Context, _ event.Event) error {
		return errSubscriberFailure
	})

	evt1, _ := event.NewEvent("OKEvent", testBusAggID, "Test", 0, nil)
	evt2, _ := event.NewEvent("FailEvent", testBusAggID, "Test", 1, nil)

	err := bus.Publish(ctx, evt1, evt2)
	if err == nil {
		t.Error("expected error when second event fails")
	}
}
