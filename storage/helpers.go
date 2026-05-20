package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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
    schema_version  INTEGER NOT NULL DEFAULT 1,
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

// SQLiteSchema returns the SQL DDL for creating the events table in SQLite.
func SQLiteSchema() string {
	return `CREATE TABLE IF NOT EXISTS events (
    id              TEXT PRIMARY KEY,
    event_type      TEXT NOT NULL,
    aggregate_type  TEXT NOT NULL,
    aggregate_id    TEXT NOT NULL,
    version         INTEGER NOT NULL,
    schema_version  INTEGER NOT NULL DEFAULT 1,
    payload         BLOB,
    metadata        TEXT,
    occurred_at     TEXT NOT NULL,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(aggregate_type, aggregate_id, version)
);

CREATE INDEX IF NOT EXISTS idx_events_aggregate ON events(aggregate_type, aggregate_id);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_occurred_at ON events(occurred_at);`
}

// scanSlice is a generic helper that deduplicates event scanning.
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

func reconstructEvent(
	idStr, eventType, aggType, aggIDStr string,
	version, schemaVersion int,
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

	opts := make([]event.Option, 0, 3+len(metaOpts))

	opts = append(opts, event.WithEventID(parsedEventID), event.WithOccurredAt(occurredAt))
	if schemaVersion > 0 {
		opts = append(opts, event.WithSchemaVersion(event.SchemaVersion(schemaVersion)))
	}

	opts = append(opts, metaOpts...)

	evt, err := event.NewEvent(
		event.Type(eventType),
		parsedAggID,
		event.AggregateType(aggType),
		event.Version(version),
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

func commitTx(tx *sql.Tx) error {
	err := tx.Commit()
	if err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// saveWithOutboxTx is the shared implementation for SaveWithOutbox.
// It performs version checking, event insertion, and outbox append in a single transaction.
func saveWithOutboxTx(
	ctx context.Context,
	db *sql.DB,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	events []event.Event,
	expectedVersion event.Version,
	checkVersionFn func(context.Context, *sql.Tx, event.AggregateType, id.AggregateID, event.Version) error,
	insertEventsFn func(context.Context, *sql.Tx, event.AggregateType, id.AggregateID, []event.Event) error,
	appendOutboxFn func(context.Context, *sql.Tx, []event.Event) error,
) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	err = checkVersionFn(ctx, tx, aggregateType, aggregateID, expectedVersion)
	if err != nil {
		return err
	}

	err = insertEventsFn(ctx, tx, aggregateType, aggregateID, events)
	if err != nil {
		return err
	}

	err = appendOutboxFn(ctx, tx, events)
	if err != nil {
		return err
	}

	return commitTx(tx)
}
