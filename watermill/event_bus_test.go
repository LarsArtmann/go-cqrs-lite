package watermill_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

func TestEventBusPublishSubscribe(t *testing.T) {
	t.Parallel()

	bus := cqrswatermill.NewEventBus()
	defer bus.Close()

	var received atomic.Int32

	err := bus.Subscribe("user.created", func(_ context.Context, _ event.Event) error {
		received.Add(1)

		return nil
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	aggID := id.NewAggregateID()
	evt, err := event.NewEvent("user.created", aggID, "User", event.Version(1),
		[]byte(`{"name":"alice"}`))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	err = bus.Publish(context.Background(), evt)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}

	waitFor(t, func() bool { return received.Load() > 0 }, 2*time.Second)
	if received.Load() != 1 {
		t.Fatalf("expected 1 event received, got %d", received.Load())
	}
}

func TestEventBusSubscribeAll(t *testing.T) {
	t.Parallel()

	bus := cqrswatermill.NewEventBus()
	defer bus.Close()

	var received atomic.Int32

	err := bus.SubscribeAll(func(_ context.Context, _ event.Event) error {
		received.Add(1)

		return nil
	})
	if err != nil {
		t.Fatalf("subscribeAll: %v", err)
	}

	aggID := id.NewAggregateID()
	for _, et := range []event.Type{"a.b", "c.d"} {
		evt, _ := event.NewEvent(et, aggID, "T", event.Version(1), nil)
		_ = bus.Publish(context.Background(), evt)
	}

	waitFor(t, func() bool { return received.Load() >= 2 }, 2*time.Second)
	if received.Load() != 2 {
		t.Fatalf("expected 2 events, got %d", received.Load())
	}
}

func TestEventBusPublishEmpty(t *testing.T) {
	t.Parallel()

	bus := cqrswatermill.NewEventBus()
	defer bus.Close()

	err := bus.Publish(context.Background())
	if err != nil {
		t.Fatalf("publish empty: %v", err)
	}
}

func TestEventBusCloseIdempotent(t *testing.T) {
	t.Parallel()

	bus := cqrswatermill.NewEventBus()

	if err := bus.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}

	if err := bus.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}

func TestEventBusPublishAfterClose(t *testing.T) {
	t.Parallel()

	bus := cqrswatermill.NewEventBus()
	_ = bus.Close()

	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent("x.y", aggID, "T", event.Version(1), nil)
	err := bus.Publish(context.Background(), evt)
	if err == nil {
		t.Fatal("expected error publishing after close")
	}
}

func TestEventBusMiddleware(t *testing.T) {
	t.Parallel()

	bus := cqrswatermill.NewEventBus()
	defer bus.Close()

	var order []string
	var mu sync.Mutex

	record := func(s string) event.Middleware {
		return func(next event.Handler) event.Handler {
			return func(ctx context.Context, evt event.Event) error {
				mu.Lock()
				order = append(order, s)
				mu.Unlock()

				return next(ctx, evt)
			}
		}
	}

	_ = bus.Use(record("outer"), record("inner"))

	var received atomic.Int32
	_ = bus.Subscribe("test.event", func(_ context.Context, _ event.Event) error {
		received.Add(1)

		return nil
	})

	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent("test.event", aggID, "T", event.Version(1), nil)
	_ = bus.Publish(context.Background(), evt)

	waitFor(t, func() bool { return received.Load() > 0 }, 2*time.Second)

	mu.Lock()
	defer mu.Unlock()
	if len(order) != 2 || order[0] != "outer" || order[1] != "inner" {
		t.Fatalf("middleware order wrong: %v", order)
	}
}

func waitFor(t *testing.T, cond func() bool, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
}
