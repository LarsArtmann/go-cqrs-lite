package sqliteengine

import (
	"context"
	"database/sql"
	"sync"
)

// stmtCache lazily prepares and caches *sql.Stmt by query string. This
// eliminates the SQL parse overhead on every MapSet/MapGet/PushdownMapScan call.
// database/sql does not cache statements from db.Query/db.Exec — each call
// re-prepares internally. By keeping *sql.Stmt alive, we amortize the prepare
// cost across all calls with the same query text.
type stmtCache struct {
	db *sql.DB
	m  sync.Map // query string → *sql.Stmt
}

func newStmtCache(db *sql.DB) *stmtCache {
	return &stmtCache{db: db}
}

// prepare returns a cached *sql.Stmt for the given query, preparing it on first
// use. Concurrent calls for the same query may both prepare (LoadOrStore
// deduplicates), but only the winner stays cached — the loser is closed.
func (c *stmtCache) prepare(ctx context.Context, query string) (*sql.Stmt, error) {
	if v, ok := c.m.Load(query); ok {
		return v.(*sql.Stmt), nil
	}

	stmt, err := c.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	actual, loaded := c.m.LoadOrStore(query, stmt)
	if loaded {
		_ = stmt.Close() //nolint:sqlclosecheck // intentional immediate close — loser of race
	}

	return actual.(*sql.Stmt), nil
}

// exec executes a statement via the cache, falling back to db.ExecContext on
// prepare errors (e.g., transient connection issues).
func (c *stmtCache) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	stmt, err := c.prepare(ctx, query) //nolint:sqlclosecheck // cached stmt
	if err != nil {
		return c.db.ExecContext(ctx, query, args...) //nolint:wrapcheck // passthrough
	}

	return stmt.ExecContext(ctx, args...) //nolint:wrapcheck // passthrough
}

// queryRow executes a one-row query via the cache.
func (c *stmtCache) queryRow(ctx context.Context, query string, args ...any) *sql.Row {
	stmt, err := c.prepare(ctx, query) //nolint:sqlclosecheck // cached stmt
	if err != nil {
		return c.db.QueryRowContext(ctx, query, args...)
	}

	return stmt.QueryRowContext(ctx, args...)
}

// query executes a multi-row query via the cache.
func (c *stmtCache) query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	stmt, err := c.prepare(ctx, query) //nolint:sqlclosecheck // cached stmt
	if err != nil {
		return c.db.QueryContext(ctx, query, args...) //nolint:wrapcheck // passthrough
	}

	return stmt.QueryContext(ctx, args...) //nolint:wrapcheck // passthrough
}

// close closes all cached statements. Called by sqliteEngine.Close.
func (c *stmtCache) close() {
	c.m.Range(func(_, val any) bool {
		_ = val.(*sql.Stmt).Close()

		return true
	})
}
