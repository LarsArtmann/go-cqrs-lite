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

// NewSQLiteEngineFromDSN creates a SQLite-backed engine that OWNS its database
// connection: Close also closes the *sql.DB. This is the variant for
// driver-factory paths (and any caller that cannot keep the handle) — it plugs
// the leak where a self-opened database is never closed. Extra pragmas are
// applied after the WAL/busy_timeout production defaults. Callers that want to
// keep the handle (or pass their own pool) should use [NewFromDSN] or
// [NewSQLiteEngine], where the caller owns the database.
func NewSQLiteEngineFromDSN(dsn string, pragmas ...string) (metaengine.Engine, error) {
	if dsn == "" {
		dsn = ":memory:"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqliteengine: open sqlite %q: %w", dsn, err)
	}

	db.SetMaxOpenConns(1)

	applied := append([]string{
		"journal_mode=WAL",
		"busy_timeout=5000",
	}, pragmas...)

	for _, pragma := range applied {
		if _, err := db.ExecContext(context.Background(), "PRAGMA "+pragma); err != nil {
			_ = db.Close()

			return nil, fmt.Errorf("sqliteengine: pragma %q: %w", pragma, err)
		}
	}

	eng, err := NewSQLiteEngine(db)
	if err != nil {
		_ = db.Close()

		return nil, err
	}

	OwnDB(eng)

	return eng, nil
}

// OwnDB marks an engine created via [NewSQLiteEngine] as the owner of its
// *sql.DB: Close will then also close the database. For driver-factory and
// adapter paths (e.g. the Turso engine, which opens a libSQL connection with
// its own driver) where no caller exists to own the handle. Engines wrapping
// a caller-supplied pool must NOT be marked.
func OwnDB(eng metaengine.Engine) {
	if se, ok := eng.(*sqliteEngine); ok {
		se.ownsDB = true
	}
}
