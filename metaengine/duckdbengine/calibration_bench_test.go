//go:build cgo

package duckdbengine_test

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"
	"strings"
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// calibration_bench_test.go measures the analytical workloads that DuckDB is
// designed for — the workloads the DuckDBNsPerOp/DuckDBNsPerRead constants
// actually model.
//
// The existing bench_test.go measures point lookups (single MapSet/MapGet),
// which are DuckDB's WORST case (~3.9M ns/op write, ~546K ns/op read). Those
// numbers would make the planner reject DuckDB for every query, defeating its
// purpose as an OLAP engine. The constants instead model the INTENDED use case:
//
//   - DuckDBNsPerOp  → batch-amortized columnar writes (BenchmarkCalibration_DuckDB_BatchInsert)
//   - DuckDBNsPerRead → vectorized scan + aggregation (BenchmarkCalibration_DuckDB_PushdownScan,
//                       BenchmarkCalibration_DuckDB_AggregateSum, BenchmarkCalibration_DuckDB_FullScan)
//
// Run: GOWORK=off go test -tags 'cgo goexperiment.jsonv2' -run='^$' \
//      -bench='BenchmarkCalibration_DuckDB' -benchmem ./...
//
// Interpretation: the "ns/row" custom metric is the per-row amortized cost.
// Compare against DuckDBNsPerOp (write) and DuckDBNsPerRead (read).

// calibrationRows is the dataset size for scan/aggregation benchmarks. Large
// enough that per-row cost dominates fixed overhead, small enough to run fast.
const calibrationRows = 10_000

// calibrationPayload is the JSON document stored per row. The "status" field
// enables selective filtering (~50% match); "amount" enables SUM aggregation.
type calibrationPayload struct {
	ID     int     `json:"id"`
	Status string  `json:"status"`
	Amount float64 `json:"amount"`
}

// openDuckDBForBench opens a raw *sql.DB with an independent in-memory DuckDB
// instance for setup and raw-SQL benchmarks. The engine registers the "duckdb"
// driver. Returns a DB whose lifecycle the caller owns.
func openDuckDBForBench(t testing.TB) *sql.DB {
	t.Helper()

	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}

	ddls := []string{
		`CREATE TABLE IF NOT EXISTS meta_map (
			collection VARCHAR NOT NULL,
			key VARCHAR NOT NULL,
			value VARCHAR NOT NULL,
			PRIMARY KEY (collection, key)
		)`,
		`CREATE TABLE IF NOT EXISTS meta_counter (
			collection VARCHAR NOT NULL,
			key VARCHAR NOT NULL,
			value BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY (collection, key)
		)`,
	}

	for _, ddl := range ddls {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatalf("ddl: %v", err)
		}
	}

	return db
}

// BenchmarkCalibration_DuckDB_BatchInsert measures batch-amortized columnar
// writes: a single multi-VALUES INSERT of 1000 rows in one statement. This is
// the canonical "batch-amortized columnar write" workload DuckDBNsPerOp models
// — DuckDB's vectorized execution engine amortizes the columnar flush across
// the entire batch in one statement.
// The "ns/row" metric calibrates the per-write constant.
//
// Contrast with BenchmarkDuckDB_MapSet (bench_test.go) which measures
// single-statement autocommit writes — DuckDB's worst case (~3.9M ns/op).
func BenchmarkCalibration_DuckDB_BatchInsert(b *testing.B) {
	const batchSize = 1000

	db := openDuckDBForBench(b)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// Pre-build the payloads (marshal is not part of the measured write cost).
	type row struct{ col, key, val string }
	batch := make([]row, batchSize)

	for i := range batchSize {
		raw, err := json.Marshal(calibrationPayload{
			ID: i, Status: "active", Amount: float64(i) * 1.5,
		})
		if err != nil {
			b.Fatalf("marshal: %v", err)
		}

		batch[i] = row{col: "batch", key: fmt.Sprintf("k%d", i), val: string(raw)}
	}

	// Pre-build the VALUES clause once (parameter placeholders are stable).
	placeholders := make([]string, batchSize)
	args := make([]any, 0, batchSize*3)

	for i, r := range batch {
		placeholders[i] = fmt.Sprintf("($%d, $%d, $%d)", i*3+1, i*3+2, i*3+3)
		args = append(args, r.col, r.key, r.val)
	}

	q := "INSERT INTO meta_map (collection, key, value) VALUES " + strings.Join(placeholders, ", ")

	b.ResetTimer()

	for iter := 0; iter < b.N; iter++ {
		// Unique collection per iteration to avoid PK conflicts. Every row's
		// collection arg (indices 0, 3, 6, ...) must be updated.
		col := fmt.Sprintf("batch_%d", iter)

		for i := range batchSize {
			args[i*3] = col
		}

		if _, err := db.ExecContext(ctx, q, args...); err != nil {
			b.Fatalf("batch insert %d: %v", iter, err)
		}
	}

	b.ReportMetric(float64(batchSize), "rows/batch")
}

// BenchmarkCalibration_DuckDB_PushdownScan measures a filtered vectorized scan:
// PushdownMapScan with a FilterEq that matches ~50% of calibrationRows. This is
// the workload DuckDBNsPerRead models — json_extract WHERE pushdown with DuckDB's
// vectorized execution. The "ns/row" metric calibrates the per-read constant.
func BenchmarkCalibration_DuckDB_PushdownScan(b *testing.B) {
	eng, err := duckdbengine.New("")
	if err != nil {
		b.Skipf("DuckDB not available: %v", err)
	}
	defer eng.Close()

	pushdown, ok := eng.(metaengine.PushdownScan)
	if !ok {
		b.Fatal("duckdb engine does not implement PushdownScan")
	}

	populateDuckDBEngine(b, eng, "scan", calibrationRows)

	ctx := context.Background()
	filters := []metaengine.FilterSpec{
		{Column: "status", Op: metaengine.FilterEq, Value: "active"},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		res, err := pushdown.PushdownMapScan(ctx, "scan", filters, nil, nil, 0)
		if err != nil {
			b.Fatalf("PushdownMapScan %d: %v", i, err)
		}

		if len(res.Items) == 0 {
			b.Fatalf("PushdownMapScan %d: expected results, got 0", i)
		}
	}

	b.StopTimer()
	b.ReportMetric(float64(calibrationRows), "rows-scanned")
}

// BenchmarkCalibration_DuckDB_AggregateSum measures vectorized SQL aggregation:
// SUM(amount) over calibrationRows via json_extract. This is DuckDB's killer
// feature (vectorized GROUP BY) that DuckDBNsPerRead is designed to represent.
// The "ns/row" metric calibrates the per-read constant for analytical reads.
func BenchmarkCalibration_DuckDB_AggregateSum(b *testing.B) {
	db := openDuckDBForBench(b)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	if err := populateDuckDBRaw(ctx, db, "agg", calibrationRows); err != nil {
		b.Fatalf("populate: %v", err)
	}

	const q = `SELECT SUM(CAST(json_extract_string(value, '$.amount') AS DOUBLE))
	           FROM meta_map WHERE collection = $1`

	// Warm up + correctness check.
	var sum sql.NullFloat64
	if err := db.QueryRowContext(ctx, q, "agg").Scan(&sum); err != nil {
		b.Fatalf("warmup query: %v", err)
	}

	if !sum.Valid || sum.Float64 <= 0 {
		b.Fatalf("warmup: expected positive sum, got %v (valid=%v)", sum.Float64, sum.Valid)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var res sql.NullFloat64
		if err := db.QueryRowContext(ctx, q, "agg").Scan(&res); err != nil {
			b.Fatalf("query %d: %v", i, err)
		}

		if !res.Valid {
			b.Fatalf("query %d: null sum", i)
		}
	}

	b.StopTimer()
	b.ReportMetric(float64(calibrationRows), "rows-aggregated")
}

// BenchmarkCalibration_DuckDB_FullScan measures an unfiltered full collection
// scan via ScanBackend.MapScan (Go-side decode of all rows). This captures the
// baseline read cost before pushdown optimizations.
func BenchmarkCalibration_DuckDB_FullScan(b *testing.B) {
	eng, err := duckdbengine.New("")
	if err != nil {
		b.Skipf("DuckDB not available: %v", err)
	}
	defer eng.Close()

	scan, ok := eng.(metaengine.ScanBackend)
	if !ok {
		b.Fatal("duckdb engine does not implement ScanBackend")
	}

	populateDuckDBEngine(b, eng, "fullscan", calibrationRows)

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		res, err := scan.MapScan(ctx, "fullscan", nil, nil, nil, 0)
		if err != nil {
			b.Fatalf("MapScan %d: %v", i, err)
		}

		if len(res.Items) != calibrationRows {
			b.Fatalf("MapScan %d: expected %d items, got %d", i, calibrationRows, len(res.Items))
		}
	}

	b.StopTimer()
	b.ReportMetric(float64(calibrationRows), "rows-scanned")
}

// populateDuckDBEngine seeds data through the engine's public MapSet API (the
// real write path). Used by benchmarks that exercise the engine's read methods.
func populateDuckDBEngine(tb testing.TB, eng metaengine.Engine, col string, n int) {
	tb.Helper()

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		tb.Fatal("engine does not implement MapBackend")
	}

	ctx := context.Background()

	for i := range n {
		status := "active"
		if i%2 == 0 {
			status = "inactive"
		}

		p := calibrationPayload{ID: i, Status: status, Amount: float64(i) * 1.5}

		if err := mb.MapSet(ctx, col, fmt.Sprintf("k%d", i), p); err != nil {
			tb.Fatalf("populate MapSet %d: %v", i, err)
		}
	}
}

// populateDuckDBRaw seeds data via a raw multi-VALUES INSERT (fast, for raw-SQL
// benchmarks that bypass the engine's read API).
func populateDuckDBRaw(ctx context.Context, db *sql.DB, col string, n int) error {
	const chunk = 500 // DuckDB parameter sanity.

	for start := 0; start < n; start += chunk {
		end := start + chunk
		if end > n {
			end = n
		}

		size := end - start
		rows := make([]string, size)
		args := make([]any, 0, size*3)

		for i := start; i < end; i++ {
			status := "active"
			if i%2 == 0 {
				status = "inactive"
			}

			raw, err := json.Marshal(calibrationPayload{
				ID: i, Status: status, Amount: float64(i) * 1.5,
			})
			if err != nil {
				return err
			}

			idx := (i - start) * 3
			rows[i-start] = fmt.Sprintf("($%d, $%d, $%d)", idx+1, idx+2, idx+3)
			args = append(args, col, fmt.Sprintf("k%d", i), string(raw))
		}

		q := fmt.Sprintf(
			`INSERT INTO meta_map (collection, key, value) VALUES %s
			 ON CONFLICT (collection, key) DO UPDATE SET value = excluded.value`,
			strings.Join(rows, ", "),
		)

		if _, err := db.ExecContext(ctx, q, args...); err != nil {
			return err
		}
	}

	return nil
}
