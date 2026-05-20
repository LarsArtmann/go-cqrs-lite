package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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

// LoadToVersion retrieves events up to and including maxVersion.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *SQLEventStore) LoadToVersion(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	maxVersion event.Version,
) ([]event.Event, error) {
	p1, p2, p3 := s.dialect.Placeholder(1), s.dialect.Placeholder(2), s.dialect.Placeholder(3)

	query := fmt.Sprintf(
		`SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM events
		WHERE aggregate_type = %s AND aggregate_id = %s AND version <= %s
		ORDER BY version ASC`,
		p1,
		p2,
		p3,
	)

	rows, err := s.db.QueryContext(ctx, query, string(aggregateType), aggregateID, maxVersion.Int())
	if err != nil {
		return nil, fmt.Errorf("query events to version: %w", err)
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

// LoadToTimestamp retrieves events where OccurredAt <= maxTime.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *SQLEventStore) LoadToTimestamp(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	maxTime time.Time,
) ([]event.Event, error) {
	p1, p2, p3 := s.dialect.Placeholder(1), s.dialect.Placeholder(2), s.dialect.Placeholder(3)

	query := fmt.Sprintf(
		`SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM events
		WHERE aggregate_type = %s AND aggregate_id = %s AND occurred_at <= %s
		ORDER BY version ASC`,
		p1,
		p2,
		p3,
	)

	rows, err := s.db.QueryContext(
		ctx, query,
		string(aggregateType),
		aggregateID,
		s.dialect.FormatTime(maxTime),
	)
	if err != nil {
		return nil, fmt.Errorf("query events to timestamp: %w", err)
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

// LoadAllFromPosition retrieves events ordered by OccurredAt, starting after the given event ID.
// Returns up to limit events. Implements event.PositionalLoader.
func (s *SQLEventStore) LoadAllFromPosition(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) ([]event.Event, error) {
	if afterEventID.IsZero() {
		return s.LoadAllFromPositionNoLimit(ctx, limit)
	}

	p1 := s.dialect.Placeholder(1)
	p2 := s.dialect.Placeholder(2)

	query := fmt.Sprintf(
		`SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM events
		WHERE id > %s
		ORDER BY occurred_at ASC`,
		p1,
	)

	args := []any{afterEventID.String()}

	if limit > 0 {
		query += " LIMIT " + p2

		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query events from position: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	return s.scanEvents(rows)
}

// LoadAllFromPositionNoLimit loads from the beginning when no position is given.
func (s *SQLEventStore) LoadAllFromPositionNoLimit(
	ctx context.Context,
	limit int,
) ([]event.Event, error) {
	if limit <= 0 {
		return s.LoadAll(ctx)
	}

	p1 := s.dialect.Placeholder(1)

	query := `SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM events
		ORDER BY occurred_at ASC
		LIMIT ` + p1

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query events from start: %w", err)
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

var (
	_ event.Store            = (*SQLEventStore)(nil)
	_ event.GlobalLoader     = (*SQLEventStore)(nil)
	_ event.PositionalLoader = (*SQLEventStore)(nil)
)
