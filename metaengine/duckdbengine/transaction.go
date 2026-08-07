package duckdbengine

import (
	"context"
	"database/sql"
	"fmt"
)

// dbExec is the common interface between *sql.DB and *sql.Tx. Every engine
// operation routes through conn() so that writes and reads participate in the
// active transaction when one exists.
type dbExec interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// conn returns the active transaction if RunInTx is in progress, otherwise
// the engine's *sql.DB.
func (e *duckdbEngine) conn() dbExec {
	if tx := e.activeTx.Load(); tx != nil {
		return tx
	}

	return e.db
}

// RunInTx executes fn within a database transaction. If fn returns nil the
// transaction is committed; otherwise it is rolled back. Concurrent RunInTx
// calls (and all stream-append operations) are serialized via e.mu — only one
// transaction is active at a time.
//
// Operations inside fn automatically route through the active transaction via
// conn(), so MapSet, CounterIncrement, StreamAppend, etc. all participate
// atomically. If fn calls a method that also locks e.mu (e.g.
// StreamAppendExpected), the lock is skipped because RunInTx already holds it.
func (e *duckdbEngine) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("duckdbengine: begin tx: %w", err)
	}

	e.activeTx.Store(tx)

	fnErr := fn(ctx)

	e.activeTx.Store(nil)

	if fnErr != nil {
		_ = tx.Rollback()

		return fnErr
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("duckdbengine: commit tx: %w", err)
	}

	return nil
}
