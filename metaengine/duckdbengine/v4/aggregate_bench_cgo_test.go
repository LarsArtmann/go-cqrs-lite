//go:build cgo

package duckdbengine_test

import (
	"context"
	"fmt"
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// seedAggBenchData inserts n rows into a DuckDB engine with 10 status groups.
func seedAggBenchData(b *testing.B, eng metaengine.Engine, n int) {
	b.Helper()

	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)

	b.StopTimer()

	for i := range n {
		status := fmt.Sprintf("status_%02d", i%10)
		val := map[string]any{
			"id":     fmt.Sprintf("item-%08d", i),
			"status": status,
			"price":  float64(i % 1000),
		}
		if err := mb.MapSet(ctx, "bench_items", val["id"], val); err != nil {
			b.Fatalf("MapSet %d: %v", i, err)
		}
	}

	b.StartTimer()
}

// BenchmarkDuckDB_GroupedAggregatePushdown measures SQL GROUP BY pushdown
// performance at 10K/100K rows. DuckDB's vectorized columnar scan computes
// COUNT per group without loading any rows into Go memory.
func BenchmarkDuckDB_GroupedAggregatePushdown(b *testing.B) {
	sizes := []int{10_000, 100_000}

	for _, n := range sizes {
		b.Run(fmt.Sprintf("rows_%dK", n/1000), func(b *testing.B) {
			eng, err := duckdbengine.New("")
			if err != nil {
				b.Skipf("DuckDB not available: %v", err)
			}

			defer eng.Close()

			seedAggBenchData(b, eng, n)

			gr := eng.(metaengine.GroupedAggregateReader)
			ctx := context.Background()

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := gr.GroupedAggregate(ctx, "bench_items",
					metaengine.AggregateCount, "", "status", nil)
				if err != nil {
					b.Fatalf("GroupedAggregate: %v", err)
				}
			}
		})
	}
}

// BenchmarkDuckDB_GroupedAggregate_GoSide measures the cost of loading all
// rows into Go and grouping manually — the fallback path when pushdown is
// unavailable. Uses the same DuckDB engine to keep storage constant.
func BenchmarkDuckDB_GroupedAggregate_GoSide(b *testing.B) {
	sizes := []int{10_000, 100_000}

	for _, n := range sizes {
		b.Run(fmt.Sprintf("rows_%dK", n/1000), func(b *testing.B) {
			eng, err := duckdbengine.New("")
			if err != nil {
				b.Skipf("DuckDB not available: %v", err)
			}

			defer eng.Close()

			seedAggBenchData(b, eng, n)

			ps := eng.(metaengine.PushdownScan)
			ctx := context.Background()

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result, err := ps.PushdownMapScan(ctx, "bench_items", nil, nil, "", 0)
				if err != nil {
					b.Fatalf("PushdownMapScan: %v", err)
				}

				// Manual grouping in Go.
				counts := make(map[string]int)
				for _, row := range result.Items {
					if m, ok := row.(map[string]any); ok {
						if status, ok := m["status"].(string); ok {
							counts[status]++
						}
					}
				}
			}
		})
	}
}

// BenchmarkDuckDB_MultiAggregate_SinglePass measures one SQL query that
// computes COUNT + SUM + AVG + MIN + MAX simultaneously.
func BenchmarkDuckDB_MultiAggregate_SinglePass(b *testing.B) {
	sizes := []int{10_000, 100_000}

	specs := []metaengine.AggregateSpec{
		{Fn: metaengine.AggregateCount, Alias: "cnt"},
		{Fn: metaengine.AggregateSum, Column: "price", Alias: "total"},
		{Fn: metaengine.AggregateAvg, Column: "price", Alias: "avg"},
		{Fn: metaengine.AggregateMin, Column: "price", Alias: "min"},
		{Fn: metaengine.AggregateMax, Column: "price", Alias: "max"},
	}

	for _, n := range sizes {
		b.Run(fmt.Sprintf("rows_%dK", n/1000), func(b *testing.B) {
			eng, err := duckdbengine.New("")
			if err != nil {
				b.Skipf("DuckDB not available: %v", err)
			}

			defer eng.Close()

			seedAggBenchData(b, eng, n)

			mr := eng.(metaengine.MultiAggregateReader)
			ctx := context.Background()

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := mr.MultiAggregate(ctx, "bench_items", specs, nil)
				if err != nil {
					b.Fatalf("MultiAggregate: %v", err)
				}
			}
		})
	}
}

// BenchmarkDuckDB_MultiAggregate_N_Calls measures 5 separate Aggregate
// queries. This is the anti-pattern: N SQL round-trips instead of 1.
func BenchmarkDuckDB_MultiAggregate_N_Calls(b *testing.B) {
	sizes := []int{10_000, 100_000}

	calls := []struct {
		fn     metaengine.AggregateFn
		column string
	}{
		{metaengine.AggregateCount, ""},
		{metaengine.AggregateSum, "price"},
		{metaengine.AggregateAvg, "price"},
		{metaengine.AggregateMin, "price"},
		{metaengine.AggregateMax, "price"},
	}

	for _, n := range sizes {
		b.Run(fmt.Sprintf("rows_%dK", n/1000), func(b *testing.B) {
			eng, err := duckdbengine.New("")
			if err != nil {
				b.Skipf("DuckDB not available: %v", err)
			}

			defer eng.Close()

			seedAggBenchData(b, eng, n)

			ar := eng.(metaengine.AggregateReader)
			ctx := context.Background()

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				for _, c := range calls {
					_, err := ar.Aggregate(ctx, "bench_items", c.fn, c.column, nil)
					if err != nil {
						b.Fatalf("Aggregate %s: %v", c.fn, err)
					}
				}
			}
		})
	}
}
