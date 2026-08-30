package sqliteengine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

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
// with variable WHERE clauses). Both *sql.DB and *sql.Tx satisfy SQLExec.
func (e *sqliteEngine) xd() metaengine.SQLExec {
	if tx := e.activeTx.Load(); tx != nil {
		return tx.tx
	}

	return e.db
}

// txStmtCache is a non-caching statement wrapper for *sql.Tx.
// Prepared statements are transaction-scoped, so no caching during tx.
type txStmtCache struct {
	tx metaengine.SQLExec
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

// txExecutor wraps a *sql.Tx and its txStmtCache.
type txExecutor struct {
	tx    *sql.Tx
	cache *txStmtCache
}

// RunInTx executes fn within a database transaction. If fn returns nil, the
// transaction is committed; otherwise rolled back. Concurrent RunInTx calls
// are serialized via txMu — only one transaction active at a time. Nested
// RunInTx is rejected via a marker in the context passed to fn (propagate
// fn's ctx into nested calls); a nested call that breaks ctx propagation
// deadlocks on the serialization mutex instead — don't do that.
func (e *sqliteEngine) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	//art-dupl:accept same ctx-marker nested-tx rejection as dgraphengine — separate go.mod
	if ctx.Value(txMarker{}) != nil {
		return errors.New("sqliteengine.RunInTx: nested transactions are not supported")
	}

	e.txMu.Lock()
	defer e.txMu.Unlock()

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqliteengine: begin tx: %w", err)
	}

	txC := &txExecutor{
		tx:    tx,
		cache: &txStmtCache{tx: tx},
	}

	e.activeTx.Store(txC)

	fnErr := fn(context.WithValue(ctx, txMarker{}, txActive{}))

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

// txMarker keys the context value that marks a RunInTx-managed context.
type txMarker struct{}

// txActive is the marker value stored under txMarker.
type txActive struct{}

// readModifyWriteCached performs a read-modify-write cycle using the cached
// statement executor (xc). Used when an outer transaction is already active —
// SQLite does not support nested BEGIN, so we reuse the outer tx's executor
// instead of calling runTxReadModifyWrite (which starts its own BeginTx).
func readModifyWriteCached(
	ctx context.Context,
	xc dbExecer,
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
		prev = metaengine.DecodeStreamValue(valStr)
	}

	newVal := update(prev)

	_, err = xc.exec(ctx, setQuery, col, encodeKey(key), encodeValue(newVal))

	return err
}
