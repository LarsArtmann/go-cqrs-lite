package duckdbengine

import (
	"context"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// inStreamTx runs fn inside one transaction unless ctx already carries one.
// A stream append is a batch: without a transaction of its own each value is
// an autocommitted statement, so an error part-way (a duplicate event id, for
// instance) leaves the earlier values committed while the caller gets an
// error for the whole batch.
func (e *duckdbEngine) inStreamTx(
	ctx context.Context,
	fn func(context.Context) error,
) error { // art-dupl:accept commit-or-rollback tail mirrors RunInTx in transaction.go; unlike it this must not take e.mu
	if txFromCtx(ctx) != nil {
		return fn(ctx)
	}

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("duckdbengine: begin tx: %w", err)
	}

	fnErr := fn(context.WithValue(ctx, txMarker{}, tx))

	if fnErr != nil {
		_ = tx.Rollback()

		return fnErr
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("duckdbengine: commit tx: %w", err)
	}

	return nil
}

var _ metaengine.AtomicAppender = (*duckdbEngine)(nil)
