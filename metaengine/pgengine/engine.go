// Package pgengine provides a Postgres-backed metaengine Engine.
//
// Postgres is a mature relational database with JSONB support. This engine
// implements MapBackend, CounterBackend, and ScanBackend with Postgres-specific
// storage: JSONB columns for efficient JSON storage, UPSERT via ON CONFLICT,
// and native GROUP BY for counter aggregation.
//
// PushdownScan pushes filter/sort into Postgres WHERE/ORDER BY using JSONB
// operators (value->>'field'), avoiding full-table scans. LayoutPlanner
// creates expression indexes on those JSONB paths for B-tree performance.
//
// Pure Go (no CGo): uses the pgx driver via database/sql.
//
// Calibrated cost model (see calibration_bench_test.go for measurements):
// Point-lookup benchmarks (2026-08-03) measured ~33K ns/op (write) and
// ~28K ns/op (read) via Docker testcontainers — Docker network overhead
// inflates these 3-5x. The values below model a production connection
// (same-datacenter network or Unix socket).
//
// Additional batch/scan measurements (BenchmarkCalibration_Postgres_*, Docker):
//   - BatchInsert (1000-row multi-VALUES): ~3,375 ns/row (WAL fsync amortized)
//   - AggregateSum (10K-row SUM): ~149 ns/row (SQL-level aggregation)
//   - PushdownScan (10K filtered): ~402 ns/row (JSONB WHERE pushdown + decode)
//   - FullScan (10K unfiltered): ~805 ns/row (full scan + Go JSON decode)
//
// The scan per-row costs are lower than PG_NsPerRead because a single query
// amortizes setup across all rows; the constant models per-operation cost
// (dominated by the single point-lookup case).
//
//	PG_NsPerOp   = 12_000  (INSERT UPSERT with JSONB encode + WAL fsync)
//	PG_NsPerRead =  5_000  (indexed SELECT + JSONB decode + B-tree cache)
package pgengine

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	_ "github.com/jackc/pgx/v5/stdlib" // register the pgx database/sql driver

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// PG_NsPerOp is the calibrated per-write cost.
// Models production Postgres (WAL fsync + same-datacenter network round-trip).
// Docker testcontainer benchmarks measured ~33K ns/op (network overhead);
// batch writes amortize the fsync to ~3,375 ns/row (BenchmarkCalibration_Postgres_BatchInsert).
const PG_NsPerOp = 12000.0

// PG_NsPerRead is the calibrated per-read cost.
// Models production Postgres (B-tree index + buffer cache hit).
// Docker testcontainer benchmarks measured ~28K ns/op (network overhead).
// Scan workloads are cheaper per-row (~402-805 ns/row via
// BenchmarkCalibration_Postgres_PushdownScan/FullScan) because a single query
// amortizes setup; the constant governs the per-operation case (point lookups).
const PG_NsPerRead = 5000.0

// pgEngine implements metaengine.Engine with Postgres as the backend.
type pgEngine struct {
	db             *sql.DB
	mu             sync.Mutex
	done           bool
	layoutMu       sync.Mutex
	appliedLayouts map[string]bool
	cal            metaengine.Calibration
}

// New creates a Postgres-backed metaengine Engine from a DSN.
// The DSN must be a valid Postgres connection string
// (e.g. "postgres://user:pass@host:5432/db?sslmode=disable").
func New(dsn string) (metaengine.Engine, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("pgengine.New: open: %w", err)
	}

	eng := &pgEngine{db: db}

	if err := eng.init(); err != nil {
		_ = db.Close()

		return nil, err
	}

	return eng, nil
}

// NewFromDB wraps an existing *sql.DB connected to Postgres.
// The caller owns the DB lifecycle — Close is a no-op.
func NewFromDB(db *sql.DB) (metaengine.Engine, error) {
	eng := &pgEngine{db: db}

	if err := eng.init(); err != nil {
		return nil, err
	}

	return eng, nil
}

func (e *pgEngine) init() error {
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
		`CREATE TABLE IF NOT EXISTS meta_stream_log (
			seq BIGSERIAL PRIMARY KEY,
			collection TEXT NOT NULL,
			stream_id TEXT NOT NULL,
			value TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_stream_log_stream ON meta_stream_log(collection, stream_id, seq)`,
		`CREATE INDEX IF NOT EXISTS idx_stream_log_journal ON meta_stream_log(collection, seq)`,
	}

	for _, ddl := range ddls {
		if _, err := e.db.ExecContext(context.Background(), ddl); err != nil {
			return fmt.Errorf("pgengine.init: %w", err)
		}
	}

	return nil
}

// Profile returns the cost profile for this Postgres engine.
func (e *pgEngine) Profile() metaengine.EngineProfile {
	p := metaengine.EngineProfile{
		Name:        "postgres",
		NsPerOp:     PG_NsPerOp,
		NsPerRead:   PG_NsPerRead,
		Persistence: metaengine.PersistencePersistent, // remote server — always survives
		// Per-read-pattern calibrated costs (see calibration_bench_test.go).
		// Postgres has a real B-tree index on meta_map PK, so point lookups
		// are genuinely fast (unlike DuckDB's columnar scan). Scan/aggregation
		// per-row costs are lower than NsPerRead because a single query
		// amortizes connection setup across all rows.
		ReadCosts: metaengine.ReadCosts{
			// Indexed B-tree point lookup. Measured ~28K via Docker (inflated);
			// production (same-datacenter) ~5K. Matches PG_NsPerRead.
			NsPerPointLookup: 5_000,
			// Measured ~402 ns/row via Docker (BenchmarkCalibration_Postgres_PushdownScan).
			// JSONB WHERE pushdown + row decode.
			NsPerFilteredScan: 400,
			// Measured ~149 ns/row via Docker (BenchmarkCalibration_Postgres_AggregateSum).
			// SQL-level SUM with JSONB cast.
			NsPerAggregate: 150,
			// Measured ~805 ns/row via Docker (BenchmarkCalibration_Postgres_FullScan).
			// Full scan + Go-side JSON decode.
			NsPerScan: 800,
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
			metaengine.ADTMap:       metaengine.LayoutRow,
			metaengine.ADTCounter:   metaengine.LayoutRow,
			metaengine.ADTSortedMap: metaengine.LayoutRow,
		},
	}
	e.cal.ApplyCalibration(&p)

	return p
}

// SetCalibration implements metaengine.Calibratable.
func (e *pgEngine) SetCalibration(costs metaengine.CalibrationCosts) {
	e.cal.SetCalibration(costs)
}

// Close closes the underlying database. Safe to call multiple times.
func (e *pgEngine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.done {
		return nil
	}

	e.done = true

	if err := e.db.Close(); err != nil {
		return fmt.Errorf("Close: %w", err)
	}

	return nil
}

// --- MapBackend ---

func (e *pgEngine) MapSet(ctx context.Context, col string, key any, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("pgengine.MapSet: marshal: %w", err)
	}

	_, err = e.db.ExecContext(
		ctx,
		`INSERT INTO meta_map (collection, key, value)
		 VALUES ($1, $2, $3::jsonb)
		 ON CONFLICT (collection, key) DO UPDATE SET value = excluded.value`,
		col, fmt.Sprint(key), string(data),
	)
	if err != nil {
		return fmt.Errorf("pgengine.MapSet: %w", err)
	}

	return nil
}

func (e *pgEngine) MapGet(ctx context.Context, col string, key any) (any, bool, error) {
	var raw []byte

	err := e.db.QueryRowContext(
		ctx,
		`SELECT value::text FROM meta_map WHERE collection = $1 AND key = $2`,
		col, fmt.Sprint(key),
	).Scan(&raw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}

		return nil, false, fmt.Errorf("pgengine.MapGet: %w", err)
	}

	var val any
	if err := json.Unmarshal(raw, &val); err != nil {
		return nil, false, fmt.Errorf("pgengine.MapGet: unmarshal: %w", err)
	}

	return val, true, nil
}

func (e *pgEngine) MapDelete(ctx context.Context, col string, key any) error {
	_, err := e.db.ExecContext(
		ctx,
		`DELETE FROM meta_map WHERE collection = $1 AND key = $2`,
		col, fmt.Sprint(key),
	)
	if err != nil {
		return fmt.Errorf("pgengine.MapDelete: %w", err)
	}

	return nil
}

// --- CounterBackend ---

func (e *pgEngine) CounterIncrement(
	ctx context.Context,
	col string,
	deltas metaengine.Delta,
) error {
	if len(deltas) == 0 {
		return nil
	}

	keys := make([]string, 0, len(deltas))
	for key := range deltas {
		keys = append(keys, key)
	}

	sort.Strings(keys) // deterministic placeholder ordering

	placeholders := make([]string, len(keys))
	args := make([]any, 0, len(keys)*3)

	for i, key := range keys {
		base := i*3 + 1
		placeholders[i] = fmt.Sprintf("($%d, $%d, $%d)", base, base+1, base+2)
		args = append(args, col, key, deltas[key])
	}

	query := fmt.Sprintf(
		`INSERT INTO meta_counter (collection, key, value) VALUES %s
			 ON CONFLICT (collection, key) DO UPDATE SET value = meta_counter.value + excluded.value`,
		strings.Join(placeholders, ", "),
	)

	if _, err := e.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("pgengine.CounterIncrement: %w", err)
	}

	return nil
}

func (e *pgEngine) CounterGet(ctx context.Context, col string) (map[string]int64, error) {
	rows, err := e.db.QueryContext(
		ctx,
		`SELECT key, value FROM meta_counter WHERE collection = $1`,
		col,
	)
	if err != nil {
		return nil, fmt.Errorf("pgengine.CounterGet: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string]int64)

	for rows.Next() {
		var key string

		var val int64

		if err := rows.Scan(&key, &val); err != nil {
			return nil, fmt.Errorf("pgengine.CounterGet: scan: %w", err)
		}

		result[key] = val
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("CounterGet: %w", err)
	}

	return result, nil
}

// Compile-time assertions.
var (
	_ metaengine.Engine           = (*pgEngine)(nil)
	_ metaengine.MapBackend       = (*pgEngine)(nil)
	_ metaengine.CounterBackend   = (*pgEngine)(nil)
	_ metaengine.ScanBackend      = (*pgEngine)(nil)
	_ metaengine.PushdownScan     = (*pgEngine)(nil)
	_ metaengine.LayoutPlanner    = (*pgEngine)(nil)
	_ metaengine.StreamLogBackend = (*pgEngine)(nil)
)
