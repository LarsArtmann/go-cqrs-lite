package storage

import (
	"context"
	"database/sql"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v3/sql"
)

// SQLEventStore persists events in a SQL database with optimistic concurrency.
type SQLEventStore struct {
	*sqlpkg.OwnedDBHandle

	insertEventSQL string
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
	handle, err := sqlpkg.NewBorrowedDBHandle(db, d)
	if err != nil {
		return nil, err
	}

	return &SQLEventStore{OwnedDBHandle: handle, insertEventSQL: buildInsertEventSQL(d)}, nil
}

// errStoreClosed is a package-level sentinel to avoid allocating a new error
// on every checkClosed call (the hot path for every Save/Load).
var errStoreClosed = errorfamily.NewInfrastructure("storage.closed", "store is closed")

func (s *SQLEventStore) checkClosed() error {
	return s.CheckClosed(errStoreClosed)
}

// HealthCheck verifies the database connection is alive via PingContext.
// Implements the stack.HealthChecker interface for Kubernetes liveness/readiness probes.
func (s *SQLEventStore) HealthCheck(ctx context.Context) error {
	if err := s.checkClosed(); err != nil {
		return err
	}

	return s.DB.PingContext(ctx)
}

// Save persists events with optimistic concurrency check.
func (s *SQLEventStore) Save(
	ctx context.Context,
	ref id.AggregateRef,
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

		return errorfamily.WrapInfrastructure(err, "storage.begin_tx",
			"begin transaction")
	}

	defer func() {
		_ = tx.Rollback()
	}()

	err = s.checkVersion(ctx, tx, ref, expectedVersion)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return errorfamily.WrapInfrastructure(err, "storage.check_version",
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
	ref id.AggregateRef,
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

		return errorfamily.WrapInfrastructure(err, "storage.begin_tx",
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
	ref id.AggregateRef,
) error {
	cqrsotel.RecordError(span, err)

	return errorfamily.WrapInfrastructure(err, "storage.insert_events",
		fmt.Sprintf("insert %d events for %s", len(events), ref))
}

func (s *SQLEventStore) checkVersion(
	ctx context.Context,
	tx *sql.Tx,
	ref id.AggregateRef,
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
