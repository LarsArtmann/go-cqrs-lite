package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/cockroachdb/errors"
	json "github.com/go-json-experiment/json"
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

func scanEvents(rows *sql.Rows) ([]event.Event, error) {
	var events []event.Event

	for rows.Next() {
		evt, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}

		events = append(events, evt)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate event rows: %w", err)
	}

	return events, nil
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

	parsedAggID, err := id.ParseAggregateID(aggIDStr)
	if err != nil {
		return nil, fmt.Errorf("parse aggregate ID %q: %w", aggIDStr, err)
	}

	parsedEventID, err := id.ParseEventID(idStr)
	if err != nil {
		return nil, fmt.Errorf("parse event ID %q: %w", idStr, err)
	}

	opts := []event.Option{event.WithEventID(parsedEventID), event.WithOccurredAt(occurredAt)}

	metaOpts, err := unmarshalEventMetadata(metadataJSON, eventType)
	if err != nil {
		return nil, err
	}

	opts = append(opts, metaOpts...)

	evt, err := event.NewEvent(
		event.Type(eventType),
		parsedAggID,
		event.AggregateType(aggType),
		version,
		payload,
		opts...)
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

	if err := json.Unmarshal(data, &meta); err != nil {
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
		return nil, errors.Wrap(err, "marshal metadata")
	}

	return data, nil
}

func insertEvents(
	ctx context.Context,
	tx *sql.Tx,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	events []event.Event,
) error {
	for _, evt := range events {
		metadata, err := marshalMetadata(evt.Metadata())
		if err != nil {
			return fmt.Errorf("marshal metadata for event %s: %w", evt.Type(), err)
		}

		_, err = tx.ExecContext(
			ctx,
			insertEventSQL,
			evt.ID(),
			string(evt.Type()),
			string(aggregateType),
			aggregateID,
			evt.Version(),
			evt.Payload(),
			metadata,
			evt.OccurredAt(),
		)
		if err != nil {
			return fmt.Errorf("insert event %s: %w", evt.Type(), err)
		}
	}

	return nil
}
