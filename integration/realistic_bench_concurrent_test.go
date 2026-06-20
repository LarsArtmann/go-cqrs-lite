//go:build scale

package integration_test

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v2"
)

// ---------------------------------------------------------------------------
// 3. Concurrent Decider — N CPU goroutines, 100 ops each
// ---------------------------------------------------------------------------

func BenchmarkRealistic_ConcurrentDecider(b *testing.B) {
	b.ReportAllocs()

	store := memory.NewMemoryStore()
	bus := eventtest.NewFakeBus()
	b.Cleanup(func() { _ = store.Close(); _ = bus.Close() })

	repo := benchNewOrderRepo(b, store, bus, 50)

	ctx := context.Background()
	workers := runtime.NumCPU()
	opsPerWorker := 100

	b.ResetTimer()

	for b.Loop() {
		var wg sync.WaitGroup
		var errCount atomic.Int64

		for w := range workers {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := range opsPerWorker {
					aggID := id.NewAggregateID()
					err := repo.Execute(ctx, aggID, "Order",
						func(_ OrderState, ver event.Version) ([]event.Event, error) {
							return []event.Event{
								newRealisticEvent(
									b,
									"OrderCreated",
									aggID,
									ver.Increment(),
									OrderCreated{
										OrderID:   aggID.String(),
										Customer:  fmt.Sprintf("w%d-op%d", workerID, j),
										Total:     99.99,
										Items:     1,
										Timestamp: time.Now().Format(time.RFC3339),
									},
								),
							}, nil
						})
					if err != nil {
						errCount.Add(1)
					}
				}
			}(w)
		}

		wg.Wait()

		if errCount.Load() > 0 {
			b.Fatalf("%d errors during concurrent execute", errCount.Load())
		}
	}

	totalOps := b.N * workers * opsPerWorker
	b.ReportMetric(float64(workers), "goroutines")
	b.ReportMetric(float64(totalOps)/b.Elapsed().Seconds(), "executes/sec")
}
