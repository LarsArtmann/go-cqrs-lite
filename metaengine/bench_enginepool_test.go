package metaengine_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// bench_enginepool_test.go — compares the same 6-query workload across
// different engine pool configurations. The planner routes each query to the
// optimal engine in the pool. This benchmark reveals whether multi-engine
// routing actually improves throughput.

// BenchmarkMultiQuery_EnginePoolComparison runs the full 6-query promise
// scenario against memory-only and memory+sqlite engine pools. The planner
// should route queries to different engines in the multi-engine case.
func BenchmarkMultiQuery_EnginePoolComparison(b *testing.B) {
	for _, pool := range promiseEnginePools() {
		b.Run(pool.name, func(b *testing.B) {
			engines := pool.engines()
			defer pool.cleanup()

			store, err := metaengine.Plan(engines, allPromiseQueries()...)
			if err != nil {
				b.Fatal(err)
			}
			defer store.Close()

			// Log routing decisions.
			plan := store.Plan()
			for _, q := range plan.Queries {
				b.Logf(
					"  %s → engine=%s cost=%.4fms",
					q.QueryName,
					q.EngineName,
					q.Cost.EstimatedLatencyMs,
				)
			}

			n := 1_000
			events := generatePromiseEvents(n)
			ctx := context.Background()

			// Phase 1: Write throughput.
			b.Run("write", func(b *testing.B) {
				b.ResetTimer()
				for range b.N {
					b.StopTimer()
					// Re-create store for each iteration to measure cold write.
					engines2 := pool.engines()
					store2, err := metaengine.Plan(engines2, allPromiseQueries()...)
					if err != nil {
						b.Fatal(err)
					}
					b.StartTimer()

					for _, e := range events {
						if err := store2.Apply(ctx, e.typeName, e.payload); err != nil {
							b.Fatal(err)
						}
					}

					b.StopTimer()
					store2.Close()
					b.StartTimer()
				}
				b.ReportMetric(float64(n)*float64(b.N)/b.Elapsed().Seconds(), "events/sec")
			})

			// Phase 2: Read throughput (after seeding).
			seedPromiseStore(b, store, n)
			orderID := OrderID("ord-000000")
			customerID := CustomerID("cus-000")

			b.Run("read", func(b *testing.B) {
				b.ResetTimer()
				for range b.N {
					_, _ = metaengine.ExecuteTyped[FindOrderInput, OrderView](
						ctx, store, FindOrderInput{ID: orderID})
					_, _ = metaengine.ExecuteTyped[CountOrdersByStatusInput, map[string]int64](
						ctx, store, CountOrdersByStatusInput{})
					_, _ = metaengine.ExecuteTyped[OrdersByCustomerInput, []OrderID](
						ctx, store, OrdersByCustomerInput{Customer: customerID})
				}
				b.ReportMetric(float64(b.N)*3/b.Elapsed().Seconds(), "queries/sec")
			})
		})
	}
}

// BenchmarkMultiQuery_EngineRoutingDecisions verifies that the planner
// distributes queries across engines when multiple are available. This is a
// correctness benchmark: it checks that routing actually happens, not just speed.
func TestPromise_EngineRoutingDecisions(t *testing.T) {
	t.Parallel()

	t.Run("memory-only routes all to memory", func(t *testing.T) {
		t.Parallel()
		store := planPromiseStore(t, []metaengine.Engine{metaengine.NewMemoryEngine()})
		defer store.Close()

		plan := store.Plan()
		for _, q := range plan.Queries {
			if q.EngineName != "memory" {
				t.Errorf("%s: expected memory, got %s", q.QueryName, q.EngineName)
			}
		}
	})

	t.Run("memory+sqlite distributes queries", func(t *testing.T) {
		t.Parallel()
		eng, db := newSQLiteEngine()
		defer func() { _ = db.Close() }()

		store := planPromiseStore(t, []metaengine.Engine{metaengine.NewMemoryEngine(), eng})
		defer store.Close()

		plan := store.Plan()
		engines := map[string]bool{}
		for _, q := range plan.Queries {
			engines[q.EngineName] = true
			t.Logf("  %s → %s", q.QueryName, q.EngineName)
		}
		if len(engines) < 2 {
			t.Errorf("expected queries distributed across 2+ engines, got %d", len(engines))
		}
	})
}

// suppress unused import warning for fmt when only running read benchmark.
var _ = fmt.Sprintf
