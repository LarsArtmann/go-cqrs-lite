package metaengine

import (
	"context"
	"database/sql"
	"fmt"
)

// dbExec is the common interface between *sql.DB and *sql.Tx.
type dbExec interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// txStmtCache is a non-caching statement wrapper for *sql.Tx.
// Prepared statements are transaction-scoped, so no caching during tx.
type txStmtCache struct {
	tx dbExec
}

func (c *txStmtCache) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return c.tx.ExecContext(ctx, query, args...)
}

func (c *txStmtCache) queryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return c.tx.QueryRowContext(ctx, query, args...)
}

func (c *txStmtCache) query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return c.tx.QueryContext(ctx, query, args...) //nolint:wrapcheck
}

func (c *txStmtCache) close() {}

// txExecutor wraps a *sql.Tx and its txStmtCache.
type txExecutor struct {
	tx    *sql.Tx
	cache *txStmtCache
}

// Transactional is an optional capability for engines that support explicit
// transactions.
type Transactional interface {
	RunInTx(ctx context.Context, fn func(context.Context) error) error
}

// RunInTx executes fn within a database transaction. If fn returns nil, the
// transaction is committed; otherwise rolled back. Concurrent RunInTx calls
// are serialized via txMu — only one transaction active at a time.
func (e *sqliteEngine) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	e.txMu.Lock()
	defer e.txMu.Unlock()

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("metaengine: begin tx: %w", err)
	}

	txC := &txExecutor{
		tx:    tx,
		cache: &txStmtCache{tx: tx},
	}

	e.activeTx.Store(txC)

	fnErr := fn(ctx)

	e.activeTx.Store(nil)

	if fnErr != nil {
		_ = tx.Rollback()

		return fnErr
	}

	return tx.Commit() //nolint:wrapcheck
}

// txExec returns the active transaction's executor, or nil if no transaction
// is active.
func (e *sqliteEngine) txExec() *txExecutor {
	return e.activeTx.Load()
}

// Compile-time interface assertion.
var _ Transactional = (*sqliteEngine)(nil)
