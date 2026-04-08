package event_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

func BenchmarkNewEvent(b *testing.B) {
	aggregateID := id.NewAggregateID()

	b.ResetTimer()

	for range b.N {
		_, _ = event.NewEvent("BenchEvent", aggregateID, "Bench", 1, nil)
	}
}

func BenchmarkMemoryBus_Publish(b *testing.B) {
	bus := event.NewMemoryBus()

	err := bus.Subscribe("BenchEvent", func(_ context.Context, _ event.Event) error {
		return nil
	})
	if err != nil {
		b.Fatalf("subscribe: %v", err)
	}

	aggregateID := id.NewAggregateID()
	evt, err := event.NewEvent("BenchEvent", aggregateID, "Bench", 1, nil)
	if err != nil {
		b.Fatalf("new event: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()

	for range b.N {
		err := bus.Publish(ctx, evt)
		if err != nil {
			b.Fatalf("publish: %v", err)
		}
	}
}

func BenchmarkMemoryStore_Save(b *testing.B) {
	store := event.NewMemoryStore()
	aggregateID := id.NewAggregateID()
	ctx := context.Background()

	b.ResetTimer()

	for range b.N {
		evt, _ := event.NewEvent("BenchEvent", aggregateID, "Bench", 1, nil)
		_ = store.Save(ctx, "Bench", aggregateID, []event.Event{evt}, 1)
	}
}

func BenchmarkMemoryStore_Load(b *testing.B) {
	store := event.NewMemoryStore()
	aggregateID := id.NewAggregateID()
	ctx := context.Background()

	for i := range 10 {
		evt, _ := event.NewEvent("BenchEvent", aggregateID, "Bench", 1, nil)
		_ = store.Save(ctx, "Bench", aggregateID, []event.Event{evt}, event.Version(i+1))
	}

	b.ResetTimer()

	for range b.N {
		_, _ = store.Load(ctx, "Bench", aggregateID)
	}
}
