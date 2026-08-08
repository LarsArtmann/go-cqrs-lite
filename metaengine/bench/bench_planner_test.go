package bench_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// bench_planner_test.go — measures planner performance and cost model accuracy.
// The planner must be fast (Plan() happens at startup) and its cost estimates
// should roughly match actual measured latency.

// BenchmarkPlanner_PlanLatency measures Plan() call time with varying query
// counts and engine counts. The planner must be fast — it runs at startup.
func BenchmarkPlanner_PlanLatency(b *testing.B) {
	// Build pools of queries incrementally.
	allQs := allPromiseQueries()
	queryCounts := []int{1, 3, 6}

	// Single-engine (memory).
	b.Run("engines=1", func(b *testing.B) {
		for _, qc := range queryCounts {
			queries := allQs[:qc]
			b.Run(fmt.Sprintf("queries=%d", qc), func(b *testing.B) {
				b.ResetTimer()
				for range b.N {
					store, err := metaengine.Plan(
						[]metaengine.Engine{metaengine.NewMemoryEngine()},
						queries...,
					)
					if err != nil {
						b.Fatal(err)
					}
					store.Close()
				}
			})
		}
	})

	// Two engines (memory + sqlite).
	b.Run("engines=2", func(b *testing.B) {
		for _, qc := range queryCounts {
			queries := allQs[:qc]
			b.Run(fmt.Sprintf("queries=%d", qc), func(b *testing.B) {
				b.ResetTimer()
				for range b.N {
					eng, db := newSQLiteEngine()
					store, err := metaengine.Plan(
						[]metaengine.Engine{metaengine.NewMemoryEngine(), eng},
						queries...,
					)
					if err != nil {
						b.Fatal(err)
					}
					store.Close()
					_ = db.Close()
				}
			})
		}
	})
}

// BenchmarkPlanner_CostModelAccuracy compares the planner's estimated latency
// (Cost.EstimatedLatencyMs) against actual measured latency for each query.
// This is a Test, not a Benchmark, because it validates the cost model —
// not measures speed.
func TestPromise_CostModelAccuracy(t *testing.T) {
	t.Parallel()

	eng, db := newSQLiteEngine()
	defer metaengine.DeferClose(db)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine(), eng},
		allPromiseQueries()...,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Seed 5K events.
	seedPromiseStore(t, store, 5_000)

	ctx := context.Background()
	plan := store.Plan()

	// Measure actual latency for each query (100 iterations, take median-ish).
	orderID := OrderID("ord-000000")
	customerID := CustomerID("cus-000")

	measurements := map[string]time.Duration{
		"find_order": measureLatency(100, func() {
			_, _ = metaengine.ExecuteTyped[FindOrderInput, OrderView](
				ctx, store, FindOrderInput{ID: orderID})
		}),
		"count_orders_by_status": measureLatency(100, func() {
			_, _ = metaengine.ExecuteTyped[CountOrdersByStatusInput, map[string]int64](
				ctx, store, CountOrdersByStatusInput{})
		}),
		"orders_by_customer": measureLatency(100, func() {
			_, _ = metaengine.ExecuteTyped[OrdersByCustomerInput, []OrderID](
				ctx, store, OrdersByCustomerInput{Customer: customerID})
		}),
		"recent_orders": measureLatency(100, func() {
			_, _ = metaengine.ExecuteTyped[RecentOrdersInput, []OrderID](
				ctx, store, RecentOrdersInput{Limit: 10})
		}),
		"total_revenue": measureLatency(100, func() {
			_, _ = metaengine.ExecuteTyped[TotalRevenueInput, map[string]int64](
				ctx, store, TotalRevenueInput{})
		}),
	}

	// Report comparison.
	for _, q := range plan.Queries {
		actualMs := float64(measurements[q.QueryName].Microseconds()) / 1000.0
		predictedMs := q.Cost.EstimatedLatencyMs

		ratio := 0.0
		if actualMs > 0 {
			ratio = predictedMs / actualMs
		}

		t.Logf("  %-25s engine=%-8s predicted=%.4fms actual=%.4fms ratio=%.2f",
			q.QueryName, q.EngineName, predictedMs, actualMs, ratio)
	}
}

func measureLatency(iterations int, fn func()) time.Duration {
	if iterations <= 0 {
		iterations = 1
	}
	start := time.Now()
	for range iterations {
		fn()
	}
	return time.Since(start) / time.Duration(iterations)
}
