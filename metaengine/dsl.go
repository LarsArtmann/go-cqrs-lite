package metaengine

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

// NewSQLiteEngineFromDSN creates a SQLite-backed engine from a DSN string,
// applying production PRAGMAs (WAL mode + busy_timeout), setting
// MaxOpenConns(1) for SQLite safety, and creating the engine tables — all
// in one call. This replaces ~25 lines of boilerplate every consumer
// previously wrote:
//
//	eng, db, err := metaengine.NewSQLiteEngineFromDSN("app.db")
//	defer db.Close()
//	defer eng.Close()
//
// For in-memory databases, pass ":memory:". The caller owns the *sql.DB;
// closing the engine does NOT close the database.
func NewSQLiteEngineFromDSN(dsn string) (Engine, *sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("metaengine: open sqlite %q: %w", dsn, err)
	}

	// SQLite: serialize writes and ensure :memory: visibility across goroutines.
	db.SetMaxOpenConns(1)

	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
	} {
		if _, err := db.ExecContext(context.Background(), pragma); err != nil {
			_ = db.Close()

			return nil, nil, fmt.Errorf("metaengine: %s: %w", pragma, err)
		}
	}

	eng, err := NewSQLiteEngine(db)
	if err != nil {
		_ = db.Close()

		return nil, nil, err
	}

	return eng, db, nil
}

// PlanFromSQLite is the one-shot convenience for the most common setup:
// create a Memory engine + a SQLite engine, plan queries against both, and
// return the store + database handle. The planner picks the cheapest engine
// per query (Memory for hot counters, SQLite for persistent reads).
//
//	store, db, err := metaengine.PlanFromSQLite("app.db", statsQuery, historyQuery)
//	defer store.Close()
//	defer db.Close()
//
// For in-memory-only (dev/test), use Plan with just NewMemoryEngine:
//
//	store, err := metaengine.Plan([]metaengine.Engine{metaengine.NewMemoryEngine()}, q)
func PlanFromSQLite(dsn string, args ...any) (*Store, *sql.DB, error) {
	sqliteEng, db, err := NewSQLiteEngineFromDSN(dsn)
	if err != nil {
		return nil, nil, err
	}

	store, err := Plan([]Engine{NewMemoryEngine(), sqliteEng}, args...)
	if err != nil {
		_ = db.Close()

		return nil, nil, err
	}

	return store, db, nil
}

// LogPlan logs the planner's decisions and diagnostics via slog. Call it
// once after Plan or PlanFromSQLite, at startup, so the optimizer's choices
// are visible in production logs:
//
//	store, _, _ := metaengine.PlanFromSQLite("app.db", stats)
//	store.LogPlan(logger)
//
// Each query logs: name, ADT, assigned engine, complexity, read pattern,
// estimated latency. Diagnostics (WARN/SCREAM) are logged separately.
// If no plan exists (nil store or planning error), this is a no-op.
func (s *Store) LogPlan(logger *slog.Logger) {
	plan := s.Plan()
	if plan == nil {
		return
	}

	for _, q := range plan.Queries {
		logger.Info("metaengine: query planned",
			"query", q.QueryName,
			"adt", string(q.ADT),
			"engine", q.EngineName,
			"complexity", string(q.Complexity),
			"read_pattern", string(q.ReadPattern),
			"estimated_latency_ms", q.Cost.EstimatedLatencyMs,
		)
	}

	for _, d := range plan.Diagnostics {
		logger.Warn("metaengine: diagnostic",
			"query", d.Query,
			"level", d.Level,
			"message", d.Message,
		)
	}
}
