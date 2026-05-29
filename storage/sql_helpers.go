package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
	ref event.AggregateRef,
	table string,
	placeholder1 string,
	placeholder2 string,
	what string,
) error {
	query := fmt.Sprintf(
		"DELETE FROM %s WHERE aggregate_type = %s AND aggregate_id = %s",
		table, placeholder1, placeholder2,
	)

	_, err := db.ExecContext(ctx, query, string(ref.Type), ref.ID)
	if err != nil {
		return event.WrapInfrastructure(
			err,
			"storage.delete_by_aggregate",
			fmt.Sprintf(
				"delete %s from table %s for %s %s",
				what,
				table,
				ref.Type,
				ref.ID,
			),
		)
	}

	return nil
}

// sharedInsertEvents is the common loop body for insertEvents (PostgreSQL) and
// sqliteInsertEvents (SQLite), separated only by the time formatter and SQL template.
func sharedInsertEvents(
	ctx context.Context,
	tx *sql.Tx,
	ref event.AggregateRef,
	events []event.Event,
	sql string,
	formatTime func(time.Time) any,
) error {
	for _, evt := range events {
		metadata, err := marshalMetadata(evt.Metadata())
		if err != nil {
			return event.WrapCorruption(err, "storage.marshal_metadata",
				"marshal metadata for event "+string(evt.Type()))
		}

		_, err = tx.ExecContext(
			ctx,
			sql,
			evt.ID(),
			string(evt.Type()),
			string(ref.Type),
			ref.ID,
			evt.Version(),
			evt.SchemaVersion().Int(),
			evt.Payload(),
			metadata,
			formatTime(evt.OccurredAt()),
		)
		if err != nil {
			return event.WrapInfrastructure(err, "storage.insert_event",
				"insert event "+string(evt.Type()))
		}
	}

	return nil
}

// checkVersionQuery is the SQL query template for checking aggregate version.
// Placeholders must be in the database driver's native format ($1, $2 for PostgreSQL, ?, ? for SQLite).
const checkVersionQuery = `SELECT COALESCE(MAX(version), 0) FROM ` + tableEvents + ` WHERE aggregate_type = %s AND aggregate_id = %s`

// sharedCheckVersion is the common implementation for checkVersion (PostgreSQL) and
// sqliteCheckVersion (SQLite), separated only by the SQL placeholder format.
func sharedCheckVersion(
	ctx context.Context,
	tx *sql.Tx,
	ref event.AggregateRef,
	expectedVersion event.Version,
	query string,
) error {
	var currentVersion int

	err := tx.QueryRowContext(ctx, query, string(ref.Type), ref.ID).
		Scan(&currentVersion)
	if err != nil {
		return event.WrapInfrastructure(err, "storage.check_version",
			"check current version")
	}

	if currentVersion != expectedVersion.Int() {
		return event.WrapConflict(ErrConcurrencyConflict, "storage.version_mismatch",
			fmt.Sprintf("expected version %d, got %d for %s %s",
				expectedVersion.Int(), currentVersion, ref.Type, ref.ID))
	}

	return nil
}

// sharedCheckpointLoad returns the last checkpoint for a projection
// using the provided placeholder format.
func sharedCheckpointLoad(
	ctx context.Context,
	db *sql.DB,
	projectionName string,
	d Dialect,
) (event.Checkpoint, error) {
	query := "SELECT event_id, processed_at FROM " + tableCheckpoints + " WHERE projection_name = " + d.Placeholder(
		1,
	)

	var eventIDStr string
	processedAtDest := d.ScanTimeDest()

	err := db.QueryRowContext(ctx, query, projectionName).Scan(&eventIDStr, processedAtDest)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return event.Checkpoint{}, nil
		}

		return event.Checkpoint{}, event.WrapInfrastructure(err, "storage.load_checkpoint",
			"load checkpoint for projection "+projectionName)
	}

	parsed, err := id.ParseEventID(eventIDStr)
	if err != nil {
		return event.Checkpoint{}, event.WrapCorruption(err, "storage.parse_event_id",
			fmt.Sprintf("parse event ID %q for projection %q", eventIDStr, projectionName))
	}

	processedAt, err := d.ParseTime(processedAtDest)
	if err != nil {
		return event.Checkpoint{}, event.WrapCorruption(err, "storage.parse_processed_at",
			fmt.Sprintf("parse processed_at for projection %q", projectionName))
	}

	return event.Checkpoint{EventID: parsed, ProcessedAt: processedAt}, nil
}

// sharedCheckpointSave persists a checkpoint using the provided placeholder format.
// placeholderFormat is "$" for PostgreSQL or "?" for SQLite.
func sharedCheckpointSave(
	ctx context.Context,
	db *sql.DB,
	projectionName string,
	cp event.Checkpoint,
	d Dialect,
) error {
	query := fmt.Sprintf(
		"INSERT INTO "+tableCheckpoints+" (projection_name, event_id, processed_at) VALUES (%s, %s, %s) ON CONFLICT (projection_name) DO UPDATE SET event_id = EXCLUDED.event_id, processed_at = EXCLUDED.processed_at",
		d.Placeholder(1),
		d.Placeholder(2),
		d.Placeholder(3),
	)

	_, err := db.ExecContext(ctx, query, projectionName, cp.EventID, d.FormatTime(cp.ProcessedAt))
	if err != nil {
		return event.WrapInfrastructure(err, "storage.save_checkpoint",
			"save checkpoint for projection "+projectionName)
	}

	return nil
}

// sharedAckBatch deletes outbox entries by ID, using the provided dialect for placeholders.
func sharedAckBatch(
	ctx context.Context,
	db *sql.DB,
	ids []event.OutboxID,
	d Dialect,
) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))

	for i, oid := range ids {
		placeholders[i] = d.Placeholder(i + 1)
		args[i] = oid.Get()
	}

	query := fmt.Sprintf(
		"DELETE FROM "+tableOutbox+" WHERE id IN (%s)",
		strings.Join(placeholders, ", "),
	)

	_, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return event.WrapInfrastructure(err, "storage.ack_outbox",
			"ack outbox entries")
	}

	return nil
}

func outboxInsertSQL(dialect Dialect) string {
	p1, p2, p3, p4 := dialect.Placeholder(
		1,
	), dialect.Placeholder(
		2,
	), dialect.Placeholder(
		3,
	), dialect.Placeholder(
		4,
	)

	return fmt.Sprintf(
		`INSERT INTO `+tableOutbox+` (id, status, events, created_at) VALUES (%s, %s, %s, %s)`,
		p1, p2, p3, p4,
	)
}
