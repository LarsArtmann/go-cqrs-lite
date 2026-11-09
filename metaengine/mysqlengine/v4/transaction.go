package mysqlengine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// txMarker keys the context value carrying the active transaction.
type txMarker struct{}

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
func (e *mysqlEngine) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	if ctx.Value(txMarker{}) != nil {
		return errors.New("mysqlengine.RunInTx: nested transactions are not supported")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("mysqlengine: begin tx: %w", err)
	}

	fnErr := fn(context.WithValue(ctx, txMarker{}, tx))

	if fnErr != nil {
		_ = tx.Rollback()

		return fnErr
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("mysqlengine: commit tx: %w", err)
	}

	return nil
}
