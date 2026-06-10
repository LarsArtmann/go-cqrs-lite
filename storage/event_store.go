package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v2"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v2/sql"
)

// SQLEventStore persists events in a SQL database with optimistic concurrency.
type SQLEventStore struct {
	*sqlpkg.ClosableBase
}

// NewSQLEventStore creates a new SQL-backed event store using PostgreSQL dialect.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLEventStore(db *sql.DB) (*SQLEventStore, error) {
	return newSQLEventStoreWithDialect(db, sqlpkg.PostgresDialect{})
}

// NewSQLiteEventStore creates a new SQLite-backed event store.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLiteEventStore(db *sql.DB) (*SQLEventStore, error) {
	return newSQLEventStoreWithDialect(db, sqlpkg.SQLiteDialect{})
}

// NewSQLEventStoreWithDialect creates a new SQL-backed event store with a custom dialect.
// This enables consumers to use any SQL backend (MySQL, CockroachDB, etc.) by implementing the Dialect interface.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLEventStoreWithDialect(db *sql.DB, d sqlpkg.Dialect) (*SQLEventStore, error) {
	return newSQLEventStoreWithDialect(db, d)
}

func newSQLEventStoreWithDialect(db *sql.DB, d sqlpkg.Dialect) (*SQLEventStore, error) {
	base, err := sqlpkg.NewClosableBase(db, d, false)
	if err != nil {
		return nil, err
	}

	return &SQLEventStore{ClosableBase: base}, nil
}

func (s *SQLEventStore) checkClosed() error {
	return s.ClosableBase.CheckClosed(
		event.NewInfrastructure("storage.closed", "store is closed"),
	)
}

// Save persists events with optimistic concurrency check.
func (s *SQLEventStore) Save(
	ctx context.Context,
	ref event.AggregateRef,
	events []event.Event,
	expectedVersion event.Version,
) error {
	if err := s.checkClosed(); err != nil {
		return err
	}

	aggregateType, aggregateID := ref.Type, ref.ID
	if len(events) == 0 {
		return nil
	}

	ctx, span := sqlpkg.StartSaveSpan(
		ctx,
		"event.store.save",
		ref,
		expectedVersion,
		len(events),
	)
	defer span.End()

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return event.WrapInfrastructure(err, "storage.begin_tx",
			"begin transaction")
	}

	defer func() {
		_ = tx.Rollback()
	}()

	err = s.checkVersion(ctx, tx, ref, expectedVersion)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return event.WrapInfrastructure(err, "storage.check_version",
			fmt.Sprintf("check version for %s %s", aggregateType, aggregateID))
	}

	err = s.insertEvents(ctx, tx, ref, events)
	if err != nil {
		return s.wrapInsertEventsErr(span, err, events, ref)
	}

	err = sqlpkg.CommitTx(tx)
	if err != nil {
		cqrsotel.RecordError(span, err)
	}

	return err
}

// AppendBatch appends events without optimistic concurrency checks.
// All events are inserted in a single transaction for atomicity.
func (s *SQLEventStore) AppendBatch(
	ctx context.Context,
	ref event.AggregateRef,
	events []event.Event,
) error {
	if err := s.checkClosed(); err != nil {
		return err
	}

	if len(events) == 0 {
		return nil
	}

	ctx, span := cqrsotel.StartSpan(
		ctx, sqlpkg.Tracer(), "event.store.append_batch",
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(append(
			cqrsotel.AggregateAttrs(ref.Type, ref.ID),
			cqrsotel.AttrInt(cqrsotel.AttrEventCount, len(events)),
		)...),
	)
	defer span.End()

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return event.WrapInfrastructure(err, "storage.begin_tx",
			"begin transaction")
	}

	defer func() {
		_ = tx.Rollback()
	}()

	err = s.insertEvents(ctx, tx, ref, events)
	if err != nil {
		return s.wrapInsertEventsErr(span, err, events, ref)
	}

	err = sqlpkg.CommitTx(tx)
	if err != nil {
		cqrsotel.RecordError(span, err)
	}

	return err
}

func (s *SQLEventStore) wrapInsertEventsErr(
	span cqrsotel.Span,
	err error,
	events []event.Event,
	ref event.AggregateRef,
) error {
	cqrsotel.RecordError(span, err)

	return event.WrapInfrastructure(err, "storage.insert_events",
		fmt.Sprintf("insert %d events for %s", len(events), ref))
}

func (s *SQLEventStore) checkVersion(
	ctx context.Context,
	tx *sql.Tx,
	ref event.AggregateRef,
	expectedVersion event.Version,
) error {
	p1, p2 := s.Dialect.Placeholder(1), s.Dialect.Placeholder(2)

	query := fmt.Sprintf(sqlpkg.CheckVersionQuery, p1, p2)

	return sqlpkg.SharedCheckVersion(ctx, tx, ref, expectedVersion, query)
}

var (
	_ event.Store           = (*SQLEventStore)(nil)
	_ event.Journal         = (*SQLEventStore)(nil)
	_ event.SeekableJournal = (*SQLEventStore)(nil)
	_ event.BackwardsSource = (*SQLEventStore)(nil)
)
