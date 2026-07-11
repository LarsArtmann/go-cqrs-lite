package memory_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
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
	ref := id.NewAggregateRef("Bench", aggID)

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
	ref := id.NewAggregateRef("Bench", aggID)

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

	benchPopulateStore(b, store, ctx, 1000)

	b.ResetTimer()

	for b.Loop() {
		_, _ = store.ReadAll(ctx)
	}
}

func BenchmarkMemoryBus_Publish(b *testing.B) {
	b.ReportAllocs()

	bus := eventtest.NewFakeBus()
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

func BenchmarkMemoryStore_ConcurrentWriters(b *testing.B) {
	b.ReportAllocs()

	store := memory.NewMemoryStore()
	b.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()

	for _, concurrency := range []int{1, 4, 8, 16} {
		b.Run(fmt.Sprintf("writers=%d", concurrency), func(b *testing.B) {
			b.ReportAllocs()

			aggIDs := make([]id.AggregateID, concurrency)
			refs := make([]id.AggregateRef, concurrency)

			for i := range concurrency {
				aggIDs[i] = id.NewAggregateID()
				refs[i] = id.NewAggregateRef("Bench", aggIDs[i])
			}

			b.ResetTimer()

			var wg sync.WaitGroup
			wg.Add(concurrency)

			for w := range concurrency {
				go func(workerID int) {
					defer wg.Done()

					for i := range b.N {
						version := event.Version(i + 1)
						evt, _ := event.NewEvent(
							"BenchEvent",
							aggIDs[workerID],
							"Bench",
							version,
							nil,
						)
						_ = store.Save(ctx, refs[workerID], []event.Event{evt}, version-1)
					}
				}(w)
			}

			wg.Wait()
		})
	}
}

func BenchmarkMemoryBus_Publish_10Subscribers(b *testing.B) {
	b.ReportAllocs()

	bus := eventtest.NewFakeBus()
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
