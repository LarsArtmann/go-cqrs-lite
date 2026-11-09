package bench

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// BenchmarkContention_SameStream measures the impact of concurrent writes
// to the SAME stream. The event store serializes writes per-stream (optimistic
// concurrency), so this benchmark reveals the contention ceiling.
//
// Compare with BenchmarkCommandPath_Concurrent (different streams) to see
// the overhead of write serialization.
func BenchmarkContention_SameStream(b *testing.B) {
	bundle, err := stack.New(
		stack.WithEventStore(memory.NewMemoryStore()),
	)
	if err != nil {
		b.Fatal(err)
	}

	defer func() { _ = bundle.Close() }()

	store, ok := bundle.EventStore()
	if !ok {
		b.Fatal("bundle has no event store")
	}

	ctx := context.Background()
	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Counter", streamID)

	var version atomic.Int64

	b.ResetTimer()

	for b.Loop() {
		v := event.Version(version.Add(1))

		evt, err := event.NewEvent(
			"counter.incremented", streamID, "Counter", v,
			[]byte(`{"amount":1}`),
		)
		if err != nil {
			b.Fatal(err)
		}

		if err := store.AppendBatch(ctx, ref, []event.Event{evt}); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/sec")
}

// BenchmarkContention_SameStream_Concurrent measures contention when multiple
// goroutines write to the same stream simultaneously. The memory store
// serializes via mutex; persistent stores use optimistic concurrency control.
func BenchmarkContention_SameStream_Concurrent(b *testing.B) {
	for _, concurrency := range []int{1, 2, 4, 8} {
		b.Run(fmt.Sprintf("workers=%d", concurrency), func(b *testing.B) {
			bundle, err := stack.New(
				stack.WithEventStore(memory.NewMemoryStore()),
			)
			if err != nil {
				b.Fatal(err)
			}

			defer func() { _ = bundle.Close() }()

			store, ok := bundle.EventStore()
			if !ok {
				b.Fatal("bundle has no event store")
			}

			ctx := context.Background()
			streamID := id.NewStreamID()
			ref := id.NewStreamRef("Counter", streamID)

			var version atomic.Int64
			var wg sync.WaitGroup
			wg.Add(concurrency)

			b.ResetTimer()

			for range concurrency {
				go func() {
					defer wg.Done()

					for range b.N / concurrency {
						v := event.Version(version.Add(1))

						evt, err := event.NewEvent(
							"counter.incremented", streamID, "Counter", v,
							[]byte(`{"amount":1}`),
						)
						if err != nil {
							b.Error(err)

							return
						}

						if err := store.AppendBatch(ctx, ref, []event.Event{evt}); err != nil {
							b.Error(err)

							return
						}
					}
				}()
			}

			wg.Wait()

			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/sec")
		})
	}
}
