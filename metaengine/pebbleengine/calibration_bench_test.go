package pebbleengine_test

import (
	"context"
	"fmt"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// calibration_bench_test.go measures per-operation costs for the Pebble engine.
// Results feed into PebbleNsPerOp/PebbleNsPerRead/PebbleNsPerWrite calibration.
//
// The per-read-pattern section calibrates the four ReadCosts fields in
// engine.go's Profile(). Pebble's filtered-scan execution path is
// RawScanReader.ScanRawValues (execute.go prefers it over MapScan), so both
// scan benches drive that path — filters/sort run in Go over decoded rows.
//
//   - NsPerPointLookup  → BenchmarkCalibration_PebbleGet (above; per-QUERY cost)
//   - NsPerFilteredScan → BenchmarkCalibration_Pebble_FilteredScan (per-ROW cost)
//   - NsPerAggregate    → BenchmarkCalibration_Pebble_CounterScan (per-ROW cost;
//     ReadAggregate executes CounterGet, a prefix scan over the counter bucket)
//   - NsPerScan         → BenchmarkCalibration_Pebble_FullScan (per-ROW cost)
//
// Run: GOWORK=off go test -run='^$' -bench='BenchmarkCalibration_Pebble' -benchmem ./...

func BenchmarkCalibration_PebbleSet(b *testing.B) {
	eng := mustNewPebbleEngine(b)

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

func BenchmarkCalibration_PebbleGet(b *testing.B) {
	eng := mustNewPebbleEngine(b)

	mb := eng.(metaengine.MapBackend)

	ctx := context.Background()

	// Pre-populate.
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

func BenchmarkCalibration_PebbleCounterIncrement(b *testing.B) {
	eng := mustNewPebbleEngine(b)

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

// pebbleCalibrationRows is the dataset size for the per-pattern scan benches
// (matches the DuckDB/Postgres calibration scale).
const pebbleCalibrationRows = 10_000

// pebbleCalibrationCounters is the counter-map size for the aggregate bench
// (matches the planner's ADTCounter scale threshold of ~1K distinct keys).
const pebbleCalibrationCounters = 1_000

// populatePebbleCalibration seeds alternating-status rows through the public
// MapSet API so every row pays the real encode path.
func populatePebbleCalibration(tb testing.TB, mb metaengine.MapBackend, col string, n int) {
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

// BenchmarkCalibration_Pebble_FilteredScan measures ScanRawValues with a
// declarative filter — the path executeFilteredScan takes for Pebble
// (RawScanReader fast path). Every row is decoded and tested in Go; ~50%
// match. Feeds ReadCosts.NsPerFilteredScan (per-ROW: divide ns/op by
// pebbleCalibrationRows).
func BenchmarkCalibration_Pebble_FilteredScan(b *testing.B) {
	eng := mustNewPebbleEngine(b)

	mb := eng.(metaengine.MapBackend)
	populatePebbleCalibration(b, mb, "scan", pebbleCalibrationRows)

	rsr := eng.(metaengine.RawScanReader)
	ctx := context.Background()
	filters := []metaengine.FilterSpec{
		{Column: "status", Op: metaengine.FilterEq, Value: "inactive"},
	}

	for b.Loop() {
		result, err := rsr.ScanRawValues(ctx, "scan", filters, nil, nil, 0)
		if err != nil {
			b.Fatalf("ScanRawValues: %v", err)
		}

		if len(result.Items) == 0 {
			b.Fatal("expected filtered results, got 0")
		}
	}

	b.ReportMetric(float64(pebbleCalibrationRows), "rows-scanned")
}

// BenchmarkCalibration_Pebble_CounterScan measures CounterGet — the actual
// ReadAggregate execution path on KV engines (execute.go routes ReadAggregate
// to CounterBackend.CounterGet). It is a prefix scan over the counter bucket.
// Feeds ReadCosts.NsPerAggregate (per-ROW: divide ns/op by
// pebbleCalibrationCounters).
func BenchmarkCalibration_Pebble_CounterScan(b *testing.B) {
	eng := mustNewPebbleEngine(b)

	cb := eng.(metaengine.CounterBackend)
	ctx := context.Background()

	for i := range pebbleCalibrationCounters {
		if err := cb.CounterIncrement(ctx, "agg", metaengine.Delta{fmt.Sprintf("c%d", i): 1}); err != nil {
			b.Fatalf("seed CounterIncrement %d: %v", i, err)
		}
	}

	for b.Loop() {
		counts, err := cb.CounterGet(ctx, "agg")
		if err != nil {
			b.Fatalf("CounterGet: %v", err)
		}

		if len(counts) != pebbleCalibrationCounters {
			b.Fatalf(
				"CounterGet: expected %d counters, got %d",
				pebbleCalibrationCounters,
				len(counts),
			)
		}
	}

	b.ReportMetric(float64(pebbleCalibrationCounters), "rows-scanned")
}

// BenchmarkCalibration_Pebble_FullScan measures a full collection scan via
// ScanRawValues (no filter, no sort, no limit): decode every row in Go.
// Feeds ReadCosts.NsPerScan (per-ROW: divide ns/op by pebbleCalibrationRows).
func BenchmarkCalibration_Pebble_FullScan(b *testing.B) {
	eng := mustNewPebbleEngine(b)

	mb := eng.(metaengine.MapBackend)
	populatePebbleCalibration(b, mb, "fullscan", pebbleCalibrationRows)

	rsr := eng.(metaengine.RawScanReader)
	ctx := context.Background()

	for b.Loop() {
		result, err := rsr.ScanRawValues(ctx, "fullscan", nil, nil, nil, 0)
		if err != nil {
			b.Fatalf("ScanRawValues: %v", err)
		}

		if len(result.Items) != pebbleCalibrationRows {
			b.Fatalf(
				"ScanRawValues: expected %d rows, got %d",
				pebbleCalibrationRows,
				len(result.Items),
			)
		}
	}

	b.ReportMetric(float64(pebbleCalibrationRows), "rows-scanned")
}
