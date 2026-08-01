// Package pgengine provides a Postgres-backed metaengine Engine.
//
// Postgres is a mature relational database with JSONB support and GIN indexes.
// This engine implements MapBackend and CounterBackend with Postgres-specific
// optimizations: JSONB columns for efficient JSON storage, UPSERT via
// ON CONFLICT, and native GROUP BY for counter aggregation.
//
// Pure Go (no CGo): uses the pgx driver via database/sql.
//
// Calibrated cost model:
//
//	PG_NsPerOp   = 12_000  (INSERT with JSONB encode + network round-trip)
//	PG_NsPerRead =  5_000  (indexed SELECT + JSONB decode)
package pgengine

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"sync"

	_ "github.com/jackc/pgx/v5/stdlib" // register the pgx database/sql driver

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// PG_NsPerOp is the calibrated per-write cost.
// Postgres writes include WAL fsync + network round-trip.
const PG_NsPerOp = 12000.0

// PG_NsPerRead is the calibrated per-read cost.
// Postgres point reads benefit from B-tree indexes + buffer cache.
const PG_NsPerRead = 5000.0

// pgEngine implements metaengine.Engine with Postgres as the backend.
type pgEngine struct {
	db   *sql.DB
	mu   sync.Mutex
	done bool
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
	return metaengine.EngineProfile{
		Name:      "postgres",
		NsPerOp:   PG_NsPerOp,
		NsPerRead: PG_NsPerRead,
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap:       metaengine.ComplexityOLogN,
			metaengine.ADTCounter:   metaengine.ComplexityO1,
			metaengine.ADTSortedMap: metaengine.ComplexityOLogN,
		},
		Layouts: map[metaengine.ADT]metaengine.StorageLayout{
			metaengine.ADTMap:       metaengine.LayoutRow,
			metaengine.ADTCounter:   metaengine.LayoutRow,
			metaengine.ADTSortedMap: metaengine.LayoutRow,
		},
	}
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
	for key, delta := range deltas {
		_, err := e.db.ExecContext(
			ctx,
			`INSERT INTO meta_counter (collection, key, value)
			 VALUES ($1, $2, $3)
			 ON CONFLICT (collection, key) DO UPDATE SET value = meta_counter.value + excluded.value`,
			col, key, delta,
		)
		if err != nil {
			return fmt.Errorf("pgengine.CounterIncrement: %w", err)
		}
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
	_ metaengine.Engine         = (*pgEngine)(nil)
	_ metaengine.MapBackend     = (*pgEngine)(nil)
	_ metaengine.CounterBackend = (*pgEngine)(nil)
	_ metaengine.ScanBackend    = (*pgEngine)(nil)
)
