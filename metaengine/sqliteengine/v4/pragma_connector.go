package sqliteengine

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"

	"modernc.org/sqlite"
)

// pragmaConnector wraps the modernc.org/sqlite connector so every physical
// connection the pool opens runs the pragmas — not just the first one.
//
// busy_timeout and cache_size are PER-CONNECTION settings in SQLite, but the
// historical constructors applied them once via db.ExecContext after
// sql.Open, which reaches exactly one pooled connection. Under
// [WithMaxOpenConns] every additional connection opened with busy_timeout=0,
// so a write contending with a concurrent reader failed with SQLITE_BUSY
// immediately — well under the 5s timeout the configuration claims (issue
// #128's "the BUSY surfaced in well under busy_timeout=5000"). Wrapping
// Connect applies the pragmas where they belong: on each new connection,
// before any caller statement runs on it.
type pragmaConnector struct {
	driver.Connector

	pragmas []string
}

// Connect opens a base connection and applies the configured pragmas on it.
// A pragma failure fails the connect (and closes the connection) so a bad
// pragma surfaces as a pool error, not as a silently-missing setting.
func (c pragmaConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}

	execer, ok := conn.(driver.ExecerContext)
	if !ok {
		_ = conn.Close()

		return nil, fmt.Errorf("sqliteengine: connection lacks ExecerContext; cannot apply pragmas")
	}

	for _, pragma := range c.pragmas {
		if _, err := execer.ExecContext(ctx, "PRAGMA "+pragma, nil); err != nil {
			_ = conn.Close()

			return nil, fmt.Errorf("sqliteengine: pragma %q: %w", pragma, err)
		}
	}

	return conn, nil
}

// openPragmaPool opens a SQLite pool whose every connection applies pragmas
// (see [pragmaConnector]) and validates them eagerly: the ping forces the
// first connection open, so an invalid pragma fails construction instead of
// the first query. MaxOpenConns(1) is the serialized-writes default;
// [WithMaxOpenConns] raises it.
func openPragmaPool(dsn string, pragmas []string) (*sql.DB, error) {
	base, err := sqlite.NewConnector(dsn)
	if err != nil {
		return nil, fmt.Errorf("sqliteengine: open sqlite %q: %w", dsn, err)
	}

	db := sql.OpenDB(pragmaConnector{Connector: base, pragmas: pragmas})
	db.SetMaxOpenConns(1)

	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("sqliteengine: connect sqlite %q: %w", dsn, err)
	}

	return db, nil
}
