package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func sqliteScanEvents(rows *sql.Rows) ([]event.Event, error) {
	return scanSlice(rows, sqliteScanEvent)
}

func sqliteScanEvent(rows *sql.Rows) (event.Event, error) {
	var (
		idStr        string
		eventType    string
		aggType      string
		aggIDStr     string
		version      int
		payload      []byte
		metadataJSON []byte
		occurredAt   string
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

	parsedTime, err := parseSQLiteTimestamp(occurredAt)
	if err != nil {
		return nil, fmt.Errorf("parse occurred_at %q: %w", occurredAt, err)
	}

	return reconstructEvent(
		idStr,
		eventType,
		aggType,
		aggIDStr,
		version,
		payload,
		metadataJSON,
		parsedTime,
	)
}

func sqliteScanSnapshot(
	row *sql.Row,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) (*event.Snapshot, error) {
	var (
		version      int
		stateBytes   []byte
		createdAtStr string
	)

	err := row.Scan(&version, &stateBytes, &createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("scan snapshot for %s/%s: %w", aggregateType, aggregateID, err)
	}

	createdAt, err := parseSQLiteTimestamp(createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at %q: %w", createdAtStr, err)
	}

	return &event.Snapshot{
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
		Version:       event.Version(version),
		State:         stateBytes,
		CreatedAt:     createdAt,
	}, nil
}

func parseSQLiteTimestamp(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}

	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05",
	} {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("%w: %q", ErrUnsupportedTimestamp, s)
}

// SQLiteInitSchema creates all required tables in the SQLite database.
func SQLiteInitSchema(ctx context.Context, db *sql.DB) error {
	for _, ddl := range []string{SQLiteSchema(), SQLiteSnapshotSchema(), SQLiteCheckpointSchema(), SQLiteOutboxSchema()} {
		_, err := db.ExecContext(ctx, ddl)
		if err != nil {
			return fmt.Errorf("exec DDL: %w\nDDL: %s", err, ddl)
		}
	}

	return nil
}
