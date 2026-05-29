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

func BenchmarkInMemoryList_1000Aggregates(b *testing.B) {
	store := memory.NewMemoryStore()
	ctx := context.Background()

	for range 1000 {
		aggID := id.NewAggregateID()
		payload, _ := json.Marshal(map[string]string{"name": "user"})
		evt, _ := event.NewEvent("UserCreated", aggID, "User", 1, payload)
		_ = store.AppendBatch(ctx, "User", aggID, []event.Event{evt})
	}

	reader := stream.NewInMemoryAggregateReader(store)

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
	store := memory.NewMemoryStore()
	ctx := context.Background()

	for range 100 {
		aggID := id.NewAggregateID()
		payload, _ := json.Marshal(map[string]string{"item": "widget"})
		evt, _ := event.NewEvent("OrderCreated", aggID, "Order", 1, payload)
		_ = store.AppendBatch(ctx, "Order", aggID, []event.Event{evt})
	}

	reader := stream.NewInMemoryAggregateReader(store)

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
	store := memory.NewMemoryStore()
	ctx := context.Background()

	for range 500 {
		aggID := id.NewAggregateID()
		payload, _ := json.Marshal(map[string]string{"sku": "ABC"})
		evt, _ := event.NewEvent("ItemAdded", aggID, "Cart", 1, payload)
		_ = store.AppendBatch(ctx, "Cart", aggID, []event.Event{evt})
	}

	reader := stream.NewInMemoryAggregateReader(store)

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
