//go:build cgo

package bench_test

import (
	"context"
	"fmt"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// ─── M4.2: DuckDB Columnar Extraction 3-Way Comparison ───
//
// This benchmark is THE deliverable of the metaengine/bench module. It
// compares three approaches to filtered+sorted scans at scale (1K/10K/100K):
//
//  1. Columnar (DuckDB + WithColumnarLayout): ALL fields extracted as native
//     typed columns (DOUBLE, INTEGER, VARCHAR). DuckDB scans native columns
//     with vectorized execution — no JSON decode per row.
//
//  2. Pushdown (DuckDB, no columnar): json_extract(meta_map, '$.field') for
//     WHERE filter + ORDER BY. DuckDB decodes JSON per row but still pushes
//     filter/sort into SQL (better than Go-side, worse than native columns).
//
//  3. Memory (O(N) Go-side filter): The Memory engine loads every row and
//     applies the filter closure in Go. Baseline for comparison.
//
// Expected ordering: Columnar fastest, Pushdown second, Memory slowest.

type benchColumnarItem struct {
	ID       string
	Status   string
	Priority int
	Amount   float64
	Category string
}

type benchColumnarInput struct {
	Status string
}

func columnarScanQuery() metaengine.QueryDecl[benchColumnarInput, benchColumnarItem] {
	return metaengine.Query[benchColumnarInput, benchColumnarItem](
		"columnar_scan",
		metaengine.OnRecord(benchColumnarItem{}, func(_ record.Record, e benchColumnarItem) (string, benchColumnarItem) {
			return e.ID, e
		}),
		metaengine.FilterOnField[benchColumnarItem]("Status", metaengine.FilterEq),
		metaengine.SortOnField[benchColumnarItem]("Amount", true),
		metaengine.WithColumnarLayout(),
	)
}

func pushdownScanQuery() metaengine.QueryDecl[benchColumnarInput, benchColumnarItem] {
	return metaengine.Query[benchColumnarInput, benchColumnarItem](
		"pushdown_scan",
		metaengine.OnRecord(benchColumnarItem{}, func(_ record.Record, e benchColumnarItem) (string, benchColumnarItem) {
			return e.ID, e
		}),
		metaengine.FilterOnField[benchColumnarItem]("Status", metaengine.FilterEq),
		metaengine.SortOnField[benchColumnarItem]("Amount", true),
	)
}

func seedColumnarItems(tb testing.TB, store *metaengine.Store, n int) {
	tb.Helper()

	ctx := context.Background()

	for i := range n {
		status := "active"
		if i%3 == 0 {
			status = "archived"
		}

		item := benchColumnarItem{
			ID:       fmt.Sprintf("item-%06d", i),
			Status:   status,
			Priority: i % 10,
			Amount:   float64(i) * 1.5,
			Category: fmt.Sprintf("cat-%d", i%5),
		}

		if err := store.Apply(ctx, "benchColumnarItem", item); err != nil {
			tb.Fatalf("seed Apply[%d]: %v", i, err)
		}
	}
}

// BenchmarkColumnarScan_DuckDB measures a filtered+sorted scan on a DuckDB
// planned table where ALL fields are native typed columns (WithColumnarLayout).
// This is the fastest path: no JSON decode, vectorized column scans.
func BenchmarkColumnarScan_DuckDB(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			eng := newDuckDBEngine(b)

			store, err := metaengine.Plan([]metaengine.Engine{eng}, columnarScanQuery())
			if err != nil {
				b.Fatalf("Plan: %v", err)
			}

			defer store.Close()

			seedColumnarItems(b, store, n)

			reader := metaengine.NewReader[benchColumnarItem](store, "columnar_scan")
			ctx := context.Background()

			b.ResetTimer()

			for range b.N {
				results, err := reader.Scan(ctx,
					metaengine.WithFilter("Status", metaengine.FilterEq, "active"),
					metaengine.WithLimit(0))
				if err != nil {
					b.Fatal(err)
				}

				if len(results) == 0 {
					b.Fatal("expected results, got 0")
				}
			}

			b.StopTimer()
			b.ReportMetric(float64(n), "rows-scanned")
		})
	}
}

// BenchmarkPushdownScan_DuckDB measures the same filtered+sorted scan on DuckDB
// WITHOUT a columnar layout — DuckDB uses json_extract on the meta_map JSON
// blob. Compare with BenchmarkColumnarScan_DuckDB to see the speedup from
// native typed columns vs JSON decoding.
func BenchmarkPushdownScan_DuckDB(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			eng := newDuckDBEngine(b)

			store, err := metaengine.Plan([]metaengine.Engine{eng}, pushdownScanQuery())
			if err != nil {
				b.Fatalf("Plan: %v", err)
			}

			defer store.Close()

			seedColumnarItems(b, store, n)

			reader := metaengine.NewReader[benchColumnarItem](store, "pushdown_scan")
			ctx := context.Background()

			b.ResetTimer()

			for range b.N {
				results, err := reader.Scan(ctx,
					metaengine.WithFilter("Status", metaengine.FilterEq, "active"),
					metaengine.WithLimit(0))
				if err != nil {
					b.Fatal(err)
				}

				if len(results) == 0 {
					b.Fatal("expected results, got 0")
				}
			}

			b.StopTimer()
			b.ReportMetric(float64(n), "rows-scanned")
		})
	}
}

// BenchmarkFilteredScan_Memory measures the same filtered+sorted scan on the
// Memory engine — O(N) Go-side filter with closure evaluation. This is the
// baseline: every row is loaded and the filter closure runs in Go.
func BenchmarkFilteredScan_Memory(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			store, err := metaengine.Plan(
				[]metaengine.Engine{metaengine.NewMemoryEngine()},
				pushdownScanQuery(),
			)
			if err != nil {
				b.Fatalf("Plan: %v", err)
			}

			defer store.Close()

			seedColumnarItems(b, store, n)

			reader := metaengine.NewReader[benchColumnarItem](store, "pushdown_scan")
			ctx := context.Background()

			b.ResetTimer()

			for range b.N {
				results, err := reader.Scan(ctx,
					metaengine.WithFilter("Status", metaengine.FilterEq, "active"),
					metaengine.WithLimit(0))
				if err != nil {
					b.Fatal(err)
				}

				if len(results) == 0 {
					b.Fatal("expected results, got 0")
				}
			}

			b.StopTimer()
			b.ReportMetric(float64(n), "rows-scanned")
		})
	}
}
