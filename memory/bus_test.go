package memory_test

import (
	"context"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
)

func parseAggID(s string) id.AggregateID {
	v, err := id.ParseAggregateID(s)
	if err != nil {
		panic(err)
	}

	return v
}

var testBusAggID = parseAggID("01HK154JFGAXYZMTS0FYGXF6RC")

var testBusUserAggID = parseAggID(
	"01HK1540X0841Y0A6BSX1VKR95",
)

var testBusOrderAggID = parseAggID(
	"01HK1541W8PVV4E88DV993TP2A",
)

func TestMemoryBus_Publish(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	ctx := context.Background()

	var received []event.Event

	err := bus.Subscribe("UserCreated", eventtest.AppendEventsHandler(&received))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	evt := eventtest.QuickEvent("UserCreated", testBusUserAggID, "User", 1, nil)

	err = bus.Publish(ctx, evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	eventtest.AssertLen(t, "events", received, 1)
}

func TestMemoryBus_SubscribeAll(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	ctx := context.Background()

	var received []event.Event

	err := bus.SubscribeAll(eventtest.AppendEventsHandler(&received))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	evt1 := eventtest.QuickEvent("UserCreated", testBusUserAggID, "User", 1, nil)
	evt2 := eventtest.QuickEvent("OrderPlaced", testBusOrderAggID, "Order", 1, nil)

	err = bus.Publish(ctx, evt1, evt2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	eventtest.AssertLen(t, "events", received, 2)
}

func TestMemoryBus_Middleware(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	ctx := context.Background()

	var callOrder []string

	err := bus.Use(
		eventtest.EventMiddleware(&callOrder, "middleware1"),
		eventtest.EventMiddleware(&callOrder, "middleware2"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_ = bus.Subscribe("TestEvent", func(_ context.Context, _ event.Event) error {
		callOrder = append(callOrder, "handler")

		return nil
	})

	evt := eventtest.QuickEvent("TestEvent", testBusAggID, "Test", 1, nil)
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

	handler := eventtest.NoopEventHandler()

	err := bus.Subscribe("TestEvent", handler)
	if err == nil {
		t.Error("expected bus closed error")
	}

	err = bus.SubscribeAll(handler)
	if err == nil {
		t.Error("expected bus closed error")
	}

	evt := eventtest.QuickEvent("TestEvent", testBusAggID, "Test", 1, nil)

	err = bus.Publish(context.Background(), evt)
	if err == nil {
		t.Error("expected bus closed error")
	}
}

func TestMemoryBus_HandlerError(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	ctx := context.Background()

	_ = bus.Subscribe("TestEvent", eventtest.FailingEventHandler("handler failed"))

	evt := eventtest.QuickEvent("TestEvent", testBusAggID, "Test", 1, nil)

	err := bus.Publish(ctx, evt)
	if err == nil {
		t.Error("expected handler error to propagate")
	}
}

func TestMemoryBus_SubscribeAllHandlerError(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	ctx := context.Background()

	_ = bus.SubscribeAll(eventtest.FailingEventHandler("all-handler failed"))

	evt := eventtest.QuickEvent("TestEvent", testBusAggID, "Test", 1, nil)

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

	_ = bus.Subscribe("FailEvent", eventtest.FailingEventHandler("subscriber failure"))

	evt1 := eventtest.QuickEvent("OKEvent", testBusAggID, "Test", 1, nil)
	evt2 := eventtest.QuickEvent("FailEvent", testBusAggID, "Test", 2, nil)

	err := bus.Publish(ctx, evt1, evt2)
	if err == nil {
		t.Error("expected error when second event fails")
	}
}

func TestMemoryBus_ConcurrentPublishAndSubscribe(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	ctx := context.Background()

	var received sync.Mutex

	var events []string

	handler := func(_ context.Context, evt event.Event) error {
		received.Lock()
		events = append(events, string(evt.Type()))
		received.Unlock()

		return nil
	}

	err := bus.SubscribeAll(handler)
	if err != nil {
		t.Fatalf("SubscribeAll: %v", err)
	}

	const publishers = 5
	const eventsPerPublisher = 20

	var wg sync.WaitGroup
	wg.Add(publishers)

	for i := range publishers {
		go func() {
			defer wg.Done()

			for evtIdx := range eventsPerPublisher {
				aggID := id.NewAggregateID()
				evt := eventtest.QuickEvent(
					"ConcurrentEvent",
					aggID,
					"Test",
					event.Version(evtIdx+1),
					nil,
				)
				_ = bus.Publish(ctx, evt)
			}
		}()

		_ = i
	}

	wg.Wait()

	received.Lock()
	count := len(events)
	received.Unlock()

	expected := publishers * eventsPerPublisher
	if count != expected {
		t.Errorf("received %d events, want %d", count, expected)
	}
}
