package stream_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/stream"
)

func seedBenchAggregates(
	b *testing.B,
	aggType string,
	evtType string,
	payloadKey string,
	payloadVal string,
	n int,
) *stream.InMemoryAggregateReader {
	b.Helper()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	for range n {
		aggID := id.NewAggregateID()
		payload, _ := json.Marshal(map[string]string{payloadKey: payloadVal})
		evt, _ := event.NewEvent(event.Type(evtType), aggID, event.AggregateType(aggType), 1, payload)
		_ = store.AppendBatch(ctx, event.AggregateType(aggType), aggID, []event.Event{evt})
	}

	return stream.NewInMemoryAggregateReader(store)
}

func BenchmarkInMemoryList_1000Aggregates(b *testing.B) {
	reader := seedBenchAggregates(b, "User", "UserCreated", "name", "user", 1000)
	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		_, err := stream.NewListBuilder(reader).
			OfType("User").
			PageSize(50).
			List(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInMemoryList_100Aggregates(b *testing.B) {
	reader := seedBenchAggregates(b, "Order", "OrderCreated", "item", "widget", 100)
	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		_, err := stream.NewListBuilder(reader).
			OfType("Order").
			PageSize(10).
			List(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInMemoryList_SmallPages(b *testing.B) {
	reader := seedBenchAggregates(b, "Cart", "ItemAdded", "sku", "ABC", 500)
	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		_, err := stream.NewListBuilder(reader).
			OfType("Cart").
			PageSize(5).
			List(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInMemoryList_TombstoneFilter(b *testing.B) {
	store := memory.NewMemoryStore()
	ctx := context.Background()

	for range 500 {
		aggID := id.NewAggregateID()
		payload, _ := json.Marshal(map[string]string{"name": "doc"})
		evt, _ := event.NewEvent("DocCreated", aggID, "Doc", 1, payload)
		_ = store.AppendBatch(ctx, "Doc", aggID, []event.Event{evt})
	}

	for range 200 {
		aggID := id.NewAggregateID()
		payload, _ := json.Marshal(map[string]string{"name": "deleted"})
		evt, _ := event.NewEvent("DocCreated", aggID, "Doc", 1, payload)
		_ = store.AppendBatch(ctx, "Doc", aggID, []event.Event{evt})
		marked, _ := event.MarkTombstone(evt)
		_ = store.AppendBatch(ctx, "Doc", aggID, []event.Event{marked})
	}

	reader := stream.NewInMemoryAggregateReader(store)

	b.ResetTimer()

	for b.Loop() {
		_, err := stream.NewListBuilder(reader).
			OfType("Doc").
			PageSize(50).
			List(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
