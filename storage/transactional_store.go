package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// SQLTransactionalStore wraps SQLEventStore and SQLOutbox to provide atomic
// save+outbox-append in a single database transaction.
//
// Both the event store and outbox must share the same *sql.DB instance.
type SQLTransactionalStore struct {
	*SQLEventStore

	outbox  *SQLOutbox
	db      *sql.DB
	dialect Dialect
}

// NewSQLTransactionalStore creates a store that atomically saves events and
// appends them to the outbox within a single transaction.
// Returns an error if any parameter is nil.
func NewSQLTransactionalStore(
	store *SQLEventStore,
	outbox *SQLOutbox,
) (*SQLTransactionalStore, error) {
	if store == nil {
		return nil, fmt.Errorf("%w: event store is required", ErrNilDB)
	}

	if outbox == nil {
		return nil, fmt.Errorf("%w: outbox is required", ErrNilDB)
	}

	return &SQLTransactionalStore{
		SQLEventStore: store,
		outbox:        outbox,
		db:            store.db,
		dialect:       store.dialect,
	}, nil
}

// NewSQLiteTransactionalStore creates a SQLite store that atomically saves events and
// appends them to the outbox within a single transaction.
// Returns an error if any parameter is nil.
func NewSQLiteTransactionalStore(
	store *SQLEventStore,
	outbox *SQLOutbox,
) (*SQLTransactionalStore, error) {
	return NewSQLTransactionalStore(store, outbox)
}

// SaveWithOutbox atomically persists events and appends them to the outbox
// within a single database transaction. If either operation fails, the entire
// transaction rolls back.
func (s *SQLTransactionalStore) SaveWithOutbox(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	events []event.Event,
	expectedVersion event.Version,
) error {
	return saveWithOutboxTx(
		ctx,
		s.db,
		aggregateType,
		aggregateID,
		events,
		expectedVersion,
		s.SQLEventStore.checkVersion,
		s.SQLEventStore.insertEvents,
		s.appendOutboxTx,
	)
}

func (s *SQLTransactionalStore) appendOutboxTx(
	ctx context.Context,
	tx *sql.Tx,
	events []event.Event,
) error {
	serialized, err := marshalOutboxEvents(events)
	if err != nil {
		return fmt.Errorf("serialize outbox events: %w", err)
	}

	outboxID := events[0].ID()

	p1, p2, p3, p4 := s.dialect.Placeholder(
		1,
	), s.dialect.Placeholder(
		2,
	), s.dialect.Placeholder(
		3,
	), s.dialect.Placeholder(
		4,
	)

	insertSQL := fmt.Sprintf(
		`INSERT INTO outbox (id, status, events, created_at) VALUES (%s, %s, %s, %s)`,
		p1, p2, p3, p4,
	)

	_, err = tx.ExecContext(
		ctx,
		insertSQL,
		outboxID,
		string(OutboxStatusPending),
		serialized,
		s.dialect.FormatTime(time.Now()),
	)
	if err != nil {
		return fmt.Errorf("insert outbox entry %s: %w", outboxID, err)
	}

	return nil
}

var _ event.TransactionalStore = (*SQLTransactionalStore)(nil)
