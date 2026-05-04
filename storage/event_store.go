// Package storage provides SQL-backed implementations of core event interfaces.
//
// Design decisions:
//   - Accepts *sql.DB for maximum flexibility (connection pooling, transactions)
//   - Uses parameterized queries for SQL injection prevention
//   - Supports optimistic concurrency via version checking
//   - DDL targets PostgreSQL (BYTEA, JSONB, TIMESTAMP WITH TIME ZONE)
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
	db *sql.DB
}

// NewSQLEventStore creates a new SQL-backed event store.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLEventStore(db *sql.DB) (*SQLEventStore, error) {
	if db == nil {
		return nil, fmt.Errorf("%w", ErrNilDB)
	}

	return &SQLEventStore{db: db}, nil
}

// ErrConcurrencyConflict indicates an optimistic concurrency violation.
// Alias of event.ErrVersionConflict for unified errors.Is checking.
var ErrConcurrencyConflict = event.ErrVersionConflict

const insertEventSQL = `INSERT INTO events (id, event_type, aggregate_type, aggregate_id, version, payload, metadata, occurred_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

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

	err = checkVersion(ctx, tx, aggregateType, aggregateID, expectedVersion)
	if err != nil {
		return err
	}

	err = insertEvents(ctx, tx, aggregateType, aggregateID, events)
	if err != nil {
		return err
	}

	return commitTx(tx)
}

func checkVersion(
	ctx context.Context,
	tx *sql.Tx,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	expectedVersion event.Version,
) error {
	var currentVersion int

	query := `SELECT COALESCE(MAX(version), 0) FROM events WHERE aggregate_type = $1 AND aggregate_id = $2`

	err := tx.QueryRowContext(ctx, query, string(aggregateType), aggregateID).
		Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("check current version: %w", err)
	}

	if currentVersion != expectedVersion.Int() {
		return fmt.Errorf("%w: expected version %d, got %d for %s %s",
			ErrConcurrencyConflict,
			expectedVersion.Int(),
			currentVersion,
			aggregateType,
			aggregateID,
		)
	}

	return nil
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

	err = insertEvents(ctx, tx, aggregateType, aggregateID, events)
	if err != nil {
		return err
	}

	return commitTx(tx)
}

func commitTx(tx *sql.Tx) error {
	err := tx.Commit()
	if err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// Load retrieves all events for an aggregate, ordered by version.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *SQLEventStore) Load(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) ([]event.Event, error) {
	query := `SELECT id, event_type, aggregate_type, aggregate_id, version, payload, metadata, occurred_at
		FROM events
		WHERE aggregate_type = $1 AND aggregate_id = $2
		ORDER BY version ASC`

	rows, err := s.db.QueryContext(ctx, query, string(aggregateType), aggregateID)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	events, err := scanEvents(rows)
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
	query := `SELECT id, event_type, aggregate_type, aggregate_id, version, payload, metadata, occurred_at
		FROM events
		WHERE aggregate_type = $1 AND aggregate_id = $2 AND version > $3
		ORDER BY version ASC`

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

	return scanEvents(rows)
}

// Delete removes all events for an aggregate.
func (s *SQLEventStore) Delete(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) error {
	query := `DELETE FROM events WHERE aggregate_type = $1 AND aggregate_id = $2`

	_, err := s.db.ExecContext(ctx, query, string(aggregateType), aggregateID)
	if err != nil {
		return fmt.Errorf("delete events for %s %s: %w", aggregateType, aggregateID, err)
	}

	return nil
}

var (
	_ event.Store        = (*SQLEventStore)(nil)
	_ event.GlobalLoader = (*SQLEventStore)(nil)
)

// LoadAll retrieves all events across all aggregates, ordered by occurrence time.
// Returns an empty slice (not an error) if no events exist.
func (s *SQLEventStore) LoadAll(ctx context.Context) ([]event.Event, error) {
	query := `SELECT id, event_type, aggregate_type, aggregate_id, version, payload, metadata, occurred_at
		FROM events
		ORDER BY occurred_at ASC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query all events: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	events, err := scanEvents(rows)
	if err != nil {
		return nil, err
	}

	return events, nil
}
