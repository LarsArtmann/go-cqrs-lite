package event_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v3"
)

func BenchmarkNewEvent(b *testing.B) {
	b.ReportAllocs()
	aggregateID := id.NewAggregateID()

	for b.Loop() {
		_, _ = event.NewEvent("BenchEvent", aggregateID, "Bench", 1, nil)
	}
}

func BenchmarkMemoryBus_Publish(b *testing.B) {
	b.ReportAllocs()
	bus := eventtest.NewFakeBus()

	err := bus.Subscribe("BenchEvent", eventtest.NoopEventHandler())
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
	b.ReportAllocs()
	store := memory.NewMemoryStore()
	aggregateID := id.NewAggregateID()
	ctx := context.Background()

	for b.Loop() {
		evt, _ := event.NewEvent("BenchEvent", aggregateID, "Bench", 1, nil)
		_ = store.Save(
			ctx,
			event.NewAggregateRef(event.AggregateType("Bench"), aggregateID),
			[]event.Event{evt},
			1,
		)
	}
}

func BenchmarkMemoryStore_Load(b *testing.B) {
	b.ReportAllocs()
	store := memory.NewMemoryStore()
	aggregateID := id.NewAggregateID()
	ctx := context.Background()

	for i := range 10 {
		evt, _ := event.NewEvent("BenchEvent", aggregateID, "Bench", 1, nil)
		_ = store.Save(
			ctx,
			event.NewAggregateRef(event.AggregateType("Bench"), aggregateID),
			[]event.Event{evt},
			event.Version(i+1),
		)
	}

	for b.Loop() {
		_, _ = store.Load(ctx, event.NewAggregateRef(event.AggregateType("Bench"), aggregateID))
	}
}
