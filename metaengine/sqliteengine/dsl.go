package sqliteengine

import (
	"context"
	"database/sql"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// NewSQLiteEngineFromDSN is the one-call SQLite setup: opens the database,
// applies PRAGMAs (WAL, busy_timeout), serializes writes (MaxOpenConns(1)),
// and creates the metaengine tables. Returns the engine + the *sql.DB handle
// (caller owns the DB; engine Close is a no-op).
func NewSQLiteEngineFromDSN(dsn string) (metaengine.Engine, *sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("sqliteengine: open %q: %w", dsn, err)
	}

	db.SetMaxOpenConns(1)

	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
	} {
		if _, err := db.ExecContext(context.Background(), pragma); err != nil {
			_ = db.Close()

			return nil, nil, fmt.Errorf("sqliteengine: %s: %w", pragma, err)
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
func PlanFromSQLite(dsn string, args ...any) (*metaengine.Store, *sql.DB, error) {
	sqliteEng, db, err := NewSQLiteEngineFromDSN(dsn)
	if err != nil {
		return nil, nil, err
	}

	store, err := metaengine.Plan([]metaengine.Engine{metaengine.NewMemoryEngine(), sqliteEng}, args...)
	if err != nil {
		_ = db.Close()

		return nil, nil, err
	}

	return store, db, nil
}
