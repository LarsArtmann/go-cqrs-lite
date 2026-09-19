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
	"sync/atomic"

	_ "github.com/jackc/pgx/v5/stdlib" // register the pgx database/sql driver

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/claimkit"
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
	metaengine.Calibration

	// ADR-0142 write-side capabilities: the shared claimkit SQL runtime,
	// attached in init (dueclaim.go) — DueClaimer + FactSink + DedupStore by
	// promotion.
	*claimkit.Claims
	*claimkit.Dedup

	db             *sql.DB
	mu             sync.Mutex
	activeTx       atomic.Pointer[sql.Tx] // non-nil inside RunInTx
	done           bool
	layoutMu       sync.Mutex
	appliedLayouts map[string]bool
	plans          map[string]metaengine.LayoutPlan // collection → planned-table layout (D1; guarded by layoutMu)
	copyMin        int                              // WithCopyAppend: bulk StreamAppend threshold; 0 = off
	durability     metaengine.DurabilityTier        // set by the driver factory (withDurabilityTier)
}

// New creates a Postgres-backed metaengine Engine from a DSN.
// The DSN must be a valid Postgres connection string
// (e.g. "postgres://user:pass@host:5432/db?sslmode=disable").
func New(dsn string, opts ...Option) (metaengine.Engine, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("pgengine.New: open: %w", err)
	}

	eng := &pgEngine{db: db}
	for _, opt := range opts {
		opt(eng)
	}

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
		`CREATE TABLE IF NOT EXISTS meta_graph_edges (
			collection TEXT NOT NULL,
			from_node TEXT NOT NULL,
			to_node TEXT NOT NULL,
			PRIMARY KEY (collection, from_node, to_node)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_graph_edges_from ON meta_graph_edges(collection, from_node)`,
		`CREATE INDEX IF NOT EXISTS idx_graph_edges_to ON meta_graph_edges(collection, to_node)`,
		`CREATE TABLE IF NOT EXISTS meta_vector (
			collection TEXT NOT NULL,
			id TEXT NOT NULL,
			vector JSONB NOT NULL,
			metadata JSONB,
			PRIMARY KEY (collection, id)
		)`,
	}

	//art-dupl:accept DDL-apply loop idiom; each dialect owns its own DDL list
	for _, ddl := range ddls {
		if _, err := e.db.ExecContext(context.Background(), ddl); err != nil {
			return fmt.Errorf("pgengine.init: %w", err)
		}
	}

	if err := e.wireClaimkit(); err != nil {
		return err
	}

	return nil
}

// Close closes the underlying database. Safe to call multiple times.
func (e *pgEngine) Close() error {
	//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
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

// HealthCheck pings the underlying database to verify connectivity.
// Implements [metaengine.HealthChecker] for Kubernetes-style liveness probes.
func (e *pgEngine) HealthCheck(ctx context.Context) error {
	return e.db.PingContext(ctx)
}

// --- MapBackend ---

func (e *pgEngine) MapSet(ctx context.Context, col string, key any, value any) error {
	//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
	if plan, ok := e.planFor(col); ok {
		return e.mapSetPlanned(ctx, plan, key, value)
	}

	//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("pgengine.MapSet: marshal: %w", err)
	}

	_, err = e.conn().ExecContext(
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
	if plan, ok := e.planFor(col); ok {
		return e.mapGetPlanned(ctx, plan, key)
	}

	var raw []byte

	err := e.conn().QueryRowContext(
		ctx,
		//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
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
	if plan, ok := e.planFor(col); ok {
		return e.mapDeletePlanned(ctx, plan, key)
	}

	//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
	_, err := e.conn().ExecContext(
		ctx,
		//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
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
	//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
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

	if _, err := e.conn().ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("pgengine.CounterIncrement: %w", err)
	}

	return nil
}

func (e *pgEngine) CounterGet(ctx context.Context, col string) (map[string]int64, error) {
	rows, err := e.conn().QueryContext(
		ctx,
		`SELECT key, value FROM meta_counter WHERE collection = $1`,
		col,
	)
	if err != nil {
		return nil, fmt.Errorf("pgengine.CounterGet: %w", err)
	}
	//art-dupl:accept cross-module SQL engine pattern — separate go.mod
	defer metaengine.DeferClose(rows)

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
	_ metaengine.Engine               = (*pgEngine)(nil)
	_ metaengine.MapBackend           = (*pgEngine)(nil)
	_ metaengine.CounterBackend       = (*pgEngine)(nil)
	_ metaengine.ScanBackend          = (*pgEngine)(nil)
	_ metaengine.PushdownScan         = (*pgEngine)(nil)
	_ metaengine.LayoutPlanner        = (*pgEngine)(nil)
	_ metaengine.StreamLogBackend     = (*pgEngine)(nil)
	_ metaengine.SeqSeekableStreamLog = (*pgEngine)(nil)
	_ metaengine.AtomicAppender       = (*pgEngine)(nil)
	_ metaengine.Transactional        = (*pgEngine)(nil)
	_ metaengine.Calibratable         = (*pgEngine)(nil)
	_ metaengine.TrackerHost          = (*pgEngine)(nil)
	_ metaengine.Prober               = (*pgEngine)(nil)
	_ metaengine.TransactMeasurer     = (*pgEngine)(nil)
)
