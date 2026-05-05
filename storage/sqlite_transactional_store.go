package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// SQLiteTransactionalStore wraps SQLiteEventStore to provide atomic
// save+outbox-append in a single database transaction.
type SQLiteTransactionalStore struct {
	*SQLiteEventStore
	outbox *SQLiteOutbox
	db     *sql.DB
}

// NewSQLiteTransactionalStore creates a store that atomically saves events and
// appends them to the outbox within a single transaction.
// Returns an error if any parameter is nil.
func NewSQLiteTransactionalStore(
	store *SQLiteEventStore,
	outbox *SQLiteOutbox,
) (*SQLiteTransactionalStore, error) {
	if store == nil {
		return nil, fmt.Errorf("%w: event store is required", ErrNilDB)
	}

	if outbox == nil {
		return nil, fmt.Errorf("%w: outbox is required", ErrNilDB)
	}

	return &SQLiteTransactionalStore{
		SQLiteEventStore: store,
		outbox:           outbox,
		db:               store.db,
	}, nil
}

// SaveWithOutbox atomically persists events and appends them to the outbox
// within a single database transaction.
func (s *SQLiteTransactionalStore) SaveWithOutbox(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	events []event.Event,
	expectedVersion event.Version,
	outbox event.Outbox,
) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	err = sqliteCheckVersion(ctx, tx, aggregateType, aggregateID, expectedVersion)
	if err != nil {
		return err
	}

	err = sqliteInsertEvents(ctx, tx, aggregateType, aggregateID, events)
	if err != nil {
		return err
	}

	err = sqliteAppendOutboxTx(ctx, tx, events)
	if err != nil {
		return err
	}

	return commitTx(tx)
}

func sqliteAppendOutboxTx(
	ctx context.Context,
	tx *sql.Tx,
	events []event.Event,
) error {
	serialized, err := marshalOutboxEvents(events)
	if err != nil {
		return fmt.Errorf("serialize outbox events: %w", err)
	}

	outboxID := events[0].ID()

	_, err = tx.ExecContext(ctx, sqliteOutboxInsertSQL, outboxID, serialized, time.Now())
	if err != nil {
		return fmt.Errorf("insert outbox entry %s: %w", outboxID, err)
	}

	return nil
}

var _ event.TransactionalStore = (*SQLiteTransactionalStore)(nil)
