package sql

import (
	"context"
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v2"
)

// RunInTx executes fn inside a database transaction.
// If fn returns an error, the transaction is rolled back.
// If fn returns nil, the transaction is committed.
// Errors from BeginTx and CommitTx are recorded on the span and wrapped
// as infrastructure errors.
func RunInTx(
	ctx context.Context,
	db *sql.DB,
	span cqrsotel.Span,
	fn func(*sql.Tx) error,
) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		cqrsotel.RecordError(span, err)
		return event.WrapInfrastructure(err, "storage.begin_tx", "begin transaction")
	}

	defer func() {
		_ = tx.Rollback()
	}()

	if err := fn(tx); err != nil {
		return err
	}

	err = CommitTx(tx)
	if err != nil {
		cqrsotel.RecordError(span, err)
	}

	return err
}
