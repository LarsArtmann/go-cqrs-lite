package metaengine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// dbExec is the common interface between *sql.DB and *sql.Tx.
type dbExec interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// dbExecer is the common interface between stmtCache and txStmtCache.
// Both provide exec/queryRow/query with identical signatures.
type dbExecer interface {
	exec(ctx context.Context, query string, args ...any) (sql.Result, error)
	queryRow(ctx context.Context, query string, args ...any) *sql.Row
	query(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// xc returns the active statement executor for cached operations.
// When inside RunInTx, returns the transaction's txStmtCache; otherwise
// returns the engine's regular stmtCache. Every Map/Set/Counter/etc.
// operation routes through xc() so writes and reads participate in the
// active transaction.
func (e *sqliteEngine) xc() dbExecer {
	if tx := e.activeTx.Load(); tx != nil {
		return tx.cache
	}

	return e.cache
}

// xd returns the active raw DB/Tx for direct SQL operations (dynamic
// queries that cannot use prepared-statement caching, e.g. PushdownMapScan
// with variable WHERE clauses). Both *sql.DB and *sql.Tx satisfy dbExec.
func (e *sqliteEngine) xd() dbExec {
	if tx := e.activeTx.Load(); tx != nil {
		return tx.tx
	}

	return e.db
}

// txStmtCache is a non-caching statement wrapper for *sql.Tx.
// Prepared statements are transaction-scoped, so no caching during tx.
type txStmtCache struct {
	tx dbExec
}

func (c *txStmtCache) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return c.tx.ExecContext(ctx, query, args...) //nolint:wrapcheck
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

// readModifyWriteCached performs a read-modify-write cycle using the cached
// statement executor (xc). Used when an outer transaction is already active —
// SQLite does not support nested BEGIN, so we reuse the outer tx's executor
// instead of calling runTxReadModifyWrite (which starts its own BeginTx).
func readModifyWriteCached(
	ctx context.Context,
	xc dbExecer, //nolint:varnamelen
	getQuery, setQuery, col string,
	key any,
	update func(prev any) any,
) error {
	var valStr string

	err := xc.queryRow(ctx, getQuery, col, encodeKey(key)).Scan(&valStr)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err //nolint:wrapcheck // passthrough
	}

	var prev any

	if err == nil {
		prev = decodeJSONValue(valStr)
	}

	newVal := update(prev)

	_, err = xc.exec(ctx, setQuery, col, encodeKey(key), encodeValue(newVal))

	return err
}

// Compile-time interface assertion.
var _ Transactional = (*sqliteEngine)(nil)
