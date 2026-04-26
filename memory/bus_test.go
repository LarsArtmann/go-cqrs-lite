package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
)

var errHandlerFailed = errors.New("handler failed")

var errAllHandlerFailed = errors.New("all-handler failed")

var errSubscriberFailure = errors.New("subscriber failure")

func TestMemoryBus_Publish(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	ctx := context.Background()

	var received []event.Event

	handler := func(_ context.Context, evt event.Event) error {
		received = append(received, evt)

		return nil
	}

	err := bus.Subscribe("UserCreated", handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	evt, _ := event.NewEvent("UserCreated", id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"), "User", 0, nil)

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

	handler := func(_ context.Context, evt event.Event) error {
		received = append(received, evt)

		return nil
	}

	err := bus.SubscribeAll(handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	evt1, _ := event.NewEvent("UserCreated", id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"), "User", 0, nil)
	evt2, _ := event.NewEvent("OrderPlaced", id.MustParseAggregateID("01HK1541W8PVV4E88DV993TP2A"), "Order", 0, nil)

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

	bus.Use(
		func(next event.Handler) event.Handler {
			return func(ctx context.Context, evt event.Event) error {
				callOrder = append(callOrder, "middleware1")

				return next(ctx, evt)
			}
		},
		func(next event.Handler) event.Handler {
			return func(ctx context.Context, evt event.Event) error {
				callOrder = append(callOrder, "middleware2")

				return next(ctx, evt)
			}
		},
	)

	_ = bus.Subscribe("TestEvent", func(_ context.Context, _ event.Event) error {
		callOrder = append(callOrder, "handler")

		return nil
	})

	evt, _ := event.NewEvent("TestEvent", id.MustParseAggregateID("01HK154JFGAXYZMTS0FYGXF6RC"), "Test", 0, nil)
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

	evt, _ := event.NewEvent("TestEvent", id.MustParseAggregateID("01HK154JFGAXYZMTS0FYGXF6RC"), "Test", 0, nil)

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

	evt, _ := event.NewEvent("TestEvent", id.MustParseAggregateID("01HK154JFGAXYZMTS0FYGXF6RC"), "Test", 0, nil)

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

	evt, _ := event.NewEvent("TestEvent", id.MustParseAggregateID("01HK154JFGAXYZMTS0FYGXF6RC"), "Test", 0, nil)

	err := bus.Publish(ctx, evt)
	if err == nil {
		t.Error("expected all-handler error to propagate")
	}
}

func TestMemoryBus_PublishMultipleEvents_SecondFails(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	ctx := context.Background()

	_ = bus.Subscribe("FailEvent", func(_ context.Context, _ event.Event) error {
		return errSubscriberFailure
	})

	evt1, _ := event.NewEvent("OKEvent", id.MustParseAggregateID("01HK154JFGAXYZMTS0FYGXF6RC"), "Test", 0, nil)
	evt2, _ := event.NewEvent("FailEvent", id.MustParseAggregateID("01HK154JFGAXYZMTS0FYGXF6RC"), "Test", 1, nil)

	err := bus.Publish(ctx, evt1, evt2)
	if err == nil {
		t.Error("expected error when second event fails")
	}
}
