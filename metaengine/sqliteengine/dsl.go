package sqliteengine

import (
	"context"
	"database/sql"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// NewFromDSN creates a SQLite-backed engine from a DSN string, applying
// production PRAGMAs (WAL mode + busy_timeout), setting MaxOpenConns(1) for
// SQLite safety, and creating the engine tables — all in one call.
//
//	eng, db, err := sqliteengine.NewFromDSN("app.db")
//	defer db.Close()
//	defer eng.Close()
//
// For in-memory databases, pass ":memory:". The caller owns the *sql.DB;
// closing the engine does NOT close the database.
func NewFromDSN(dsn string) (metaengine.Engine, *sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("sqliteengine: open sqlite %q: %w", dsn, err)
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

// PlanFromDSN is the one-shot convenience for the most common setup: create a
// Memory engine + a SQLite engine, plan queries against both, and return the
// store + database handle.
//
//	store, db, err := sqliteengine.PlanFromDSN("app.db", statsQuery, historyQuery)
//	defer store.Close()
//	defer db.Close()
func PlanFromDSN(dsn string, args ...any) (*metaengine.Store, *sql.DB, error) {
	sqliteEng, db, err := NewFromDSN(dsn)
	if err != nil {
		return nil, nil, err
	}

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine(), sqliteEng},
		args...)
	if err != nil {
		_ = db.Close()

		return nil, nil, err
	}

	return store, db, nil
}
