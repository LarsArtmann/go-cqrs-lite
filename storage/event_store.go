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
	db      *sql.DB
	dialect Dialect
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

func newSQLEventStoreWithDialect(db *sql.DB, d Dialect) (*SQLEventStore, error) {
	if db == nil {
		return nil, fmt.Errorf("%w", ErrNilDB)
	}

	return &SQLEventStore{db: db, dialect: d}, nil
}

// ErrConcurrencyConflict indicates an optimistic concurrency violation.
// Alias of event.ErrVersionConflict for unified errors.Is checking.
var ErrConcurrencyConflict = event.ErrVersionConflict

// Close is a no-op. The *sql.DB is borrowed from the caller, who owns its lifecycle.
func (s *SQLEventStore) Close() error { return nil }

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
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	err = s.checkVersion(ctx, tx, aggregateType, aggregateID, expectedVersion)
	if err != nil {
		return err
	}

	err = s.insertEvents(ctx, tx, aggregateType, aggregateID, events)
	if err != nil {
		return err
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
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	err = s.insertEvents(ctx, tx, aggregateType, aggregateID, events)
	if err != nil {
		return err
	}

	return commitTx(tx)
}

// Load retrieves all events for an aggregate, ordered by version.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *SQLEventStore) Load(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) ([]event.Event, error) {
	p1, p2 := s.dialect.Placeholder(1), s.dialect.Placeholder(2)

	query := fmt.Sprintf(
		`SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM events
		WHERE aggregate_type = %s AND aggregate_id = %s
		ORDER BY version ASC`,
		p1,
		p2,
	)

	rows, err := s.db.QueryContext(ctx, query, string(aggregateType), aggregateID)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	events, err := s.scanEvents(rows)
	if err != nil {
		return nil, err
	}

	if len(events) == 0 {
		return nil, event.ErrAggregateNotFound
	}

	return events, nil
}

// LoadFromVersion retrieves events starting from a given version.
func (s *SQLEventStore) LoadFromVersion(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	version event.Version,
) ([]event.Event, error) {
	p1, p2, p3 := s.dialect.Placeholder(1), s.dialect.Placeholder(2), s.dialect.Placeholder(3)

	query := fmt.Sprintf(
		`SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM events
		WHERE aggregate_type = %s AND aggregate_id = %s AND version > %s
		ORDER BY version ASC`,
		p1,
		p2,
		p3,
	)

	rows, err := s.db.QueryContext(
		ctx,
		query,
		string(aggregateType),
		aggregateID,
		version.Int(),
	)
	if err != nil {
		return nil, fmt.Errorf("query events from version: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	return s.scanEvents(rows)
}

// Delete removes all events for an aggregate.
func (s *SQLEventStore) Delete(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) error {
	p1, p2 := s.dialect.Placeholder(1), s.dialect.Placeholder(2)

	return deleteByAggregate(s.db, ctx, aggregateType, aggregateID, "events", p1, p2, "events")
}

// LoadAll retrieves all events across all aggregates, ordered by occurrence time.
// Returns an empty slice (not an error) if no events exist.
func (s *SQLEventStore) LoadAll(ctx context.Context) ([]event.Event, error) {
	query := `SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM events
		ORDER BY occurred_at ASC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query all events: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	return s.scanEvents(rows)
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

func (s *SQLEventStore) insertEvents(
	ctx context.Context,
	tx *sql.Tx,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	events []event.Event,
) error {
	ph := make([]string, 9)

	for i := range 9 {
		ph[i] = s.dialect.Placeholder(i + 1)
	}

	insertSQL := fmt.Sprintf(
		`INSERT INTO events (id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)`,
		ph[0],
		ph[1],
		ph[2],
		ph[3],
		ph[4],
		ph[5],
		ph[6],
		ph[7],
		ph[8],
	)

	return sharedInsertEvents(
		ctx, tx, aggregateType, aggregateID, events,
		insertSQL, s.dialect.FormatTime,
	)
}

func (s *SQLEventStore) scanEvents(rows *sql.Rows) ([]event.Event, error) {
	return scanSlice(rows, s.scanEvent)
}

func (s *SQLEventStore) scanEvent(rows *sql.Rows) (event.Event, error) {
	var (
		idStr         string
		eventType     string
		aggType       string
		aggIDStr      string
		version       int
		schemaVersion int
		payload       []byte
		metadataJSON  []byte
	)

	timeDest := s.dialect.ScanTimeDest()

	err := rows.Scan(
		&idStr, &eventType, &aggType, &aggIDStr,
		&version, &schemaVersion, &payload, &metadataJSON,
		timeDest,
	)
	if err != nil {
		return nil, fmt.Errorf("scan event row: %w", err)
	}

	occurredAt, err := s.dialect.ParseTime(timeDest)
	if err != nil {
		return nil, fmt.Errorf("parse occurred_at: %w", err)
	}

	return reconstructEvent(
		idStr, eventType, aggType, aggIDStr,
		version, schemaVersion, payload, metadataJSON,
		occurredAt,
	)
}

var (
	_ event.Store        = (*SQLEventStore)(nil)
	_ event.GlobalLoader = (*SQLEventStore)(nil)
)
