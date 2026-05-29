package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
)

// SQLTransactionalStore wraps SQLEventStore and SQLOutbox to provide atomic
// save+outbox-append in a single database transaction.
//
// Both the event store and outbox must share the same *sql.DB instance.
type SQLTransactionalStore struct {
	*SQLEventStore

	outbox *SQLOutbox
}

// NewSQLTransactionalStore creates a store that atomically saves events and
// appends them to the outbox within a single transaction.
// Returns an error if any parameter is nil.
func NewSQLTransactionalStore(
	store *SQLEventStore,
	outbox *SQLOutbox,
) (*SQLTransactionalStore, error) {
	if store == nil {
		return nil, event.WrapInfrastructure(ErrNilDB, "storage.nil_event_store",
			"event store is required")
	}

	if outbox == nil {
		return nil, event.WrapInfrastructure(ErrNilDB, "storage.nil_outbox",
			"outbox is required")
	}

	return &SQLTransactionalStore{
		SQLEventStore: store,
		outbox:        outbox,
	}, nil
}

// SaveWithOutbox atomically persists events and appends them to the outbox
// within a single database transaction. If either operation fails, the entire
// transaction rolls back.
func (s *SQLTransactionalStore) SaveWithOutbox(
	ctx context.Context,
	ref event.AggregateRef,
	events []event.Event,
	expectedVersion event.Version,
) error {
	ctx, span := startSaveSpan(
		ctx,
		"event.store.save_with_outbox",
		ref,
		expectedVersion,
		len(events),
	)
	defer span.End()

	err := saveWithOutboxTx(
		ctx,
		s.db,
		ref,
		events,
		expectedVersion,
		s.checkVersion,
		s.insertEvents,
		s.appendOutboxTx,
	)
	if err != nil {
		cqrsotel.RecordError(span, err)
	}

	return err
}

func (s *SQLTransactionalStore) appendOutboxTx(
	ctx context.Context,
	tx *sql.Tx,
	events []event.Event,
) error {
	serialized, err := marshalOutboxEvents(events)
	if err != nil {
		return event.WrapCorruption(err, "storage.serialize_outbox",
			"serialize outbox events")
	}

	outboxID := events[0].ID()

	insertSQL := outboxInsertSQL(s.dialect)

	_, err = tx.ExecContext(
		ctx,
		insertSQL,
		outboxID,
		string(OutboxStatusPending),
		serialized,
		s.dialect.FormatTime(time.Now()),
	)
	if err != nil {
		return event.WrapInfrastructure(err, "storage.insert_outbox",
			"insert outbox entry "+outboxID.String())
	}

	return nil
}

var (
	_ event.TransactionalSink  = (*SQLTransactionalStore)(nil)
	_ event.TransactionalStore = (*SQLTransactionalStore)(
		nil,
	)
)
