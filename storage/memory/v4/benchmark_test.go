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

func benchEvent(tb testing.TB, streamID id.StreamID, v event.Version) event.Event {
	tb.Helper()

	evt, err := event.NewEvent("BenchEvent", streamID, "Bench", v, nil)
	if err != nil {
		tb.Fatalf("NewEvent: %v", err)
	}

	return evt
}

func BenchmarkMemoryStore_Save(b *testing.B) {
	b.ReportAllocs()

	store := memory.NewMemoryStore()
	b.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		// New stream per iteration so expectedVersion=0 matches the empty stream.
		streamID := id.NewStreamID()
		ref := id.NewStreamRef("Bench", streamID)
		evt := benchEvent(b, streamID, 1)
		if err := store.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
			b.Fatalf("Save: %v", err)
		}
	}
}

func BenchmarkMemoryStore_Load(b *testing.B) {
	b.ReportAllocs()

	store := memory.NewMemoryStore()
	b.Cleanup(func() { _ = store.Close() })

	streamID := id.NewStreamID()
	ctx := context.Background()
	ref := id.NewStreamRef("Bench", streamID)

	for i := range 100 {
		evt := benchEvent(b, streamID, event.Version(i+1))
		if err := store.AppendBatch(ctx, ref, []event.Event{evt}); err != nil {
			b.Fatalf("seed AppendBatch %d: %v", i, err)
		}
	}

	b.ResetTimer()

	for b.Loop() {
		events, err := store.Load(ctx, ref)
		if err != nil {
			b.Fatalf("Load: %v", err)
		}
		if len(events) == 0 {
			b.Fatal("Load returned empty — store not populated")
		}
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
		events, err := store.ReadAll(ctx)
		if err != nil {
			b.Fatalf("ReadAll: %v", err)
		}
		if len(events) == 0 {
			b.Fatal("ReadAll returned empty — store not populated")
		}
	}
}

func BenchmarkMemoryBus_Publish(b *testing.B) {
	b.ReportAllocs()

	bus := eventtest.NewFakeBus()
	b.Cleanup(func() { _ = bus.Close() })

	_ = bus.SubscribeAll(func(_ context.Context, _ event.Event) error { return nil })

	streamID := id.NewStreamID()
	evt := benchEvent(b, streamID, 1)
	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		if err := bus.Publish(ctx, evt); err != nil {
			b.Fatalf("Publish: %v", err)
		}
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

			aggIDs := make([]id.StreamID, concurrency)
			refs := make([]id.StreamRef, concurrency)

			for i := range concurrency {
				aggIDs[i] = id.NewStreamID()
				refs[i] = id.NewStreamRef("Bench", aggIDs[i])
			}

			b.ResetTimer()

			var wg sync.WaitGroup
			var firstErr error
			var errOnce sync.Once

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
						if err := store.Save(
							ctx,
							refs[workerID],
							[]event.Event{evt},
							version-1,
						); err != nil {
							errOnce.Do(func() {
								firstErr = fmt.Errorf("Save w=%d i=%d: %w", workerID, i, err)
							})
							return
						}
					}
				}(w)
			}

			wg.Wait()

			if firstErr != nil {
				b.Fatalf("%v", firstErr)
			}

			// Verify each worker's stream has data.
			for i := range concurrency {
				loaded, err := store.Load(ctx, refs[i])
				if err != nil {
					b.Fatalf("verify Load w=%d: %v", i, err)
				}
				if len(loaded) == 0 {
					b.Fatalf("verify Load w=%d: stream empty — Save was a no-op", i)
				}
			}
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

	streamID := id.NewStreamID()
	evt := benchEvent(b, streamID, 1)
	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		if err := bus.Publish(ctx, evt); err != nil {
			b.Fatalf("Publish: %v", err)
		}
	}
}
