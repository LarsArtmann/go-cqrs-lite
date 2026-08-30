package badgerengine_test

import (
	"context"
	"fmt"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// calibration_bench_test.go measures per-operation costs for the Badger engine.
// Results feed into BadgerNsPerOp/BadgerNsPerRead/BadgerNsPerWrite calibration.
//
// The per-read-pattern section calibrates the four ReadCosts fields in
// engine.go's Profile():
//
//   - NsPerPointLookup  → BenchmarkCalibration_BadgerGet (above; per-QUERY cost)
//   - NsPerFilteredScan → BenchmarkCalibration_Badger_FilteredScan (per-ROW cost;
//     KV engines have no SQL pushdown, so ReadFilteredScan degrades to a full
//     MapScan with a Go-side predicate)
//   - NsPerAggregate    → BenchmarkCalibration_Badger_CounterScan (per-ROW cost;
//     ReadAggregate executes CounterGet, a prefix scan over the counter bucket)
//   - NsPerScan         → BenchmarkCalibration_Badger_FullScan (per-ROW cost)
//
// Run: GOWORK=off go test -run='^$' -bench='BenchmarkCalibration_Badger' -benchmem ./...

func BenchmarkCalibration_BadgerSet(b *testing.B) {
	eng := mustNewBadgerEngine(b)

	mb := eng.(metaengine.MapBackend)

	ctx := context.Background()
	var i int

	for b.Loop() {
		if err := mb.MapSet(ctx, "bench", i, i*2); err != nil {
			b.Fatalf("MapSet %d: %v", i, err)
		}
		i++
	}
}

func BenchmarkCalibration_BadgerGet(b *testing.B) {
	eng := mustNewBadgerEngine(b)

	mb := eng.(metaengine.MapBackend)

	ctx := context.Background()

	for i := range 1000 {
		_ = mb.MapSet(ctx, "bench", i, i*2)
	}

	var i int

	for b.Loop() {
		_, found, err := mb.MapGet(ctx, "bench", i%1000)
		if err != nil {
			b.Fatalf("MapGet %d: %v", i, err)
		}
		if !found {
			b.Fatalf("MapGet %d: key not found", i)
		}
		i++
	}
}

func BenchmarkCalibration_BadgerCounterIncrement(b *testing.B) {
	eng := mustNewBadgerEngine(b)

	cb := eng.(metaengine.CounterBackend)

	ctx := context.Background()
	var i int

	for b.Loop() {
		key := fmt.Sprintf("k%d", i%100)
		if err := cb.CounterIncrement(ctx, "bench", metaengine.Delta{key: 1}); err != nil {
			b.Fatalf("CounterIncrement %d: %v", i, err)
		}
		i++
	}
}

// badgerCalibrationRows is the dataset size for the per-pattern scan benches
// (matches the DuckDB/Postgres calibration scale).
const badgerCalibrationRows = 10_000

// badgerCalibrationCounters is the counter-map size for the aggregate bench
// (matches the planner's ADTCounter scale threshold of ~1K distinct keys).
const badgerCalibrationCounters = 1_000

// populateBadgerCalibration seeds alternating-status rows through the public
// MapSet API so every row pays the real encode path.
func populateBadgerCalibration(tb testing.TB, mb metaengine.MapBackend, col string, n int) {
	tb.Helper()

	ctx := context.Background()

	for i := range n {
		status := "active"
		if i%2 == 0 {
			status = "inactive"
		}

		if err := mb.MapSet(ctx, col, fmt.Sprintf("k%d", i), map[string]any{
			"id": i, "status": status, "amount": float64(i) * 1.5,
		}); err != nil {
			tb.Fatalf("populate MapSet %d: %v", i, err)
		}
	}
}

// BenchmarkCalibration_Badger_FilteredScan measures the Go-side filtered scan
// that ReadFilteredScan degrades to on KV engines: MapScan decodes every row,
// the predicate rejects ~50%. Feeds ReadCosts.NsPerFilteredScan (per-ROW:
// divide ns/op by badgerCalibrationRows).
func BenchmarkCalibration_Badger_FilteredScan(b *testing.B) {
	eng := mustNewBadgerEngine(b)

	mb := eng.(metaengine.MapBackend)
	populateBadgerCalibration(b, mb, "scan", badgerCalibrationRows)

	sb := eng.(metaengine.ScanBackend)
	ctx := context.Background()

	for b.Loop() {
		result, err := sb.MapScan(ctx, "scan", func(item any) bool {
			m, ok := item.(map[string]any)

			return ok && m["status"] == "inactive"
		}, nil, nil, 0)
		if err != nil {
			b.Fatalf("MapScan: %v", err)
		}

		if len(result.Items) == 0 {
			b.Fatal("expected filtered results, got 0")
		}
	}

	b.ReportMetric(float64(badgerCalibrationRows), "rows-scanned")
}

// BenchmarkCalibration_Badger_CounterScan measures CounterGet — the actual
// ReadAggregate execution path on KV engines (execute.go routes ReadAggregate
// to CounterBackend.CounterGet). It is a prefix scan over the counter bucket.
// Feeds ReadCosts.NsPerAggregate (per-ROW: divide ns/op by
// badgerCalibrationCounters).
func BenchmarkCalibration_Badger_CounterScan(b *testing.B) {
	eng := mustNewBadgerEngine(b)

	cb := eng.(metaengine.CounterBackend)
	ctx := context.Background()

	for i := range badgerCalibrationCounters {
		if err := cb.CounterIncrement(ctx, "agg", metaengine.Delta{fmt.Sprintf("c%d", i): 1}); err != nil {
			b.Fatalf("seed CounterIncrement %d: %v", i, err)
		}
	}

	for b.Loop() {
		counts, err := cb.CounterGet(ctx, "agg")
		if err != nil {
			b.Fatalf("CounterGet: %v", err)
		}

		if len(counts) != badgerCalibrationCounters {
			b.Fatalf("CounterGet: expected %d counters, got %d", badgerCalibrationCounters, len(counts))
		}
	}

	b.ReportMetric(float64(badgerCalibrationCounters), "rows-scanned")
}

// BenchmarkCalibration_Badger_FullScan measures a full collection scan
// (MapScan, no filter, no limit): decode every row into Go. Feeds
// ReadCosts.NsPerScan (per-ROW: divide ns/op by badgerCalibrationRows).
func BenchmarkCalibration_Badger_FullScan(b *testing.B) {
	eng := mustNewBadgerEngine(b)

	mb := eng.(metaengine.MapBackend)
	populateBadgerCalibration(b, mb, "fullscan", badgerCalibrationRows)

	sb := eng.(metaengine.ScanBackend)
	ctx := context.Background()

	for b.Loop() {
		result, err := sb.MapScan(ctx, "fullscan", nil, nil, nil, 0)
		if err != nil {
			b.Fatalf("MapScan: %v", err)
		}

		if len(result.Items) != badgerCalibrationRows {
			b.Fatalf("MapScan: expected %d rows, got %d", badgerCalibrationRows, len(result.Items))
		}
	}

	b.ReportMetric(float64(badgerCalibrationRows), "rows-scanned")
}
