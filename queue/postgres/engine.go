package postgres

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver for the claim substrate

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
// (queue.Store[T] over pgx); this surface is the generic claim substrate.
//
// The engine keeps its own database/sql pool (pgx stdlib) — the queue's
// pgxpool and this pool are two handles to one database.
// art-dupl:accept engine scaffolding twin of queue/mysql; dep-isolated modules, claimkit carries the semantics
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
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("queue/postgres engine: open: %w", err)
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
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return nil, fmt.Errorf("queue/postgres engine: migrate tasks: %w", err)
	}

	claims, err := claimkit.New(ctx, db, claiming.DialectPostgres)
	if err != nil {
		return nil, fmt.Errorf("queue/postgres engine: claimkit: %w", err)
	}

	dedup, err := claimkit.NewDedup(ctx, db, claiming.DialectPostgres)
	if err != nil {
		return nil, fmt.Errorf("queue/postgres engine: claimkit dedup: %w", err)
	}

	return &Engine{db: db, ownsDB: ownsDB, Claims: claims, Dedup: dedup}, nil
}

// Profile declares the engine's capabilities: native indexed claims and
// dedup over SQL, and NOTHING else — task storage is not a projection ADT;
// the planner must not route fold queries here.
func (e *Engine) Profile() metaengine.EngineProfile {
	return metaengine.EngineProfile{
		Name:        "queue-postgres",
		NsPerOp:     5000, // calibrated prior: server round trip per op
		Persistence: metaengine.PersistencePersistent,
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTDueClaim: metaengine.ComplexityOLogN,
			metaengine.ADTDedup:    metaengine.ComplexityOLogN,
		},
		NetworkRTT: 500 * 1000, // 500µs server RTT prior (calibratable)
	}
}

// PingContext verifies connectivity (driver-factory health probe).
func (e *Engine) PingContext(ctx context.Context) error {
	//art-dupl:accept engine scaffolding twin; one-line driver ping, only the pool handle differs
	return e.db.PingContext(ctx) //nolint:wrapcheck // driver-provided ping error
}

// Close releases the database when the engine owns it.
func (e *Engine) Close() error {
	if !e.ownsDB {
		return nil
	}

	return e.db.Close() //nolint:wrapcheck // driver-provided close error
}
