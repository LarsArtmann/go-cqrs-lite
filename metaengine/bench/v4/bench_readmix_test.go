package bench_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// bench_readmix_test.go — measures read-side performance after events have been
// ingested. Tests all 6 query types in round-robin and under concurrent write
// load.

// BenchmarkMultiQuery_ReadMix measures combined read throughput by executing
// all 6 query types in round-robin after seeding 10K events. This is the
// real-world read pattern: a dashboard querying multiple projections.
func BenchmarkMultiQuery_ReadMix(b *testing.B) {
	for _, n := range []int{1_000, 10_000} {
		b.Run(fmt.Sprintf("seeded=%d", n), func(b *testing.B) {
			store := planPromiseStore(b, []metaengine.Engine{metaengine.NewMemoryEngine()})
			defer store.Close()

			seedPromiseStore(b, store, n)
			ctx := context.Background()

			// Pre-compute valid lookup keys.
			orderID := OrderID(fmt.Sprintf("ord-%06d", 0))
			customerID := CustomerID("cus-000")

			b.ResetTimer()

			for range b.N {
				// Query 1: find_order (Map point lookup).
				_, _ = metaengine.ExecuteTyped[FindOrderInput, OrderView](
					ctx, store, FindOrderInput{ID: orderID})

				// Query 2: list_by_status (FilteredMap scan).
				_, _ = metaengine.ExecuteTyped[ListOrdersByStatusInput, OrderView](
					ctx, store, ListOrdersByStatusInput{Status: "pending", Limit: 10})

				// Query 3: count_by_status (Counter aggregate).
				_, _ = metaengine.ExecuteTyped[CountOrdersByStatusInput, map[string]int64](
					ctx, store, CountOrdersByStatusInput{})

				// Query 4: orders_by_customer (Multimap lookup).
				_, _ = metaengine.ExecuteTyped[OrdersByCustomerInput, []OrderID](
					ctx, store, OrdersByCustomerInput{Customer: customerID})

				// Query 5: recent_orders (Log tail).
				_, _ = metaengine.ExecuteTyped[RecentOrdersInput, []OrderID](
					ctx, store, RecentOrdersInput{Limit: 10})

				// Query 6: total_revenue (Counter aggregate).
				_, _ = metaengine.ExecuteTyped[TotalRevenueInput, map[string]int64](
					ctx, store, TotalRevenueInput{})
			}

			// Report queries/sec (6 queries per iteration).
			b.ReportMetric(float64(b.N)*6/b.Elapsed().Seconds(), "queries/sec")
		})
	}
}

// BenchmarkMultiQuery_MixedWorkload measures read latency under concurrent
// write load. Writers Apply events while readers ExecuteTyped queries. This
// reveals contention between the write fan-out and read paths.
func BenchmarkMultiQuery_MixedWorkload(b *testing.B) {
	for _, writerRatio := range []int{1, 4, 8} {
		b.Run(fmt.Sprintf("writers=%d", writerRatio), func(b *testing.B) {
			store := planPromiseStore(b, []metaengine.Engine{metaengine.NewMemoryEngine()})
			defer store.Close()

			// Pre-seed some data.
			seedPromiseStore(b, store, 500)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			events := generatePromiseEvents(2000)
			orderID := OrderID("ord-000000")
			customerID := CustomerID("cus-000")
			var readCount atomic.Int64
			var writeCount atomic.Int64

			// Start writer goroutines.
			var wg sync.WaitGroup
			wg.Add(writerRatio)
			for range writerRatio {
				go func() {
					defer wg.Done()
					for _, e := range events {
						select {
						case <-ctx.Done():
							return
						default:
						}
						if err := store.Apply(ctx, e.typeName, e.payload); err != nil {
							return
						}
						writeCount.Add(1)
					}
				}()
			}

			// Reader goroutine (the benchmark goroutine).
			b.ResetTimer()

			for range b.N {
				_, _ = metaengine.ExecuteTyped[FindOrderInput, OrderView](
					ctx, store, FindOrderInput{ID: orderID})
				_, _ = metaengine.ExecuteTyped[CountOrdersByStatusInput, map[string]int64](
					ctx, store, CountOrdersByStatusInput{})
				_, _ = metaengine.ExecuteTyped[OrdersByCustomerInput, []OrderID](
					ctx, store, OrdersByCustomerInput{Customer: customerID})
				_, _ = metaengine.ExecuteTyped[RecentOrdersInput, []OrderID](
					ctx, store, RecentOrdersInput{Limit: 10})
				_, _ = metaengine.ExecuteTyped[TotalRevenueInput, map[string]int64](
					ctx, store, TotalRevenueInput{})
				readCount.Add(5)
			}

			b.StopTimer()
			cancel()
			wg.Wait()

			b.ReportMetric(float64(readCount.Load())/b.Elapsed().Seconds(), "reads/sec")
			b.ReportMetric(float64(writeCount.Load())/b.Elapsed().Seconds(), "writes/sec")
		})
	}
}
