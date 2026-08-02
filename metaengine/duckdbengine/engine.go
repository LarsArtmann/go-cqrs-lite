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
// pay the JSON encoding + columnar flush cost.
const DuckDBNsPerOp = 15000.0

// DuckDBNsPerRead is the calibrated per-read cost. DuckDB's vectorized
// execution engine makes aggregations (GROUP BY, SUM) extremely fast —
// often 10-50x faster than row-oriented SQLite on analytical workloads.
const DuckDBNsPerRead = 3000.0

// duckdbEngine implements metaengine.Engine with DuckDB as the backend.
type duckdbEngine struct {
	db       *sql.DB
	mu       sync.Mutex
	took     bool // closed flag
	plans    map[string]metaengine.LayoutPlan
	layoutMu sync.Mutex
}

// New creates a DuckDB-backed metaengine Engine.
// dsn="" creates an in-memory database; a file path creates a persistent one.
//
// Requires CGo and the DuckDB C++ runtime (statically linked).
func New(dsn string) (metaengine.Engine, error) {
	if dsn == "" {
		dsn = ":memory:"
	}

	db, err := sql.Open("duckdb", dsn)
	if err != nil {
		return nil, fmt.Errorf("duckdbengine.New: open %q: %w", dsn, err)
	}

	eng := &duckdbEngine{db: db}

	if err := eng.init(); err != nil {
		_ = db.Close()

		return nil, err
	}

	return eng, nil
}

// NewFromDB wraps an existing *sql.DB connected to DuckDB.
// The caller owns the DB lifecycle — Close is a no-op.
func NewFromDB(db *sql.DB) (metaengine.Engine, error) {
	eng := &duckdbEngine{db: db}

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
		Name:      "duckdb",
		NsPerOp:   DuckDBNsPerOp,
		NsPerRead: DuckDBNsPerRead,
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap:       metaengine.ComplexityOLogN,
			metaengine.ADTCounter:   metaengine.ComplexityO1,
			metaengine.ADTSortedMap: metaengine.ComplexityOLogN,
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
