package duckdbengine

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"sync"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// DuckDBNsPerOp is the calibrated per-write-operation cost.
// DuckDB's columnar writes are amortized across batches; individual inserts
// pay the JSON encoding + columnar flush cost. Point-lookup benchmarks
// (BenchmarkDuckDB_MapSet, ~3.9M ns/op) are NOT representative — they measure
// DuckDB's worst case. This value models batch-amortized columnar writes.
//
// Measured (BenchmarkCalibration_DuckDB_BatchInsert, 1000-row multi-VALUES
// INSERT, AMD Ryzen dev machine): ~8,950 ns/row. The constant adds a 1.7x
// conservative margin for slower hardware and larger payloads.
// Re-run the benchmark on target hardware before trusting absolute estimates.
const DuckDBNsPerOp = 15000.0

// DuckDBNsPerRead is the calibrated per-read cost for scans and aggregations —
// DuckDB's intended OLAP workload. Its vectorized execution engine makes
// aggregations (GROUP BY, SUM) extremely fast, often 10-50x faster than
// row-oriented SQLite on analytical workloads at scale.
// Point-lookup benchmarks (BenchmarkDuckDB_MapGet, ~546K ns/op) measure
// full column scans for a single key, which is NOT the intended use case.
//
// Measured (10K rows, AMD Ryzen dev machine):
//   - BenchmarkCalibration_DuckDB_AggregateSum: ~111 ns/row (vectorized SUM)
//   - BenchmarkCalibration_DuckDB_PushdownScan: ~425 ns/row (filtered scan + JSON decode)
//   - BenchmarkCalibration_DuckDB_FullScan:     ~810 ns/row (full scan + JSON decode)
//
// The constant (1.5x the full-scan measurement) is conservative for slower
// hardware; it also leaves headroom for DuckDB's vectorized advantage to grow
// at 1M+ rows where columnar compression and zone maps dominate.
const DuckDBNsPerRead = 1200.0

// duckdbEngine implements metaengine.Engine with DuckDB as the backend.
type duckdbEngine struct {
	db          *sql.DB
	persistence metaengine.Persistence
	mu          sync.Mutex
	took        bool // closed flag
	plans       map[string]metaengine.LayoutPlan
	layoutMu    sync.Mutex
}

// New creates a DuckDB-backed metaengine Engine.
// dsn="" creates an in-memory database; a file path creates a persistent one.
//
// Requires CGo and the DuckDB C++ runtime (statically linked).
func New(dsn string) (metaengine.Engine, error) {
	persistence := metaengine.PersistencePersistent
	if dsn == "" {
		dsn = ":memory:"
		persistence = metaengine.PersistenceVolatile
	}

	db, err := sql.Open("duckdb", dsn)
	if err != nil {
		return nil, fmt.Errorf("duckdbengine.New: open %q: %w", dsn, err)
	}

	eng := &duckdbEngine{db: db, persistence: persistence}

	if err := eng.init(); err != nil {
		_ = db.Close()

		return nil, err
	}

	return eng, nil
}

// NewFromDB wraps an existing *sql.DB connected to DuckDB.
// The caller owns the DB lifecycle — Close is a no-op.
func NewFromDB(db *sql.DB) (metaengine.Engine, error) {
	eng := &duckdbEngine{db: db, persistence: metaengine.PersistencePersistent}

	if err := eng.init(); err != nil {
		return nil, err
	}

	return eng, nil
}

func (e *duckdbEngine) init() error {
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
		if _, err := e.db.ExecContext(context.Background(), ddl); err != nil {
			return fmt.Errorf("duckdbengine.init: %w", err)
		}
	}

	return nil
}

// Profile returns the cost profile for this DuckDB engine.
func (e *duckdbEngine) Profile() metaengine.EngineProfile {
	return metaengine.EngineProfile{
		Name:        "duckdb",
		NsPerOp:     DuckDBNsPerOp,
		NsPerRead:   DuckDBNsPerRead,
		Persistence: e.persistence,
		// Per-read-pattern calibrated costs (see calibration_bench_test.go).
		// DuckDB's read operations span 4000x: a point lookup (full PK scan +
		// JSON decode via database/sql) is ~500x slower than a vectorized
		// aggregation. Without ReadCosts, the planner uses NsPerRead for all
		// patterns, overestimating scans and underestimating point lookups.
		ReadCosts: metaengine.ReadCosts{
			// Measured ~546K ns (BenchmarkDuckDB_MapGet). Set to a conservative
			// 50K: DuckDB is NOT a point-lookup engine, but the PK index on
			// meta_map avoids a full scan. The 50K still makes the planner
			// prefer Memory (500ns) for point lookups by 100x.
			NsPerPointLookup: 50_000,
			// Measured ~454 ns/row (BenchmarkCalibration_DuckDB_PushdownScan).
			// json_extract WHERE pushdown + vectorized scan.
			NsPerFilteredScan: 450,
			// Measured ~133 ns/row (BenchmarkCalibration_DuckDB_AggregateSum).
			// Vectorized SUM — DuckDB's killer feature.
			NsPerAggregate: 150,
			// Measured ~975 ns/row (BenchmarkCalibration_DuckDB_FullScan).
			// Full scan + Go-side JSON decode of all rows.
			NsPerScan: 1_000,
		},
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap:       metaengine.ComplexityOLogN,
			metaengine.ADTCounter:   metaengine.ComplexityO1,
			metaengine.ADTSortedMap: metaengine.ComplexityOLogN,
			metaengine.ADTSet:       metaengine.ComplexityON,
			metaengine.ADTGraph:     metaengine.ComplexityON,
			metaengine.ADTLog:       metaengine.ComplexityON,
			metaengine.ADTMultimap:  metaengine.ComplexityON,
			metaengine.ADTVector:    metaengine.ComplexityON,
			metaengine.ADTSearch:    metaengine.ComplexityON,
			metaengine.ADTSpatial:   metaengine.ComplexityON,
		},
		DegradedADTs: map[metaengine.ADT]bool{
			metaengine.ADTSet:      true,
			metaengine.ADTGraph:    true,
			metaengine.ADTLog:      true,
			metaengine.ADTMultimap: true,
			metaengine.ADTVector:   true,
			metaengine.ADTSearch:   true,
			metaengine.ADTSpatial:  true,
		},
		Layouts: map[metaengine.ADT]metaengine.StorageLayout{
			metaengine.ADTMap:       metaengine.LayoutColumnar,
			metaengine.ADTCounter:   metaengine.LayoutColumnar,
			metaengine.ADTSortedMap: metaengine.LayoutColumnar,
		},
	}
}

// Close closes the underlying database. Safe to call multiple times.
func (e *duckdbEngine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.took {
		return nil
	}

	e.took = true

	if err := e.db.Close(); err != nil {
		return fmt.Errorf("Close: %w", err)
	}

	return nil
}

// --- MapBackend ---

func (e *duckdbEngine) MapSet(ctx context.Context, col string, key any, value any) error {
	if plan, ok := e.plans[col]; ok {
		return e.mapSetPlanned(ctx, plan, key, value)
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("duckdbengine.MapSet: marshal value: %w", err)
	}

	_, err = e.db.ExecContext(
		ctx,
		`INSERT INTO meta_map (collection, key, value)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (collection, key) DO UPDATE SET value = excluded.value`,
		col, fmt.Sprint(key), string(data),
	)
	if err != nil {
		return fmt.Errorf("duckdbengine.MapSet: %w", err)
	}

	return nil
}

func (e *duckdbEngine) MapGet(ctx context.Context, col string, key any) (any, bool, error) {
	if plan, ok := e.plans[col]; ok {
		return e.mapGetPlanned(ctx, plan, key)
	}

	var raw string

	err := e.db.QueryRowContext(
		ctx,
		`SELECT value FROM meta_map WHERE collection = $1 AND key = $2`,
		col, fmt.Sprint(key),
	).Scan(&raw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}

		return nil, false, fmt.Errorf("duckdbengine.MapGet: %w", err)
	}

	var val any
	if err := json.Unmarshal([]byte(raw), &val); err != nil {
		return nil, false, fmt.Errorf("duckdbengine.MapGet: unmarshal: %w", err)
	}

	return val, true, nil
}

func (e *duckdbEngine) MapDelete(ctx context.Context, col string, key any) error {
	if plan, ok := e.plans[col]; ok {
		return e.mapDeletePlanned(ctx, plan, key)
	}

	_, err := e.db.ExecContext(
		ctx,
		`DELETE FROM meta_map WHERE collection = $1 AND key = $2`,
		col, fmt.Sprint(key),
	)
	if err != nil {
		return fmt.Errorf("duckdbengine.MapDelete: %w", err)
	}

	return nil
}

// --- CounterBackend ---
//
// CounterGet retrieves all counter values for a collection via a single
// SELECT pass. DuckDB's columnar storage makes this efficient for
// analytical workloads.

func (e *duckdbEngine) CounterIncrement(
	ctx context.Context,
	col string,
	deltas metaengine.Delta,
) error {
	// Increment each delta individually. DuckDB's ON CONFLICT requires
	// per-row upsert — multi-row VALUES with ON CONFLICT is not supported
	// in the same way as Postgres. Each upsert is still fast due to
	// DuckDB's vectorized execution engine.
	for key, delta := range deltas {
		_, err := e.db.ExecContext(
			ctx,
			`INSERT INTO meta_counter (collection, key, value)
			 VALUES ($1, $2, $3)
			 ON CONFLICT (collection, key) DO UPDATE SET value = meta_counter.value + excluded.value`,
			col, key, delta,
		)
		if err != nil {
			return fmt.Errorf("duckdbengine.CounterIncrement: %w", err)
		}
	}

	return nil
}

func (e *duckdbEngine) CounterGet(ctx context.Context, col string) (map[string]int64, error) {
	rows, err := e.db.QueryContext(
		ctx,
		`SELECT key, value FROM meta_counter WHERE collection = $1`,
		col,
	)
	if err != nil {
		return nil, fmt.Errorf("duckdbengine.CounterGet: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string]int64)

	for rows.Next() {
		var key string

		var val int64

		if err := rows.Scan(&key, &val); err != nil {
			return nil, fmt.Errorf("duckdbengine.CounterGet: scan: %w", err)
		}

		result[key] = val
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("CounterGet: %w", err)
	}

	return result, nil
}

// Compile-time assertions that duckdbEngine implements the interfaces.
var (
	_ metaengine.Engine            = (*duckdbEngine)(nil)
	_ metaengine.MapBackend        = (*duckdbEngine)(nil)
	_ metaengine.CounterBackend    = (*duckdbEngine)(nil)
	_ metaengine.ScanBackend       = (*duckdbEngine)(nil)
	_ metaengine.PushdownScan      = (*duckdbEngine)(nil)
	_ metaengine.LayoutPlanner     = (*duckdbEngine)(nil)
	_ metaengine.LayoutPlanApplier = (*duckdbEngine)(nil)
)
