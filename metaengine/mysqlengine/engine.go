// Package mysqlengine provides a MySQL-backed metaengine Engine.
//
// MySQL is a mature relational database with JSON support (MySQL 5.7+). This
// engine implements MapBackend, CounterBackend, ScanBackend, and
// StreamLogBackend with MySQL-specific storage: JSON columns for efficient JSON
// storage, UPSERT via ON DUPLICATE KEY UPDATE, and indexed AUTO_INCREMENT for
// stream log sequences.
//
// PushdownScan pushes filter/sort into MySQL WHERE/ORDER BY using JSON
// operators (value->'$.field'), avoiding full-table scans.
//
// Pure Go (no CGo): uses the go-sql-driver/mysql driver via database/sql.
//
// Key MySQL dialect differences from Postgres (pgengine):
//   - Placeholders: ? (not $1, $2)
//   - UPSERT: INSERT ... ON DUPLICATE KEY UPDATE (not ON CONFLICT DO UPDATE)
//   - JSON access: value->'$.field' (not value->'field')
//   - Text columns: VARCHAR(255) for PRIMARY KEY (MySQL can't index TEXT)
//   - Reserved word `key` escaped with backticks
package mysqlengine

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"

	_ "github.com/go-sql-driver/mysql" // register the MySQL database/sql driver

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// MySQLNsPerOp is the estimated per-write cost.
// Models production MySQL (InnoDB WAL fsync + same-datacenter network round-trip).
const MySQLNsPerOp = 12000.0

// MySQLNsPerRead is the estimated per-read cost.
// Models production MySQL (InnoDB B-tree index + buffer pool cache hit).
const MySQLNsPerRead = 5000.0

// mysqlEngine implements metaengine.Engine with MySQL as the backend.
var _ metaengine.TrackerHost = (*mysqlEngine)(nil)

type mysqlEngine struct {
	db             *sql.DB
	mu             sync.Mutex
	activeTx       atomic.Pointer[sql.Tx] // non-nil inside RunInTx
	done           bool
	layoutMu       sync.Mutex
	appliedLayouts map[string]bool
	metaengine.Calibration
}

// New creates a MySQL-backed metaengine Engine from a DSN.
// The DSN must be a valid MySQL connection string
// (e.g. "user:pass@tcp(host:3306)/dbname?parseTime=true").
func New(dsn string) (metaengine.Engine, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("mysqlengine.New: open: %w", err)
	}

	eng := &mysqlEngine{db: db}

	if err := eng.init(); err != nil {
		_ = db.Close()

		return nil, err
	}

	return eng, nil
}

// NewFromDB wraps an existing *sql.DB connected to MySQL.
// The caller owns the DB lifecycle — Close is a no-op.
func NewFromDB(db *sql.DB) (metaengine.Engine, error) {
	eng := &mysqlEngine{db: db}

	if err := eng.init(); err != nil {
		return nil, err
	}

	return eng, nil
}

func (e *mysqlEngine) init() error {
	ddls := []string{
		`CREATE TABLE IF NOT EXISTS meta_map (
			collection VARCHAR(255) NOT NULL,
			` + "`key`" + ` VARCHAR(255) NOT NULL,
			value JSON NOT NULL,
			PRIMARY KEY (collection, ` + "`key`" + `)
		)`,
		`CREATE TABLE IF NOT EXISTS meta_counter (
			collection VARCHAR(255) NOT NULL,
			` + "`key`" + ` VARCHAR(255) NOT NULL,
			value BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY (collection, ` + "`key`" + `)
		)`,
		`CREATE TABLE IF NOT EXISTS meta_stream_log (
			seq BIGINT AUTO_INCREMENT PRIMARY KEY,
			collection VARCHAR(255) NOT NULL,
			stream_id VARCHAR(255) NOT NULL,
			value TEXT NOT NULL,
			INDEX idx_stream_log_stream (collection, stream_id, seq),
			INDEX idx_stream_log_journal (collection, seq)
		)`,
	}

	for _, ddl := range ddls {
		if _, err := e.db.ExecContext(context.Background(), ddl); err != nil {
			return fmt.Errorf("mysqlengine.init: %w", err)
		}
	}

	return nil
}

// Profile returns the cost profile for this MySQL engine.
func (e *mysqlEngine) Profile() metaengine.EngineProfile {
	p := metaengine.EngineProfile{
		Name:        "mysql",
		NsPerOp:     MySQLNsPerOp,
		NsPerRead:   MySQLNsPerRead,
		Persistence: metaengine.PersistencePersistent,
		// MySQL is a networked service. RequiresNetwork declares the structural
		// fact; NetworkRTT is a same-datacenter PRIOR replaced by a live probe
		// (SELECT 1) once ProbeEngine runs. See METAENGINE-LIVE-LATENCY-MODEL.md.
		RequiresNetwork: true,
		NetworkRTT:      MySQL_NetworkRTT,
		ReadCosts: metaengine.ReadCosts{
			NsPerPointLookup:  5_000,
			NsPerFilteredScan: 400,
			NsPerAggregate:    150,
			NsPerScan:         800,
		},
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap:       metaengine.ComplexityOLogN,
			metaengine.ADTCounter:   metaengine.ComplexityO1,
			metaengine.ADTSortedMap: metaengine.ComplexityOLogN,
			metaengine.ADTSet:       metaengine.ComplexityON,
			metaengine.ADTLog:       metaengine.ComplexityON,
			metaengine.ADTMultimap:  metaengine.ComplexityON,
		},
		DegradedADTs: map[metaengine.ADT]bool{
			metaengine.ADTSet:      true,
			metaengine.ADTLog:      true,
			metaengine.ADTMultimap: true,
		},
		Layouts: map[metaengine.ADT]metaengine.StorageLayout{
			metaengine.ADTMap:       metaengine.LayoutRow,
			metaengine.ADTCounter:   metaengine.LayoutRow,
			metaengine.ADTSortedMap: metaengine.LayoutRow,
		},
	}
	e.ApplyCalibration(&p)

	return p
}

// Close closes the underlying database. Safe to call multiple times.
func (e *mysqlEngine) Close() error {
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
func (e *mysqlEngine) HealthCheck(ctx context.Context) error {
	return e.db.PingContext(ctx)
}

// conn returns the active transaction if RunInTx is in progress, otherwise
// the engine's *sql.DB.
// art-dupl:accept cross-module SQL engine pattern — separate go.mod
func (e *mysqlEngine) conn() metaengine.SQLExec {
	if tx := e.activeTx.Load(); tx != nil {
		return tx
	}

	return e.db
}

// inTx runs fn in a transaction. If RunInTx is in progress, fn participates
// in the outer transaction. Otherwise a new transaction is started and
// committed (or rolled back on error).
// art-dupl:accept cross-module SQL engine pattern — separate go.mod
func (e *mysqlEngine) inTx(ctx context.Context, fn func(metaengine.SQLExec) error) error {
	if tx := e.activeTx.Load(); tx != nil {
		return fn(tx)
	}

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("mysqlengine: begin tx: %w", err)
	}

	fnErr := fn(tx)

	if fnErr != nil {
		_ = tx.Rollback()

		return fnErr
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("mysqlengine: commit tx: %w", err)
	}

	return nil
}

// Compile-time assertions.
var (
	_ metaengine.Engine           = (*mysqlEngine)(nil)
	_ metaengine.MapBackend       = (*mysqlEngine)(nil)
	_ metaengine.CounterBackend   = (*mysqlEngine)(nil)
	_ metaengine.ScanBackend      = (*mysqlEngine)(nil)
	_ metaengine.PushdownScan     = (*mysqlEngine)(nil)
	_ metaengine.LayoutPlanner    = (*mysqlEngine)(nil)
	_ metaengine.StreamLogBackend = (*mysqlEngine)(nil)
	_ metaengine.AtomicAppender   = (*mysqlEngine)(nil)
	_ metaengine.Transactional    = (*mysqlEngine)(nil)
)
