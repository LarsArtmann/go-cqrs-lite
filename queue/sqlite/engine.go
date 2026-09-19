package sqlite

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
// the ONE shared claimkit runtime, over the SAME *sql.DB the tasks live in —
// so timers, dedup keys, and task claims share one database and one
// transaction domain. The queue's own task tables and their richer
// claim semantics (priority+aging, dependency gating) remain engine-owned
// (queue.Store[T]); this surface is the generic claim substrate, not a
// replacement for the task contract.
type Engine struct {
	db     *sql.DB
	ownsDB bool

	*claimkit.Claims

	*claimkit.Dedup
}

// NewEngine opens (creating if needed) a queue database at path and attaches
// the claim/dedup capabilities. The queue's task schema and the claimkit
// tables coexist; both are created idempotently.
func NewEngine(path string) (*Engine, error) {
	dsn := fmt.Sprintf(
		"file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)",
		path,
	)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("queue/sqlite engine: open: %w", err)
	}

	db.SetMaxOpenConns(1)

	eng, err := newEngine(context.Background(), db, true)
	if err != nil {
		_ = db.Close()

		return nil, err
	}

	return eng, nil
}

// NewEngineFromDB attaches the claim/dedup capabilities to an existing queue
// database. The caller retains ownership of db.
func NewEngineFromDB(db *sql.DB) (*Engine, error) {
	return newEngine(context.Background(), db, false)
}

func newEngine(ctx context.Context, db *sql.DB, ownsDB bool) (*Engine, error) {
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return nil, fmt.Errorf("queue/sqlite engine: migrate tasks: %w", err)
	}

	claims, err := claimkit.New(ctx, db, claiming.DialectSQLite)
	if err != nil {
		return nil, fmt.Errorf("queue/sqlite engine: claimkit: %w", err)
	}

	dedup, err := claimkit.NewDedup(ctx, db, claiming.DialectSQLite)
	if err != nil {
		return nil, fmt.Errorf("queue/sqlite engine: claimkit dedup: %w", err)
	}

	return &Engine{db: db, ownsDB: ownsDB, Claims: claims, Dedup: dedup}, nil
}

// Profile declares the engine's capabilities: native indexed claims and
// dedup over SQL, and NOTHING else — task storage is not a projection ADT;
// the planner must not route fold queries here.
func (e *Engine) Profile() metaengine.EngineProfile {
	return metaengine.EngineProfile{
		Name:        "queue-sqlite",
		NsPerOp:     metaengine.SQLiteNsPerOp,
		Persistence: metaengine.PersistencePersistent,
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTDueClaim: metaengine.ComplexityOLogN,
			metaengine.ADTDedup:    metaengine.ComplexityOLogN,
		},
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
