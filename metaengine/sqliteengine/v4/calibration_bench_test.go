package sqliteengine_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "modernc.org/sqlite"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// calibration_bench_test.go measures the per-read-pattern workloads that
// metaengine.SQLiteEngineProfile's ReadCosts model. SQLite is the only
// embedded SQL engine: point lookups hit the PK index, filtered scans push
// json_extract() WHERE into SQLite, and aggregation can push down as SQL SUM
// — but the planner-priced ReadAggregate path is CounterGet (ADR-0133), so
// the aggregate bench measures CounterGet, not SUM.
//
//   - NsPerPointLookup  → BenchmarkCalibration_SQLite_PointLookup (per-QUERY)
//   - NsPerFilteredScan → BenchmarkCalibration_SQLite_FilteredScan (per-ROW)
//   - NsPerAggregate    → BenchmarkCalibration_SQLite_CounterGet (per-ROW)
//   - NsPerScan         → BenchmarkCalibration_SQLite_FullScan (per-ROW)
//
// The measured values feed metaengine.SQLiteEngineProfile() in the core
// module (the sqlite profile factory lives there, not in this module).
//
// Run: GOWORK=off go test -run='^$' -bench='BenchmarkCalibration_SQLite' -benchmem ./...

const sqliteCalibrationRows = 10_000

const sqliteCalibrationCounters = 1_000

// newSQLiteBenchEngine opens an in-memory engine (single connection — the
// shared-cache DSN keeps one schema; bench workloads are single-goroutine).
func newSQLiteBenchEngine(tb testing.TB) (metaengine.Engine, *sql.DB) {
	tb.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		tb.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		tb.Fatalf("NewSQLiteEngine: %v", err)
	}

	return eng, db
}

// populateSQLiteCalibration seeds alternating-status rows through the public
// MapSet API so every row pays the real encode path.
func populateSQLiteCalibration(tb testing.TB, mb metaengine.MapBackend, col string, n int) {
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

// BenchmarkCalibration_SQLite_PointLookup measures an indexed PK lookup
// (MapGet) over 10K keys. Feeds ReadCosts.NsPerPointLookup (per-QUERY).
func BenchmarkCalibration_SQLite_PointLookup(b *testing.B) {
	eng, db := newSQLiteBenchEngine(b)
	defer metaengine.DeferClose(db)
	defer eng.Close()

	mb := eng.(metaengine.MapBackend)
	populateSQLiteCalibration(b, mb, "bench", sqliteCalibrationRows)

	ctx := context.Background()
	var i int

	for b.Loop() {
		_, found, err := mb.MapGet(ctx, "bench", fmt.Sprintf("k%d", i%sqliteCalibrationRows))
		if err != nil {
			b.Fatalf("MapGet %d: %v", i, err)
		}

		if !found {
			b.Fatalf("MapGet %d: key not found", i)
		}

		i++
	}
}

// BenchmarkCalibration_SQLite_FilteredScan measures PushdownMapScan with a
// json_extract WHERE matching ~50% — SQLite's actual filtered-scan execution
// path (executeFilteredScan prefers PushdownScan). Feeds
// ReadCosts.NsPerFilteredScan (per-ROW: divide ns/op by sqliteCalibrationRows).
func BenchmarkCalibration_SQLite_FilteredScan(b *testing.B) {
	eng, db := newSQLiteBenchEngine(b)
	defer metaengine.DeferClose(db)
	defer eng.Close()

	mb := eng.(metaengine.MapBackend)
	populateSQLiteCalibration(b, mb, "scan", sqliteCalibrationRows)

	ps, ok := eng.(metaengine.PushdownScan)
	if !ok {
		b.Fatal("sqlite engine does not implement PushdownScan")
	}

	ctx := context.Background()
	filters := []metaengine.FilterSpec{
		{Column: "status", Op: metaengine.FilterEq, Value: "inactive"},
	}

	for b.Loop() {
		res, err := ps.PushdownMapScan(ctx, "scan", filters, nil, nil, 0)
		if err != nil {
			b.Fatalf("PushdownMapScan: %v", err)
		}

		if len(res.Items) == 0 {
			b.Fatal("expected filtered results, got 0")
		}
	}

	b.ReportMetric(float64(sqliteCalibrationRows), "rows-scanned")
}

// BenchmarkCalibration_SQLite_CounterGet measures CounterGet over a 1K-key
// counter map — the actual ReadAggregate execution path (ADR-0133). Feeds
// ReadCosts.NsPerAggregate (per-ROW: divide ns/op by
// sqliteCalibrationCounters).
func BenchmarkCalibration_SQLite_CounterGet(b *testing.B) {
	eng, db := newSQLiteBenchEngine(b)
	defer metaengine.DeferClose(db)
	defer eng.Close()

	cb := eng.(metaengine.CounterBackend)
	ctx := context.Background()

	for i := range sqliteCalibrationCounters {
		if err := cb.CounterIncrement(
			ctx,
			"aggr",
			metaengine.Delta{fmt.Sprintf("c%d", i): 1},
		); err != nil {
			b.Fatalf("seed CounterIncrement %d: %v", i, err)
		}
	}

	for b.Loop() {
		counts, err := cb.CounterGet(ctx, "aggr")
		if err != nil {
			b.Fatalf("CounterGet: %v", err)
		}

		if len(counts) != sqliteCalibrationCounters {
			b.Fatalf(
				"CounterGet: expected %d counters, got %d",
				sqliteCalibrationCounters,
				len(counts),
			)
		}
	}

	b.ReportMetric(float64(sqliteCalibrationCounters), "rows-scanned")
}

// BenchmarkCalibration_SQLite_FullScan measures a full collection scan via
// ScanBackend.MapScan (no filter, no limit): decode every row into Go.
// Feeds ReadCosts.NsPerScan (per-ROW: divide ns/op by sqliteCalibrationRows).
func BenchmarkCalibration_SQLite_FullScan(b *testing.B) {
	eng, db := newSQLiteBenchEngine(b)
	defer metaengine.DeferClose(db)
	defer eng.Close()

	mb := eng.(metaengine.MapBackend)
	populateSQLiteCalibration(b, mb, "fullscan", sqliteCalibrationRows)

	sb := eng.(metaengine.ScanBackend)
	ctx := context.Background()

	for b.Loop() {
		result, err := sb.MapScan(ctx, "fullscan", nil, nil, nil, 0)
		if err != nil {
			b.Fatalf("MapScan: %v", err)
		}

		if len(result.Items) != sqliteCalibrationRows {
			b.Fatalf("MapScan: expected %d rows, got %d", sqliteCalibrationRows, len(result.Items))
		}
	}

	b.ReportMetric(float64(sqliteCalibrationRows), "rows-scanned")
}
