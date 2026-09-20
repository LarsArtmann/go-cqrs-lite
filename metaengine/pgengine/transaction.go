package pgengine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// txMarker keys the context value carrying the active transaction.
type txMarker struct{}

// conn returns the ambient transaction when ctx descends from a RunInTx
// callback, otherwise the engine's *sql.DB. Transaction affinity flows
// through the context — never engine-global state.
func (e *pgEngine) conn(ctx context.Context) metaengine.SQLExec {
	if tx := txFromCtx(ctx); tx != nil {
		return tx
	}

	return e.db
}

// inTx runs fn in a transaction. If ctx descends from a RunInTx callback,
// fn participates in that transaction. Otherwise a new transaction is
// started and committed (or rolled back on error). Used by StreamAppend and
// StreamAppendExpected which need per-call atomicity when called standalone.
func (e *pgEngine) inTx(ctx context.Context, fn func(metaengine.SQLExec) error) error {
	if tx := txFromCtx(ctx); tx != nil {
		return fn(tx)
	}

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("pgengine: begin tx: %w", err)
	}

	fnErr := fn(tx)

	if fnErr != nil {
		_ = tx.Rollback()

		return fnErr
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("pgengine: commit tx: %w", err)
	}

	return nil
}

// txFromCtx returns the transaction carried by ctx, or nil when ctx does
// not descend from a RunInTx callback.
func txFromCtx(ctx context.Context) *sql.Tx {
	tx, _ := ctx.Value(txMarker{}).(*sql.Tx)

	return tx
}

// RunInTx executes fn within a database transaction. If fn returns nil the
// transaction is committed; otherwise it is rolled back. Concurrent RunInTx
// calls are serialized via e.mu — only one transaction is active at a time.
//
// The transaction is carried in fn's context: every engine operation that
// receives a ctx descended from fn joins the transaction, everything else
// talks to the base pool — transaction affinity flows through the context,
// NEVER engine-global state (the engine-global activeTx leaked transactions
// to unrelated goroutines: dirty reads and "sql: Rows are closed"; ported
// from sqliteengine 22ab7b218, 2026-09-20). A nested RunInTx is rejected.
func (e *pgEngine) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	if ctx.Value(txMarker{}) != nil {
		return errors.New("pgengine.RunInTx: nested transactions are not supported")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("pgengine: begin tx: %w", err)
	}

	fnErr := fn(context.WithValue(ctx, txMarker{}, tx))

	if fnErr != nil {
		_ = tx.Rollback()

		return fnErr
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("pgengine: commit tx: %w", err)
	}

	return nil
}
