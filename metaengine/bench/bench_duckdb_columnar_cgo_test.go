//go:build cgo

package bench_test

import (
	"context"
	"fmt"
	"sync"
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
// Expected ordering: Columnar fastest, Pushdown second. The Go-side Memory
// baseline for the same filtered scan lives in benchkit (map filtered-Scan
// phase), which is engine-agnostic and does not need DuckDB to build.

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
		metaengine.OnRecord(
			benchColumnarItem{},
			func(_ record.Record, e benchColumnarItem) (string, benchColumnarItem) {
				return e.ID, e
			},
		),
		metaengine.FilterOnField[benchColumnarItem]("Status", metaengine.FilterEq),
		metaengine.SortOnField[benchColumnarItem]("Amount", true),
		metaengine.WithColumnarLayout(),
	)
}

func pushdownScanQuery() metaengine.QueryDecl[benchColumnarInput, benchColumnarItem] {
	return metaengine.Query[benchColumnarInput, benchColumnarItem](
		"pushdown_scan",
		metaengine.OnRecord(
			benchColumnarItem{},
			func(_ record.Record, e benchColumnarItem) (string, benchColumnarItem) {
				return e.ID, e
			},
		),
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

// scanBenchFixture is a seeded DuckDB store + reader, cached across -count
// repetitions by cachedScanFixture below.
type scanBenchFixture struct {
	store  *metaengine.Store
	reader *metaengine.TypedReader[benchColumnarItem]
}

// scanFixtureCache caches one seeded fixture per (layout, n) for the whole
// test process. Rationale: seeding n=100_000 through Store.Apply costs
// minutes of wall time (per-row INSERT overhead) while the measured scan is
// sub-second; go test's -count re-invokes the whole benchmark function, which
// previously re-seeded every repetition and blew the CI Benchmark Matrix
// past its 45-minute timeout every night since 2026-10-26. The scan is
// read-only, so reuse across repetitions is measurement-honest (setup was
// already outside the timer via ResetTimer). The in-memory database is freed
// at process exit; no tb.Cleanup may close it, because later -count runs
// still use the fixture after the creating run returned.
var scanFixtureCache sync.Map // "columnar|1000" → *scanBenchFixture

// cachedScanFixture returns a seeded fixture for the named layout and scale,
// building and seeding it on first use only.
func cachedScanFixture(
	tb testing.TB,
	layout string,
	n int,
	decl metaengine.QueryDecl[benchColumnarInput, benchColumnarItem],
	queryName string,
) *scanBenchFixture {
	tb.Helper()

	key := fmt.Sprintf("%s|%d", layout, n)

	if v, ok := scanFixtureCache.Load(key); ok {
		return v.(*scanBenchFixture)
	}

	eng := newDuckDBEngine(tb)

	store, err := metaengine.Plan([]metaengine.Engine{eng}, decl)
	if err != nil {
		tb.Fatalf("Plan: %v", err)
	}

	seedColumnarItems(tb, store, n)

	fixture := &scanBenchFixture{
		store:  store,
		reader: metaengine.NewReader[benchColumnarItem](store, queryName),
	}

	actual, _ := scanFixtureCache.LoadOrStore(key, fixture)

	return actual.(*scanBenchFixture)
}

// BenchmarkColumnarScan_DuckDB measures a filtered+sorted scan on a DuckDB
// planned table where ALL fields are native typed columns (WithColumnarLayout).
// This is the fastest path: no JSON decode, vectorized column scans.
func BenchmarkColumnarScan_DuckDB(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			fixture := cachedScanFixture(b, "columnar", n, columnarScanQuery(), "columnar_scan")
			ctx := context.Background()

			b.ResetTimer()

			for range b.N {
				results, err := fixture.reader.Scan(ctx,
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
			fixture := cachedScanFixture(b, "pushdown", n, pushdownScanQuery(), "pushdown_scan")
			ctx := context.Background()

			b.ResetTimer()

			for range b.N {
				results, err := fixture.reader.Scan(ctx,
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
