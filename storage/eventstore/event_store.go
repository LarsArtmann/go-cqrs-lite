package eventstore

import (
	"context"
	"database/sql"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
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

func (s *SQLEventStore) checkClosed() error {
	return s.CheckClosed(sqlpkg.ErrClosed)
}

// Save persists events with optimistic concurrency check.
func (s *SQLEventStore) Save(
	ctx context.Context,
	ref id.StreamRef,
	events []event.Event,
	expectedVersion event.Version,
) error {
	if err := s.checkClosed(); err != nil {
		return err
	}

	streamType, streamID := ref.Type, ref.ID
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
			fmt.Sprintf("check version for %s %s", streamType, streamID))
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
	ref id.StreamRef,
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
			cqrsotel.StreamAttrs(ref.Type, ref.ID),
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

// SaveMultiBatch persists events for multiple streams in a single database
// transaction. All entries are committed atomically — either all succeed or
// none. No optimistic concurrency checks are performed (same semantics as
// AppendBatch). The caller must ensure events carry correct version numbers.
func (s *SQLEventStore) SaveMultiBatch(
	ctx context.Context,
	entries []event.MultiBatchEntry,
) error {
	if err := s.checkClosed(); err != nil {
		return err
	}

	totalEvents := 0
	for _, e := range entries {
		totalEvents += len(e.Events)
	}

	if totalEvents == 0 {
		return nil
	}

	ctx, span := cqrsotel.StartSpan(
		ctx, sqlpkg.Tracer(), "event.store.save_multi_batch",
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(
			cqrsotel.AttrInt(cqrsotel.AttrStreamCount, len(entries)),
			cqrsotel.AttrInt(cqrsotel.AttrEventCount, totalEvents),
		),
	)
	defer span.End()

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return errorfamily.WrapInfrastructure(err, "storage.begin_tx",
			"begin transaction for multi-batch save")
	}

	defer func() {
		_ = tx.Rollback()
	}()

	for _, entry := range entries {
		if len(entry.Events) == 0 {
			continue
		}

		err = s.insertEvents(ctx, tx, entry.Ref, entry.Events)
		if err != nil {
			cqrsotel.RecordError(span, err)

			return errorfamily.WrapInfrastructure(err, "storage.insert_events",
				fmt.Sprintf("insert %d events for %s in multi-batch",
					len(entry.Events), entry.Ref))
		}
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
	ref id.StreamRef,
) error {
	cqrsotel.RecordError(span, err)

	return errorfamily.WrapInfrastructure(err, "storage.insert_events",
		fmt.Sprintf("insert %d events for %s", len(events), ref))
}

func (s *SQLEventStore) checkVersion(
	ctx context.Context,
	tx *sql.Tx,
	ref id.StreamRef,
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
	_ event.MultiSink       = (*SQLEventStore)(nil)
)
