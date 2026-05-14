package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

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

// deleteByAggregate is the shared implementation for Delete methods across event
// and snapshot stores. It deduplicates the identical 7-line Delete bodies that
// differ only by placeholder syntax ($1/$2 vs ?) and error message prefix.
func deleteByAggregate(
	db *sql.DB,
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	table string,
	placeholder1 string,
	placeholder2 string,
	what string,
) error {
	query := fmt.Sprintf(
		"DELETE FROM %s WHERE aggregate_type = %s AND aggregate_id = %s",
		table, placeholder1, placeholder2,
	)

	_, err := db.ExecContext(ctx, query, string(aggregateType), aggregateID)
	if err != nil {
		return fmt.Errorf("delete %s for %s %s: %w", what, aggregateType, aggregateID, err)
	}

	return nil
}

// scanSlice is a generic helper that deduplicates scanEvents and sqliteScanEvents.
// Both iterate rows, apply a per-row scan function, and return a slice.
func scanSlice[T any](rows *sql.Rows, fn func(*sql.Rows) (T, error)) ([]T, error) {
	var result []T

	for rows.Next() {
		item, err := fn(rows)
		if err != nil {
			return nil, err
		}

		result = append(result, item)
	}

	err := rows.Err()
	if err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	return result, nil
}

func scanEvents(rows *sql.Rows) ([]event.Event, error) {
	return scanSlice(rows, scanEvent)
}

func scanEvent(rows *sql.Rows) (event.Event, error) {
	var (
		idStr        string
		eventType    string
		aggType      string
		aggIDStr     string
		version      int
		payload      []byte
		metadataJSON []byte
		occurredAt   time.Time
	)

	err := rows.Scan(
		&idStr,
		&eventType,
		&aggType,
		&aggIDStr,
		&version,
		&payload,
		&metadataJSON,
		&occurredAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan event row: %w", err)
	}

	return reconstructEvent(
		idStr,
		eventType,
		aggType,
		aggIDStr,
		version,
		payload,
		metadataJSON,
		occurredAt,
	)
}

func reconstructEvent(
	idStr, eventType, aggType, aggIDStr string,
	version int,
	payload, metadataJSON []byte,
	occurredAt time.Time,
) (event.Event, error) {
	parsedAggID, err := id.ParseAggregateID(aggIDStr)
	if err != nil {
		return nil, fmt.Errorf(
			"parse aggregate ID %q for %s v%d: %w", aggIDStr, aggType, version, err,
		)
	}

	parsedEventID, err := id.ParseEventID(idStr)
	if err != nil {
		return nil, fmt.Errorf("parse event ID %q for %s v%d: %w", idStr, aggType, version, err)
	}

	metaOpts, err := unmarshalEventMetadata(metadataJSON, eventType)
	if err != nil {
		return nil, fmt.Errorf("metadata for %s/%s v%d: %w", aggType, eventType, version, err)
	}

	opts := make([]event.Option, 0, 2+len(metaOpts)) //nolint:mnd
	opts = append(opts, event.WithEventID(parsedEventID), event.WithOccurredAt(occurredAt))
	opts = append(opts, metaOpts...)

	evt, err := event.NewEvent(
		event.Type(eventType),
		parsedAggID,
		event.AggregateType(aggType),
		version,
		payload,
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("reconstruct event %s: %w", eventType, err)
	}

	return evt, nil
}

func unmarshalEventMetadata(data []byte, eventType string) ([]event.Option, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var meta event.Metadata

	err := json.Unmarshal(data, &meta)
	if err != nil {
		return nil, fmt.Errorf("unmarshal metadata for event %s: %w", eventType, err)
	}

	return []event.Option{event.WithMetadata(&meta)}, nil
}

func marshalMetadata(m *event.Metadata) ([]byte, error) {
	if m == nil {
		return nil, nil
	}

	data, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}

	return data, nil
}

// sharedInsertEvents is the common loop body for insertEvents (PostgreSQL) and
// sqliteInsertEvents (SQLite), separated only by the time formatter and SQL template.
func sharedInsertEvents(
	ctx context.Context,
	tx *sql.Tx,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	events []event.Event,
	sql string,
	formatTime func(time.Time) any,
) error {
	for _, evt := range events {
		metadata, err := marshalMetadata(evt.Metadata())
		if err != nil {
			return fmt.Errorf("marshal metadata for event %s: %w", evt.Type(), err)
		}

		_, err = tx.ExecContext(
			ctx,
			sql,
			evt.ID(),
			string(evt.Type()),
			string(aggregateType),
			aggregateID,
			evt.Version(),
			evt.Payload(),
			metadata,
			formatTime(evt.OccurredAt()),
		)
		if err != nil {
			return fmt.Errorf("insert event %s: %w", evt.Type(), err)
		}
	}

	return nil
}

// insertEvents persists events using PostgreSQL's native time.Time.
func insertEvents(
	ctx context.Context,
	tx *sql.Tx,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	events []event.Event,
) error {
	return sharedInsertEvents(
		ctx,
		tx,
		aggregateType,
		aggregateID,
		events,
		insertEventSQL,
		func(t time.Time) any { return t },
	)
}

// checkVersionQuery is the SQL query template for checking aggregate version.
// Placeholders must be in the database driver's native format ($1, $2 for PostgreSQL, ?, ? for SQLite).
const checkVersionQuery = `SELECT COALESCE(MAX(version), 0) FROM events WHERE aggregate_type = %s AND aggregate_id = %s`

// sharedCheckVersion is the common implementation for checkVersion (PostgreSQL) and
// sqliteCheckVersion (SQLite), separated only by the SQL placeholder format.
func sharedCheckVersion(
	ctx context.Context,
	tx *sql.Tx,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	expectedVersion event.Version,
	query string,
) error {
	var currentVersion int

	err := tx.QueryRowContext(ctx, query, string(aggregateType), aggregateID).
		Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("check current version: %w", err)
	}

	if currentVersion != expectedVersion.Int() {
		return fmt.Errorf(
			"%w: expected version %d, got %d for %s %s",
			ErrConcurrencyConflict,
			expectedVersion.Int(),
			currentVersion,
			aggregateType,
			aggregateID,
		)
	}

	return nil
}

// sharedAckBatch deletes outbox entries by ID, using the provided placeholder format.
// placeholderFormat is "$" for PostgreSQL or "?" for SQLite.
func sharedAckBatch(
	ctx context.Context,
	db *sql.DB,
	ids []event.OutboxID,
	placeholderFormat string,
) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))

	for i, oid := range ids {
		placeholders[i] = placeholderFormat + strconv.Itoa(i+1)
		args[i] = string(oid)
	}

	query := fmt.Sprintf("DELETE FROM outbox WHERE id IN (%s)", strings.Join(placeholders, ", "))

	_, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("ack outbox entries: %w", err)
	}

	return nil
}
