// Package storage provides SQL-backed implementations of core event interfaces.
//
// Design decisions:
//   - Accepts *sql.DB for maximum flexibility (connection pooling, transactions)
//   - Uses parameterized queries for SQL injection prevention
//   - Supports optimistic concurrency via version checking
//   - Designed for PostgreSQL but compatible with any SQL database
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
	codec   event.Codec
	nowFunc func() time.Time
}

// NewSQLEventStore creates a new SQL-backed event store.
func NewSQLEventStore(db *sql.DB, opts ...SQLEventStoreOption) *SQLEventStore {
	s := &SQLEventStore{
		db:      db,
		codec:   event.JSONCodec{},
		nowFunc: time.Now,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// SQLEventStoreOption configures an SQLEventStore.
type SQLEventStoreOption func(*SQLEventStore)

// WithStoreCodec sets the codec for event payload serialization.
func WithStoreCodec(codec event.Codec) SQLEventStoreOption {
	return func(s *SQLEventStore) {
		s.codec = codec
	}
}

// Schema returns the SQL DDL for creating the events table.
func Schema() string {
	return `CREATE TABLE IF NOT EXISTS events (
    id              TEXT PRIMARY KEY,
    event_type      VARCHAR(255) NOT NULL,
    aggregate_type  VARCHAR(255) NOT NULL,
    aggregate_id    TEXT NOT NULL,
    version         INTEGER NOT NULL,
    payload         BYTEA,
    metadata        JSONB,
    occurred_at     TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(aggregate_type, aggregate_id, version)
);

CREATE INDEX IF NOT EXISTS idx_events_aggregate ON events(aggregate_type, aggregate_id);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_occurred_at ON events(occurred_at);`
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
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	var currentVersion int

	query := `SELECT COALESCE(MAX(version), 0) FROM events WHERE aggregate_type = $1 AND aggregate_id = $2`

	err = tx.QueryRowContext(ctx, query, string(aggregateType), aggregateID.String()).
		Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("check current version: %w", err)
	}

	if currentVersion != expectedVersion.Int() {
		return fmt.Errorf(
			"concurrency conflict: expected version %d, got %d for %s %s",
			expectedVersion.Int(),
			currentVersion,
			aggregateType,
			aggregateID.String(),
		)
	}

	insertQuery := `INSERT INTO events (id, event_type, aggregate_type, aggregate_id, version, payload, metadata, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	for _, evt := range events {
		_, err = tx.ExecContext(
			ctx,
			insertQuery,
			evt.ID().String(),
			string(evt.Type()),
			string(aggregateType),
			aggregateID.String(),
			evt.Version(),
			evt.Payload(),
			nil,
			evt.OccurredAt(),
		)
		if err != nil {
			return fmt.Errorf("insert event %s: %w", evt.Type(), err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// AppendBatch appends events without optimistic concurrency checks.
func (s *SQLEventStore) AppendBatch(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	events []event.Event,
) error {
	if len(events) == 0 {
		return nil
	}

	insertQuery := `INSERT INTO events (id, event_type, aggregate_type, aggregate_id, version, payload, metadata, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	for _, evt := range events {
		_, err := s.db.ExecContext(
			ctx,
			insertQuery,
			evt.ID().String(),
			string(evt.Type()),
			string(aggregateType),
			aggregateID.String(),
			evt.Version(),
			evt.Payload(),
			nil,
			evt.OccurredAt(),
		)
		if err != nil {
			return fmt.Errorf("insert event %s: %w", evt.Type(), err)
		}
	}

	return nil
}

// Load retrieves all events for an aggregate, ordered by version.
func (s *SQLEventStore) Load(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) ([]event.Event, error) {
	query := `SELECT id, event_type, aggregate_type, aggregate_id, version, payload, occurred_at
		FROM events
		WHERE aggregate_type = $1 AND aggregate_id = $2
		ORDER BY version ASC`

	rows, err := s.db.QueryContext(ctx, query, string(aggregateType), aggregateID.String())
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	return scanEvents(rows)
}

// LoadFromVersion retrieves events starting from a given version.
func (s *SQLEventStore) LoadFromVersion(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	version event.Version,
) ([]event.Event, error) {
	query := `SELECT id, event_type, aggregate_type, aggregate_id, version, payload, occurred_at
		FROM events
		WHERE aggregate_type = $1 AND aggregate_id = $2 AND version > $3
		ORDER BY version ASC`

	rows, err := s.db.QueryContext(
		ctx,
		query,
		string(aggregateType),
		aggregateID.String(),
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

	_, err := s.db.ExecContext(ctx, query, string(aggregateType), aggregateID.String())
	if err != nil {
		return fmt.Errorf("delete events for %s %s: %w", aggregateType, aggregateID.String(), err)
	}

	return nil
}

func scanEvents(rows *sql.Rows) ([]event.Event, error) {
	var events []event.Event

	for rows.Next() {
		var (
			idStr      string
			eventType  string
			aggType    string
			aggIDStr   string
			version    int
			payload    []byte
			occurredAt time.Time
		)

		err := rows.Scan(&idStr, &eventType, &aggType, &aggIDStr, &version, &payload, &occurredAt)
		if err != nil {
			return nil, fmt.Errorf("scan event row: %w", err)
		}

		parsedAggID, err := id.ParseAggregateID(aggIDStr)
		if err != nil {
			return nil, fmt.Errorf("parse aggregate ID %q: %w", aggIDStr, err)
		}

		evt, err := event.NewEvent(
			event.Type(eventType),
			parsedAggID,
			event.AggregateType(aggType),
			version,
			payload,
		)
		if err != nil {
			return nil, fmt.Errorf("reconstruct event %s: %w", eventType, err)
		}

		events = append(events, evt)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate event rows: %w", err)
	}

	return events, nil
}

var _ event.Store = (*SQLEventStore)(nil)
