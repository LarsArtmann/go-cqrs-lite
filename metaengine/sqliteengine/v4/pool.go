package sqliteengine

import (
	"database/sql"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Connection-pool option for the DSN constructors.
//
// The constructors default to MaxOpenConns(1): it serializes writes into the
// WAL ordering the engine's conflict contract assumes. Read-heavy deployments
// (the Kith read-storm class, issue #112) pay for that serialization on every
// query behind one long scan. WAL supports concurrent readers, so raising the
// ceiling is safe for reads; writes may then queue behind busy_timeout
// retries instead of the pool. Raise it ONLY when the deployment is
// read-dominated.
//
// The production pragmas (busy_timeout, cache_size) hold on EVERY pooled
// connection: the DSN constructors apply them per connection at Connect
// (issue #128) — a raised ceiling does not reintroduce busy_timeout=0
// connections. A caller-supplied pool ([NewSQLiteEngine]) applies its own
// pragmas; the engine cannot retrofit them there.

// WithMaxOpenConns raises the database handle's connection-pool ceiling above
// the constructors' serialized default of 1. Usable with
// [NewSQLiteEngineFromDSNWith], [NewSQLiteEngine], and [NewFromDSN]:
//
//	eng, err := sqliteengine.NewSQLiteEngineFromDSNWith("app.db", nil,
//		sqliteengine.WithMaxOpenConns(8))
//
// n <= 0 keeps the default (the database/sql meaning of 0 — unlimited — is a
// footgun behind an option named "max", so it is deliberately not honored).
// The engine applies the ceiling at construction, before any query runs.
//
// Do NOT raise the ceiling on a ":memory:" DSN: every pooled connection to
// :memory: opens its OWN separate in-memory database (no shared cache), so a
// second connection sees empty tables — data appears to vanish under
// concurrency. Use a file DSN (e.g. under t.TempDir()) when raising it.
func WithMaxOpenConns(n int) EngineOption {
	return func(e *sqliteEngine) {
		if n > 0 {
			e.db.SetMaxOpenConns(n)
		}
	}
}

// DBStats exposes the pool statistics of the engine's database handle. The
// owning DSN constructors ([NewSQLiteEngineFromDSN]) never hand the *sql.DB
// to the caller, so [WithMaxOpenConns] consumers have no other way to verify
// the ceiling took effect. Returns zero stats for non-SQLite engines.
func DBStats(eng metaengine.Engine) sql.DBStats {
	if se, ok := eng.(*sqliteEngine); ok {
		return se.db.Stats()
	}

	return sql.DBStats{}
}
