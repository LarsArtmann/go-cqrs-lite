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

// xc returns the statement executor for cached operations: the ambient
// transaction's txStmtCache when ctx descends from RunInTx, otherwise the
// engine's regular stmtCache. Transaction affinity flows through the
// context — NEVER engine-global state — so a transaction is visible only
// to the call tree that opened it. A concurrent caller with its own ctx
// always talks to the base pool and can neither observe uncommitted
// writes nor have its rows closed by a foreign tx commit.
func (e *sqliteEngine) xc(ctx context.Context) dbExecer {
	if tx := txFromCtx(ctx); tx != nil {
		return tx.cache
	}

	return e.cache
}

// xd returns the raw DB or ambient transaction for direct SQL operations
// (dynamic queries that cannot use prepared-statement caching, e.g.
// PushdownMapScan with variable WHERE clauses). Like xc, the transaction
// is resolved from ctx; both *sql.DB and *sql.Tx satisfy SQLExec.
func (e *sqliteEngine) xd(ctx context.Context) metaengine.SQLExec {
	if tx := txFromCtx(ctx); tx != nil {
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
// are serialized via txMu — only one transaction active at a time.
//
// The transaction is carried in fn's context: every engine operation that
// receives a ctx descended from fn joins the transaction, everything else
// talks to the base pool. Propagate fn's ctx into nested calls; a nested
// RunInTx is rejected via the same context marker.
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

	fnErr := fn(context.WithValue(ctx, txMarker{}, txC))

	if fnErr != nil {
		_ = tx.Rollback()

		return fnErr
	}

	return tx.Commit() //nolint:wrapcheck
}

// txFromCtx returns the transaction executor carried by ctx, or nil when
// ctx does not descend from a RunInTx callback.
func txFromCtx(ctx context.Context) *txExecutor {
	tx, _ := ctx.Value(txMarker{}).(*txExecutor)

	return tx
}

// txExec returns the transaction executor carried by ctx, or nil when no
// transaction is ambient. It answers "am I inside a RunInTx callback?" for
// call sites that need the executor itself, not just routing.
func (e *sqliteEngine) txExec(ctx context.Context) *txExecutor {
	return txFromCtx(ctx)
}

// txMarker keys the context value carrying the active transaction executor.
type txMarker struct{}

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
