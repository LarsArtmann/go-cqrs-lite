package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/claiming/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/claimkit"
)

// Engine exposes the queue database as a metaengine engine (ADR-0142):
// metaengine.DueClaimer, metaengine.FactSink, and metaengine.DedupStore via
// the ONE shared claimkit runtime, pointed at the SAME database the tasks
// live in — timers, dedup keys, and task claims share one database and one
// transaction domain. The queue's own task tables and their richer claim
// semantics (priority+aging, dependency gating) remain engine-owned
// (queue.Store[T]); this surface is the generic claim substrate.
type Engine struct {
	db     *sql.DB
	ownsDB bool

	*claimkit.Claims

	*claimkit.Dedup
}

// NewEngine connects to the queue database at dsn and attaches the
// claim/dedup capabilities; the queue task schema is applied idempotently so
// a fresh database is ready for both surfaces.
func NewEngine(ctx context.Context, dsn string) (*Engine, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("queue/mysql engine: open: %w", err)
	}

	eng, err := newEngine(ctx, db, true)
	if err != nil {
		_ = db.Close()

		return nil, err
	}

	return eng, nil
}

// NewEngineFromDB attaches the claim/dedup capabilities to an existing
// database/sql handle. The caller retains ownership of db.
func NewEngineFromDB(ctx context.Context, db *sql.DB) (*Engine, error) {
	return newEngine(ctx, db, false)
}

func newEngine(ctx context.Context, db *sql.DB, ownsDB bool) (*Engine, error) {
	for _, stmt := range schemaStmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return nil, fmt.Errorf("queue/mysql engine: migrate tasks: %w", err)
		}
	}

	claims, err := claimkit.New(ctx, db, claiming.DialectMySQL)
	if err != nil {
		return nil, fmt.Errorf("queue/mysql engine: claimkit: %w", err)
	}

	dedup, err := claimkit.NewDedup(ctx, db, claiming.DialectMySQL)
	if err != nil {
		return nil, fmt.Errorf("queue/mysql engine: claimkit dedup: %w", err)
	}

	return &Engine{db: db, ownsDB: ownsDB, Claims: claims, Dedup: dedup}, nil
}

// mysqlNsPerOp / mysqlNetworkRTT mirror the metaengine/mysqlengine priors
// (same-datacenter round trip, calibratable by the live probe) without a
// production dep on that module.
const (
	mysqlNsPerOp    = 12000.0
	mysqlNetworkRTT = 1 * 1000 * 1000 // 1ms
)

// Profile declares the engine's capabilities: native indexed claims and
// dedup over SQL, and NOTHING else — task storage is not a projection ADT;
// the planner must not route fold queries here.
func (e *Engine) Profile() metaengine.EngineProfile {
	return metaengine.EngineProfile{
		Name:        "queue-mysql",
		NsPerOp:     mysqlNsPerOp,
		Persistence: metaengine.PersistencePersistent,
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTDueClaim: metaengine.ComplexityOLogN,
			metaengine.ADTDedup:    metaengine.ComplexityOLogN,
		},
		NetworkRTT: mysqlNetworkRTT,
	}
}

// PingContext verifies connectivity (driver-factory health probe).
func (e *Engine) PingContext(ctx context.Context) error {
	return e.db.PingContext(ctx) //nolint:wrapcheck // driver-provided ping error
}

// Close releases the database when the engine owns it.
func (e *Engine) Close() error {
	if !e.ownsDB {
		return nil
	}

	return e.db.Close() //nolint:wrapcheck // driver-provided close error
}
