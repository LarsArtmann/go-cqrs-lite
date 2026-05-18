package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

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
			evt.SchemaVersion().Int(),
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

// sharedCheckpointLoad returns the last processed event ID for a projection
// using the provided placeholder format.
func sharedCheckpointLoad(
	ctx context.Context,
	db *sql.DB,
	projectionName string,
	placeholder string,
) (id.EventID, error) {
	query := "SELECT event_id FROM checkpoints WHERE projection_name = " + placeholder

	var eventIDStr string

	err := db.QueryRowContext(ctx, query, projectionName).Scan(&eventIDStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return id.EventID{}, nil
		}

		return id.EventID{}, fmt.Errorf(
			"load checkpoint for projection %q: %w",
			projectionName,
			err,
		)
	}

	parsed, err := id.ParseEventID(eventIDStr)
	if err != nil {
		return id.EventID{}, fmt.Errorf(
			"parse event ID %q for projection %q: %w",
			eventIDStr,
			projectionName,
			err,
		)
	}

	return parsed, nil
}

// sharedCheckpointSave persists a checkpoint using the provided placeholder format.
// placeholderFormat is "$" for PostgreSQL or "?" for SQLite.
func sharedCheckpointSave(
	ctx context.Context,
	db *sql.DB,
	projectionName string,
	eventID id.EventID,
	placeholderFormat string,
) error {
	query := fmt.Sprintf(
		"INSERT INTO checkpoints (projection_name, event_id) VALUES (%s, %s) ON CONFLICT (projection_name) DO UPDATE SET event_id = EXCLUDED.event_id",
		placeholderFormat,
		placeholderFormat,
	)

	_, err := db.ExecContext(ctx, query, projectionName, eventID)
	if err != nil {
		return fmt.Errorf("save checkpoint for projection %q: %w", projectionName, err)
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
