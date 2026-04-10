package event_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/event/internal/evtest"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

func BenchmarkNewEvent(b *testing.B) {
	aggregateID := id.NewAggregateID()

	for b.Loop() {
		_, _ = event.NewEvent("BenchEvent", aggregateID, "Bench", 1, nil)
	}
}

func BenchmarkMemoryBus_Publish(b *testing.B) {
	bus := event.NewMemoryBus()

	err := bus.Subscribe("BenchEvent", evtest.NoopHandler())
	if err != nil {
		b.Fatalf("subscribe: %v", err)
	}

	aggregateID := id.NewAggregateID()

	evt, err := event.NewEvent("BenchEvent", aggregateID, "Bench", 1, nil)
	if err != nil {
		b.Fatalf("new event: %v", err)
	}

	ctx := context.Background()

	for b.Loop() {
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

	for b.Loop() {
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

	for b.Loop() {
		_, _ = store.Load(ctx, "Bench", aggregateID)
	}
}
