package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/sql"
)

type SQLTransactionalStore struct {
	*SQLEventStore

	outbox *SQLOutbox
}

func NewSQLTransactionalStore(store *SQLEventStore, outbox *SQLOutbox) (*SQLTransactionalStore, error) {
	if store == nil {
		return nil, event.WrapInfrastructure(sqlpkg.ErrNilDB, "storage.nil_event_store", "event store is required")
	}
	if outbox == nil {
		return nil, event.WrapInfrastructure(sqlpkg.ErrNilDB, "storage.nil_outbox", "outbox is required")
	}
	return &SQLTransactionalStore{SQLEventStore: store, outbox: outbox}, nil
}

func (s *SQLTransactionalStore) SaveWithOutbox(
	ctx context.Context,
	ref event.AggregateRef,
	events []event.Event,
	expectedVersion event.Version,
) error {
	ctx, span := sqlpkg.StartSaveSpan(ctx, "event.store.save_with_outbox", ref, expectedVersion, len(events))
	defer span.End()
	err := sqlpkg.SaveWithOutboxTx(
		ctx,
		s.DB,
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

func (s *SQLTransactionalStore) appendOutboxTx(ctx context.Context, tx *sql.Tx, events []event.Event) error {
	serialized, err := marshalOutboxEvents(events)
	if err != nil {
		return event.WrapCorruption(err, "storage.serialize_outbox", "serialize outbox events")
	}
	outboxID := events[0].ID()
	insertSQL := sqlpkg.OutboxInsertSQL(s.Dialect)
	_, err = tx.ExecContext(
		ctx,
		insertSQL,
		outboxID,
		string(sqlpkg.OutboxStatusPending),
		serialized,
		s.Dialect.FormatTime(time.Now()),
	)
	if err != nil {
		return event.WrapInfrastructure(err, "storage.insert_outbox", "insert outbox entry "+outboxID.String())
	}
	return nil
}

var _ event.TransactionalSink = (*SQLTransactionalStore)(nil)
