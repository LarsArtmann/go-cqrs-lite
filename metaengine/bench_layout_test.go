package metaengine_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// bench_layout_test.go — measures the layout planning payoff: SQLite's
// FilterOnField + SortOnField pushdown (json_extract WHERE/ORDER BY) vs
// Memory's O(N) Go-side scan. At 10K rows the pushdown should be significantly
// faster.

// BenchmarkLayoutPlanning_MemoryVsSQLite measures filtered+sorted scan latency
// across Memory (O(N) Go closure filter) and SQLite (json_extract pushdown)
// using the promise domain's list_orders_by_status query.
func BenchmarkLayoutPlanning_MemoryVsSQLite(b *testing.B) {
	for _, n := range []int{1_000, 10_000} {
		// Memory engine: O(N) Go-side filter+sort.
		b.Run(fmt.Sprintf("Memory_n%d", n), func(b *testing.B) {
			store := planPromiseStore(b, []metaengine.Engine{metaengine.NewMemoryEngine()})
			defer store.Close()
			seedPromiseStore(b, store, n)
			ctx := context.Background()

			b.ResetTimer()
			for range b.N {
				_, _ = metaengine.ExecuteTyped[ListOrdersByStatusInput, OrderView](
					ctx, store, ListOrdersByStatusInput{Status: "pending", Limit: 50})
			}
		})

		// SQLite engine: json_extract pushdown for WHERE + ORDER BY.
		b.Run(fmt.Sprintf("SQLite_n%d", n), func(b *testing.B) {
			eng, db := newSQLiteEngine()
			defer func() { _ = db.Close() }()

			store := planPromiseStore(b, []metaengine.Engine{metaengine.NewMemoryEngine(), eng})
			defer store.Close()
			seedPromiseStore(b, store, n)
			ctx := context.Background()

			b.ResetTimer()
			for range b.N {
				_, _ = metaengine.ExecuteTyped[ListOrdersByStatusInput, OrderView](
					ctx, store, ListOrdersByStatusInput{Status: "pending", Limit: 50})
			}
		})
	}
}

// BenchmarkLayoutPlanning_PlanTime measures the Plan() overhead when layout
// planning is active (SQLite engine with FilterOnField+SortOnField). This is
// the one-time cost at startup.
func BenchmarkLayoutPlanning_PlanTime(b *testing.B) {
	b.Run("memory-only", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			store, err := metaengine.Plan(
				[]metaengine.Engine{metaengine.NewMemoryEngine()},
				allPromiseQueries()...,
			)
			if err != nil {
				b.Fatal(err)
			}
			store.Close()
		}
	})

	b.Run("memory+sqlite", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			eng, db := newSQLiteEngine()
			store, err := metaengine.Plan(
				[]metaengine.Engine{metaengine.NewMemoryEngine(), eng},
				allPromiseQueries()...,
			)
			if err != nil {
				b.Fatal(err)
			}
			store.Close()
			_ = db.Close()
		}
	})
}
