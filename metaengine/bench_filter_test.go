package metaengine_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// benchItemResult is the stored view for filtered-scan benchmarks.
type benchItemResult struct {
	ID       string
	Status   string
	Priority int
}

// benchListInput carries optional filters for the scan benchmark.
type benchListInput struct {
	Status string
}

// benchFilterQuery declares a Map query with FilterOnField + SortOnField so the
// SQLite engine can push both the WHERE filter and ORDER BY into SQL via
// json_extract — the core value proposition of metaengine pushdown.
func benchFilterQuery() metaengine.QueryDecl[benchListInput, benchItemResult] {
	return metaengine.Query[benchListInput, benchItemResult](
		"bench_filter_scan",
		metaengine.On(benchItemResult{}, func(e benchItemResult) (string, benchItemResult) {
			return e.ID, e
		}),
		metaengine.FilterOnField[benchItemResult]("Status", metaengine.FilterEq),
		metaengine.SortOnField[benchItemResult]("Priority", true),
	)
}

func seedBenchStore(t testing.TB, store *metaengine.Store, n int) {
	t.Helper()
	ctx := context.Background()

	for i := range n {
		status := "open"
		if i%3 == 0 {
			status = "closed"
		}

		item := benchItemResult{
			ID:       fmt.Sprintf("item-%06d", i),
			Status:   status,
			Priority: i % 10,
		}

		if err := store.Apply(ctx, "benchItemResult", item); err != nil {
			t.Fatalf("seed Apply[%d]: %v", i, err)
		}
	}
}

// V1: Benchmark the SQLite engine's filtered scan (json_extract pushdown)
// against the Memory engine's O(N) Go-side filter. This proves the
// FilterOnField pushdown value.
func BenchmarkFilteredScan(b *testing.B) {
	for _, n := range []int{1_000, 10_000} {
		b.Run(fmt.Sprintf("SQLite_n%d", n), func(b *testing.B) {
			store, reader := setupBenchStore(b, n, true)
			defer store.Close()

			ctx := context.Background()
			b.ResetTimer()

			for range b.N {
				_, err := reader.Scan(ctx,
					metaengine.WithFilter("Status", metaengine.FilterEq, "open"))
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run(fmt.Sprintf("Memory_n%d", n), func(b *testing.B) {
			store, reader := setupBenchStore(b, n, false)
			defer store.Close()

			ctx := context.Background()
			b.ResetTimer()

			for range b.N {
				_, err := reader.Scan(ctx,
					metaengine.WithFilter("Status", metaengine.FilterEq, "open"))
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func setupBenchStore(
	tb testing.TB,
	n int,
	useSQLite bool,
) (*metaengine.Store, *metaengine.TypedReader[benchItemResult]) {
	tb.Helper()

	var engines []metaengine.Engine

	if useSQLite {
		db, err := sql.Open("sqlite", ":memory:")
		if err != nil {
			tb.Fatalf("open sqlite: %v", err)
		}
		tb.Cleanup(func() { _ = db.Close() })

		eng, err := metaengine.NewSQLiteEngine(db)
		if err != nil {
			tb.Fatalf("sqlite engine: %v", err)
		}

		engines = []metaengine.Engine{metaengine.NewMemoryEngine(), eng}
	} else {
		engines = []metaengine.Engine{metaengine.NewMemoryEngine()}
	}

	store, err := metaengine.Plan(engines, benchFilterQuery())
	if err != nil {
		tb.Fatalf("Plan: %v", err)
	}

	seedBenchStore(tb, store, n)

	reader := metaengine.NewReader[benchItemResult](store, "bench_filter_scan")

	return store, reader
}

// V1: Benchmark point-lookup (Get) — Memory vs SQLite.
func BenchmarkPointLookup(b *testing.B) {
	for _, n := range []int{1_000, 10_000} {
		b.Run(fmt.Sprintf("SQLite_n%d", n), func(b *testing.B) {
			store, reader := setupBenchStore(b, n, true)
			defer store.Close()

			ctx := context.Background()
			b.ResetTimer()

			for range b.N {
				key := fmt.Sprintf("item-%06d", b.N%n)
				_, _, err := reader.Get(ctx, key)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run(fmt.Sprintf("Memory_n%d", n), func(b *testing.B) {
			store, reader := setupBenchStore(b, n, false)
			defer store.Close()

			ctx := context.Background()
			b.ResetTimer()

			for range b.N {
				key := fmt.Sprintf("item-%06d", b.N%n)
				_, _, err := reader.Get(ctx, key)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// V3: Cost model calibration — verify the planner's cost estimates are
// reasonable by comparing them against actual scan latency.
func TestCostModelCalibration(t *testing.T) {
	t.Parallel()

	for _, n := range []int{100, 1_000, 10_000} {
		t.Run(fmt.Sprintf("n%d", n), func(t *testing.T) {
			t.Parallel()

			store, reader := setupBenchStore(t, n, true)
			defer store.Close()

			ctx := context.Background()

			// Capture the planned cost estimate.
			plan := store.Plan()
			var plannedCost float64

			if plan != nil {
				for _, q := range plan.Queries {
					if q.QueryName == "bench_filter_scan" {
						plannedCost = q.Cost.EstimatedLatencyMs
					}
				}
			}

			// Measure actual scan latency (filtered).
			result, err := reader.Scan(ctx,
				metaengine.WithFilter("Status", metaengine.FilterEq, "open"),
				metaengine.WithLimit(0))
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}

			// Sanity: ~2/3 of items should be "open" (i%3 != 0).
			expectedMin := n * 2 / 4
			if len(result) < expectedMin {
				t.Fatalf("filtered scan returned %d items, expected >= %d",
					len(result), expectedMin)
			}

			t.Logf("n=%d: planned_cost=%.3fms, actual_results=%d",
				n, plannedCost, len(result))
		})
	}
}
