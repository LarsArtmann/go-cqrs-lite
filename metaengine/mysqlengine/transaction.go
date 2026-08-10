package mysqlengine

import (
	"context"
	"fmt"
)

// RunInTx executes fn within a database transaction. If fn returns nil the
// transaction is committed; otherwise it is rolled back. Concurrent RunInTx
// calls are serialized via e.mu — only one transaction is active at a time.
//
// Operations inside fn automatically route through the active transaction via
// conn(), so MapSet, CounterIncrement, StreamAppend, etc. all participate
// atomically.
//art-dupl:accept cross-module SQL engine pattern — separate go.mod
func (e *mysqlEngine) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("mysqlengine: begin tx: %w", err)
	}

	e.activeTx.Store(tx)

	fnErr := fn(ctx)

	e.activeTx.Store(nil)

	if fnErr != nil {
		_ = tx.Rollback()

		return fnErr
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("mysqlengine: commit tx: %w", err)
	}

	return nil
}
