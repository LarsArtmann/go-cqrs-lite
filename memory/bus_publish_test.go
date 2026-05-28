package memory_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
)

func TestBus_UsePublish(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()

	var callCount atomic.Int32

	err := bus.UsePublish(func(p event.Publisher) event.Publisher {
		return event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
			callCount.Add(1)

			return p.Publish(ctx, events...)
		})
	})
	if err != nil {
		t.Fatalf("UsePublish: %v", err)
	}

	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent("test.event", aggID, "Test", 1, []byte(`{}`))

	err = bus.Publish(context.Background(), evt)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}

	if callCount.Load() != 1 {
		t.Errorf("expected 1 publish middleware call, got %d", callCount.Load())
	}
}

func TestBus_UsePublish_Multiple(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()

	var order []int32

	appendOrderMW := func(n int32) event.PublishMiddleware {
		return func(p event.Publisher) event.Publisher {
			return event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
				order = append(order, n)

				return p.Publish(ctx, events...)
			})
		}
	}

	if err := bus.UsePublish(appendOrderMW(1)); err != nil {
		t.Fatalf("UsePublish 1: %v", err)
	}
	if err := bus.UsePublish(appendOrderMW(2)); err != nil {
		t.Fatalf("UsePublish 2: %v", err)
	}

	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent("test.event", aggID, "Test", 1, []byte(`{}`))

	if err := bus.Publish(context.Background(), evt); err != nil {
		t.Fatalf("publish: %v", err)
	}

	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Errorf("expected order [1,2] (last registered runs first), got %v", order)
	}
}
