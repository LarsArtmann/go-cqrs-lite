package sqliteengine

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// NewFromDSN creates a SQLite-backed engine from a DSN string, applying
// production PRAGMAs (WAL mode, busy_timeout, and a 64 MiB page cache),
// setting MaxOpenConns(1) for SQLite safety, and creating the engine tables
// — all in one call.
//
//	eng, db, err := sqliteengine.NewFromDSN("app.db")
//	defer db.Close()
//	defer eng.Close()
//
// For in-memory databases, pass ":memory:". The caller owns the *sql.DB;
// closing the engine does NOT close the database.
//
// MaxOpenConns(1) is deliberate: it serializes writes into the WAL ordering
// the engine's conflict contract assumes — but it serializes READS too, so a
// long scan blocks every other query behind it. WAL supports concurrent
// readers with modernc sqlite; raise the ceiling on the returned handle
// (db.SetMaxOpenConns(n)) for read-heavy deployments, accepting that writes
// may then queue behind busy_timeout retries instead of the pool.
//
// The cache_size pragma (negative KiB form) keeps SQLite's default 2 MB page
// cache from thrashing scan-heavy databases — a measured 536 MB journal
// re-read 8.16 GB from disk in 43 minutes on the 2 MB default. A later
// `PRAGMA cache_size=...` on the returned handle overrides it (SQLite
// last-write-wins).
func NewFromDSN(dsn string) (metaengine.Engine, *sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("sqliteengine: open sqlite %q: %w", dsn, err)
	}

	db.SetMaxOpenConns(1)

	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA cache_size=-65536",
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
	return NewSQLiteEngineFromDSNWith(dsn, pragmas)
}

// NewSQLiteEngineFromDSNWith is the options-aware variant of
// [NewSQLiteEngineFromDSN]: same pragma handling, plus EngineOption
// capabilities (e.g. WithMaterializedViews).
func NewSQLiteEngineFromDSNWith(
	dsn string,
	pragmas []string,
	opts ...EngineOption,
) (metaengine.Engine, error) {
	if dsn == "" {
		dsn = ":memory:"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqliteengine: open sqlite %q: %w", dsn, err)
	}

	db.SetMaxOpenConns(1)

	var synchronousValue string

	// cache_size defaults to 64 MiB (negative-KiB form) BEFORE caller pragmas
	// so a caller-supplied cache_size overrides it (last-write-wins).
	applied := append([]string{
		"journal_mode=WAL",
		"busy_timeout=5000",
		"cache_size=-65536",
	}, pragmas...)

	for _, pragma := range applied {
		if name, value, isSync := strings.Cut(pragma, "="); isSync &&
			strings.EqualFold(strings.TrimSpace(name), "synchronous") {
			synchronousValue = strings.TrimSpace(value)
		}
	}

	for _, pragma := range applied {
		if _, err := db.ExecContext(context.Background(), "PRAGMA "+pragma); err != nil {
			_ = db.Close()

			return nil, fmt.Errorf("sqliteengine: pragma %q: %w", pragma, err)
		}
	}

	eng, err := NewSQLiteEngine(db, opts...)
	if err != nil {
		_ = db.Close()

		return nil, err
	}

	if se, ok := eng.(*sqliteEngine); ok {
		se.synchronousPragma = synchronousValue
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
