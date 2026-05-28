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

	var callCount int32

	bus.UsePublish(func(p event.Publisher) event.Publisher {
		return event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
			atomic.AddInt32(&callCount, 1)
			return p.Publish(ctx, events...)
		})
	})

	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent("test.event", aggID, "Test", 1, []byte(`{}`))

	err := bus.Publish(context.Background(), evt)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}

	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected 1 publish middleware call, got %d", atomic.LoadInt32(&callCount))
	}
}

func TestBus_UsePublish_Multiple(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()

	var order []int32

	bus.UsePublish(func(p event.Publisher) event.Publisher {
		return event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
			order = append(order, 1)
			return p.Publish(ctx, events...)
		})
	})

	bus.UsePublish(func(p event.Publisher) event.Publisher {
		return event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
			order = append(order, 2)
			return p.Publish(ctx, events...)
		})
	})

	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent("test.event", aggID, "Test", 1, []byte(`{}`))

	err := bus.Publish(context.Background(), evt)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}

	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Errorf("expected order [1,2] (last registered runs first), got %v", order)
	}
}
