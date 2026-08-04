package pgengine_test

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"
	"strings"
	"testing"

	pgengine "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// calibration_bench_test.go measures the SQL workloads that the Postgres cost
// constants (PG_NsPerOp, PG_NsPerRead) actually model.
//
// The existing bench_test.go measures single-statement point lookups
// (MapSet/MapGet), which include per-statement WAL fsync + JSONB encode
// overhead. Those are valid worst-case measurements, but they do not represent
// the batch write or scan/aggregation workloads the cost model estimates.
//
// This file adds four calibration benchmarks:
//
//   - PG_NsPerOp  → batch write (BenchmarkCalibration_Postgres_BatchInsert)
//   - PG_NsPerRead → vectorized scan + aggregation (PushdownScan, AggregateSum, FullScan)
//
// Run: GOWORK=off go test -run='^$' -bench='BenchmarkCalibration_Postgres' -benchmem ./...
//
// Docker network note: testcontainers connect over the Docker bridge network,
// which adds ~0.2-0.5ms RTT per query. Production Postgres (same-datacenter or
// Unix socket) is 3-5x faster. The constants model production, not Docker.

const pgCalibrationRows = 10_000

type pgCalibrationPayload struct {
	ID     int     `json:"id"`
	Status string  `json:"status"`
	Amount float64 `json:"amount"`
}

// openPGForBench opens a raw *sql.DB to the test Postgres instance and creates
// the engine's schema. Uses the same DSN resolution (testcontainers or
// POSTGRES_TEST_DSN) as the rest of the package. Returns a DB the caller owns.
func openPGForBench(t testing.TB) *sql.DB {
	t.Helper()

	dsn := pgDSN(t)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open pgx: %v", err)
	}

	ddls := []string{
		`CREATE TABLE IF NOT EXISTS meta_map (
			collection TEXT NOT NULL,
			key TEXT NOT NULL,
			value JSONB NOT NULL,
			PRIMARY KEY (collection, key)
		)`,
		`CREATE TABLE IF NOT EXISTS meta_counter (
			collection TEXT NOT NULL,
			key TEXT NOT NULL,
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

// populatePGEngine seeds data through the engine's public MapSet API.
func populatePGEngine(tb testing.TB, eng metaengine.Engine, col string, n int) {
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

		if err := mb.MapSet(ctx, col, fmt.Sprintf("k%d", i), pgCalibrationPayload{
			ID: i, Status: status, Amount: float64(i) * 1.5,
		}); err != nil {
			tb.Fatalf("populate MapSet %d: %v", i, err)
		}
	}
}

// populatePGRaw seeds data via a raw multi-VALUES INSERT (fast, for raw-SQL
// benchmarks). Chunks to respect Postgres parameter limits.
func populatePGRaw(ctx context.Context, db *sql.DB, col string, n int) error {
	const chunk = 500

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

			raw, err := json.Marshal(pgCalibrationPayload{
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

// BenchmarkCalibration_Postgres_BatchInsert measures batch-amortized writes: a
// single multi-VALUES INSERT of 1000 rows. This calibrates PG_NsPerOp — the
// per-row cost when the WAL fsync is amortized across a batch.
// The "ns/row" metric is the per-write constant.
//
// Contrast with BenchmarkPostgres_MapSet (bench_test.go) which measures
// single-statement autocommit writes (per-statement fsync overhead).
func BenchmarkCalibration_Postgres_BatchInsert(b *testing.B) {
	const batchSize = 1000

	db := openPGForBench(b)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// Pre-build payloads (marshal is not part of the measured write cost).
	type row struct{ col, key, val string }
	batch := make([]row, batchSize)

	for i := range batchSize {
		raw, err := json.Marshal(pgCalibrationPayload{
			ID: i, Status: "active", Amount: float64(i) * 1.5,
		})
		if err != nil {
			b.Fatalf("marshal: %v", err)
		}

		batch[i] = row{col: "batch", key: fmt.Sprintf("k%d", i), val: string(raw)}
	}

	placeholders := make([]string, batchSize)
	args := make([]any, 0, batchSize*3)

	for i, r := range batch {
		placeholders[i] = fmt.Sprintf("($%d, $%d, $%d)", i*3+1, i*3+2, i*3+3)
		args = append(args, r.col, r.key, r.val)
	}

	q := "INSERT INTO meta_map (collection, key, value) VALUES " + strings.Join(placeholders, ", ")

	b.ResetTimer()

	for iter := 0; iter < b.N; iter++ {
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

// BenchmarkCalibration_Postgres_PushdownScan measures a filtered scan with
// JSONB WHERE pushdown (PushdownMapScan). ~50% of rows match. This calibrates
// PG_NsPerRead for the scan workload. The "ns/row" metric is the per-read cost.
func BenchmarkCalibration_Postgres_PushdownScan(b *testing.B) {
	dsn := pgDSN(b)

	eng, err := pgengine.New(dsn)
	if err != nil {
		b.Skipf("Postgres not available: %v", err)
	}
	defer eng.Close()

	pushdown, ok := eng.(metaengine.PushdownScan)
	if !ok {
		b.Fatal("postgres engine does not implement PushdownScan")
	}

	populatePGEngine(b, eng, "scan", pgCalibrationRows)

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
	b.ReportMetric(float64(pgCalibrationRows), "rows-scanned")
}

// BenchmarkCalibration_Postgres_AggregateSum measures SQL-level aggregation:
// SUM(amount) over pgCalibrationRows via JSONB extraction. This calibrates
// PG_NsPerRead for the analytical aggregation workload. The "ns/row" metric is
// the per-read cost.
func BenchmarkCalibration_Postgres_AggregateSum(b *testing.B) {
	db := openPGForBench(b)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	if err := populatePGRaw(ctx, db, "agg", pgCalibrationRows); err != nil {
		b.Fatalf("populate: %v", err)
	}

	const q = `SELECT SUM((value->>'amount')::numeric)
	           FROM meta_map WHERE collection = $1`

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
	b.ReportMetric(float64(pgCalibrationRows), "rows-aggregated")
}

// BenchmarkCalibration_Postgres_FullScan measures an unfiltered full collection
// scan via ScanBackend.MapScan (Go-side decode of all rows). This captures the
// baseline read cost before pushdown optimizations.
func BenchmarkCalibration_Postgres_FullScan(b *testing.B) {
	dsn := pgDSN(b)

	eng, err := pgengine.New(dsn)
	if err != nil {
		b.Skipf("Postgres not available: %v", err)
	}
	defer eng.Close()

	scan, ok := eng.(metaengine.ScanBackend)
	if !ok {
		b.Fatal("postgres engine does not implement ScanBackend")
	}

	populatePGEngine(b, eng, "fullscan", pgCalibrationRows)

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		res, err := scan.MapScan(ctx, "fullscan", nil, nil, nil, 0)
		if err != nil {
			b.Fatalf("MapScan %d: %v", i, err)
		}

		if len(res.Items) != pgCalibrationRows {
			b.Fatalf("MapScan %d: expected %d items, got %d", i, pgCalibrationRows, len(res.Items))
		}
	}

	b.StopTimer()
	b.ReportMetric(float64(pgCalibrationRows), "rows-scanned")
}
