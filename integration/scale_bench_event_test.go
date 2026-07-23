package integration_test

import (
	"context"
	"encoding/json/v2"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func BenchmarkScale_EventCreation_WithPayload(b *testing.B) {
	b.ReportAllocs()

	aggID := id.NewStreamID()
	payload, err := json.Marshal(map[string]string{"name": "test-item", "sku": "ABC-123"})
	if err != nil {
		b.Fatalf("json.Marshal: %v", err)
	}

	b.ResetTimer()

	for b.Loop() {
		_, _ = event.NewEvent("ItemCreated", aggID, "Item", 1, payload)
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/sec")
}

func BenchmarkScale_EventSave_10KAggregates_100EventsEach(b *testing.B) {
	b.ReportAllocs()

	store := memory.NewMemoryStore()
	b.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	aggCount := 10_000
	eventsPerAgg := 100

	aggIDs := make([]id.StreamID, aggCount)
	for i := range aggIDs {
		aggIDs[i] = id.NewStreamID()
	}

	b.ResetTimer()

	for b.Loop() {
		for _, aggID := range aggIDs {
			events := make([]event.Event, eventsPerAgg)

			for v := range eventsPerAgg {
				events[v] = newBenchEvent(b, "ItemUpdated", aggID, event.Version(v+1))
			}

			err := store.AppendBatch(
				ctx,
				id.NewStreamRef("Item", aggID),
				events,
			)
			if err != nil {
				b.Fatalf("AppendBatch: %v", err)
			}
		}
	}

	totalEvents := int64(aggCount) * int64(eventsPerAgg)
	b.ReportMetric(float64(b.N*int(totalEvents))/b.Elapsed().Seconds(), "events/sec")
}

func BenchmarkScale_EventPublish_MemoryBus_100KEvents(b *testing.B) {
	b.ReportAllocs()

	bus := eventtest.NewFakeBus()
	b.Cleanup(func() { _ = bus.Close() })

	err := bus.SubscribeAll(noopEventHandler())
	if err != nil {
		b.Fatalf("SubscribeAll: %v", err)
	}

	aggID := id.NewStreamID()
	events := make([]event.Event, 100)
	for i := range events {
		events[i] = newBenchEvent(b, "ItemUpdated", aggID, event.Version(i+1))
	}

	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		err := bus.Publish(ctx, events...)
		if err != nil {
			b.Fatalf("Publish: %v", err)
		}
	}

	b.ReportMetric(float64(b.N*100)/b.Elapsed().Seconds(), "events/sec")
}
