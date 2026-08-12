package pgengine

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
func (e *pgEngine) conn() dbExec {
	if tx := e.activeTx.Load(); tx != nil {
		return tx
	}

	return e.db
}

// inTx runs fn in a transaction. If RunInTx is in progress, fn participates
// in the outer transaction. Otherwise a new transaction is started and
// committed (or rolled back on error). Used by StreamAppend and
// StreamAppendExpected which need per-call atomicity when called standalone.
func (e *pgEngine) inTx(ctx context.Context, fn func(dbExec) error) error {
	if tx := e.activeTx.Load(); tx != nil {
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

// RunInTx executes fn within a database transaction. If fn returns nil the
// transaction is committed; otherwise it is rolled back. Concurrent RunInTx
// calls are serialized via e.mu — only one transaction is active at a time.
//
// Operations inside fn automatically route through the active transaction via
// conn(), so MapSet, CounterIncrement, StreamAppend, etc. all participate
// atomically. If fn calls a method that starts its own transaction (e.g.
// StreamAppend), it detects the active tx via inTx and reuses it instead of
// starting a nested one.
func (e *pgEngine) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("pgengine: begin tx: %w", err)
	}

	e.activeTx.Store(tx)

	fnErr := fn(ctx)

	e.activeTx.Store(nil)

	if fnErr != nil {
		_ = tx.Rollback()

		return fnErr
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("pgengine: commit tx: %w", err)
	}

	return nil
}
