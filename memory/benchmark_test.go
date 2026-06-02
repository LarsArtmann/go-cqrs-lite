package memory_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
)

func benchEvent(tb testing.TB, aggID id.AggregateID, v event.Version) event.Event {
	tb.Helper()

	evt, err := event.NewEvent("BenchEvent", aggID, "Bench", v, nil)
	if err != nil {
		tb.Fatalf("NewEvent: %v", err)
	}

	return evt
}

func BenchmarkMemoryStore_Save(b *testing.B) {
	b.ReportAllocs()

	store := memory.NewMemoryStore()
	b.Cleanup(func() { _ = store.Close() })

	aggID := id.NewAggregateID()
	ctx := context.Background()
	ref := event.NewAggregateRef("Bench", aggID)

	b.ResetTimer()

	for b.Loop() {
		evt := benchEvent(b, aggID, 1)
		_ = store.Save(ctx, ref, []event.Event{evt}, 1)
	}
}

func BenchmarkMemoryStore_Load(b *testing.B) {
	b.ReportAllocs()

	store := memory.NewMemoryStore()
	b.Cleanup(func() { _ = store.Close() })

	aggID := id.NewAggregateID()
	ctx := context.Background()
	ref := event.NewAggregateRef("Bench", aggID)

	for i := range 100 {
		evt := benchEvent(b, aggID, event.Version(i+1))
		_ = store.AppendBatch(ctx, ref, []event.Event{evt})
	}

	b.ResetTimer()

	for b.Loop() {
		_, _ = store.Load(ctx, ref)
	}
}

func BenchmarkMemoryStore_ReadAll(b *testing.B) {
	b.ReportAllocs()

	store := memory.NewMemoryStore()
	b.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()

	for range 1000 {
		aggID := id.NewAggregateID()
		evt := benchEvent(b, aggID, 1)
		_ = store.AppendBatch(ctx, event.NewAggregateRef("Bench", aggID), []event.Event{evt})
	}

	b.ResetTimer()

	for b.Loop() {
		_, _ = store.ReadAll(ctx)
	}
}

func BenchmarkMemoryBus_Publish(b *testing.B) {
	b.ReportAllocs()

	bus := memory.NewMemoryBus()
	b.Cleanup(func() { _ = bus.Close() })

	_ = bus.SubscribeAll(func(_ context.Context, _ event.Event) error { return nil })

	aggID := id.NewAggregateID()
	evt := benchEvent(b, aggID, 1)
	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		_ = bus.Publish(ctx, evt)
	}
}

func BenchmarkMemoryBus_Publish_10Subscribers(b *testing.B) {
	b.ReportAllocs()

	bus := memory.NewMemoryBus()
	b.Cleanup(func() { _ = bus.Close() })

	for range 10 {
		_ = bus.SubscribeAll(func(_ context.Context, _ event.Event) error { return nil })
	}

	aggID := id.NewAggregateID()
	evt := benchEvent(b, aggID, 1)
	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		_ = bus.Publish(ctx, evt)
	}
}
