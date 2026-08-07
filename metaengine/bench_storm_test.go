package metaengine_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// bench_storm_test.go — measures event ingestion under heavy concurrent load.
// Models real-world event storms: many goroutines pushing events simultaneously.

// BenchmarkEventStorm_Concurrent measures total throughput when 8 goroutines
// each push 1000 mixed events into a 6-query store. This reveals contention
// and lock overhead in the fan-out path under realistic concurrent load.
func BenchmarkEventStorm_Concurrent(b *testing.B) {
	for _, concurrency := range []int{1, 4, 8} {
		b.Run(fmt.Sprintf("workers=%d", concurrency), func(b *testing.B) {
			eventsPerWorker := 1_000
			ctx := context.Background()

			b.ResetTimer()

			for range b.N {
				b.StopTimer()
				store := planPromiseStore(b, []metaengine.Engine{metaengine.NewMemoryEngine()})
				b.StartTimer()

				totalEvents := concurrency * eventsPerWorker
				events := generatePromiseEvents(totalEvents)
				var failCount atomic.Int64

				var wg sync.WaitGroup
				wg.Add(concurrency)

				for w := range concurrency {
					go func(workerID int) {
						defer wg.Done()
						start := eventsPerWorker * workerID
						end := start + eventsPerWorker
						for i := start; i < end; i++ {
							if err := store.Apply(ctx, events[i].typeName, events[i].payload); err != nil {
								failCount.Add(1)
								return
							}
						}
					}(w)
				}
				wg.Wait()

				if failCount.Load() > 0 {
					b.Fatalf("%d workers failed", failCount.Load())
				}

				b.StopTimer()
				store.Close()
				b.StartTimer()
			}

			b.ReportMetric(float64(concurrency*eventsPerWorker)*float64(b.N)/b.Elapsed().Seconds(), "events/sec")
		})
	}
}
