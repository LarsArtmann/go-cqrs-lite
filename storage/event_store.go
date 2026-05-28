package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// SQLEventStore persists events in a SQL database with optimistic concurrency.
type SQLEventStore struct {
	sqlBase

	ownDB bool
}

// NewSQLEventStore creates a new SQL-backed event store using PostgreSQL dialect.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLEventStore(db *sql.DB) (*SQLEventStore, error) {
	return newSQLEventStoreWithDialect(db, PostgresDialect{})
}

// NewSQLiteEventStore creates a new SQLite-backed event store.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLiteEventStore(db *sql.DB) (*SQLEventStore, error) {
	return newSQLEventStoreWithDialect(db, SQLiteDialect{})
}

// NewSQLEventStoreWithDialect creates a new SQL-backed event store with a custom dialect.
// This enables consumers to use any SQL backend (MySQL, CockroachDB, etc.) by implementing the Dialect interface.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLEventStoreWithDialect(db *sql.DB, d Dialect) (*SQLEventStore, error) {
	return newSQLEventStoreWithDialect(db, d)
}

func newSQLEventStoreWithDialect(db *sql.DB, d Dialect) (*SQLEventStore, error) {
	base, err := newSQLBase(db, d)
	if err != nil {
		return nil, err
	}

	return &SQLEventStore{sqlBase: base}, nil
}

// ErrConcurrencyConflict indicates an optimistic concurrency violation.
// Alias of event.ErrVersionConflict for unified errors.Is checking.
var ErrConcurrencyConflict = event.ErrVersionConflict

// Close closes the store. If WithOwnership was set, also closes the underlying *sql.DB.
func (s *SQLEventStore) Close() error {
	if s.ownDB {
		return s.db.Close()
	}

	return nil
}

// Save persists events with optimistic concurrency check.
func (s *SQLEventStore) Save(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	events []event.Event,
	expectedVersion event.Version,
) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return event.WrapInfrastructure(err, "storage.begin_tx",
			"begin transaction")
	}

	defer func() {
		_ = tx.Rollback()
	}()

	err = s.checkVersion(ctx, tx, aggregateType, aggregateID, expectedVersion)
	if err != nil {
		return event.WrapInfrastructure(err, "storage.check_version",
			fmt.Sprintf("check version for %s %s", aggregateType, aggregateID))
	}

	err = s.insertEvents(ctx, tx, aggregateType, aggregateID, events)
	if err != nil {
		return event.WrapInfrastructure(err, "storage.insert_events",
			fmt.Sprintf("insert %d events for %s %s", len(events), aggregateType, aggregateID))
	}

	return commitTx(tx)
}

// AppendBatch appends events without optimistic concurrency checks.
// All events are inserted in a single transaction for atomicity.
func (s *SQLEventStore) AppendBatch(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	events []event.Event,
) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return event.WrapInfrastructure(err, "storage.begin_tx",
			"begin transaction")
	}

	defer func() {
		_ = tx.Rollback()
	}()

	err = s.insertEvents(ctx, tx, aggregateType, aggregateID, events)
	if err != nil {
		return event.WrapInfrastructure(err, "storage.insert_events",
			fmt.Sprintf("insert %d events for %s %s", len(events), aggregateType, aggregateID))
	}

	return commitTx(tx)
}

func (s *SQLEventStore) checkVersion(
	ctx context.Context,
	tx *sql.Tx,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	expectedVersion event.Version,
) error {
	p1, p2 := s.dialect.Placeholder(1), s.dialect.Placeholder(2)

	query := fmt.Sprintf(checkVersionQuery, p1, p2)

	return sharedCheckVersion(ctx, tx, aggregateType, aggregateID, expectedVersion, query)
}

var (
	_ event.Store            = (*SQLEventStore)(nil)
	_ event.GlobalLoader     = (*SQLEventStore)(nil)
	_ event.PositionalLoader = (*SQLEventStore)(nil)
	_ event.BackwardsLoader  = (*SQLEventStore)(nil)
)
