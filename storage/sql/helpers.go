package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// DeleteByAggregate is the shared implementation for Delete methods across event
// and snapshot stores.
func DeleteByAggregate(
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

// SharedInsertEvents is the common loop body for inserting events, separated only by the time formatter and SQL template.
func SharedInsertEvents(
	ctx context.Context,
	tx *sql.Tx,
	ref event.AggregateRef,
	events []event.Event,
	sqlQuery string,
	formatTime func(time.Time) any,
) error {
	for _, evt := range events {
		metadata, err := MarshalMetadata(evt.Metadata())
		if err != nil {
			return event.WrapCorruption(err, "storage.marshal_metadata",
				"marshal metadata for event "+string(evt.Type()))
		}

		_, err = tx.ExecContext(
			ctx,
			sqlQuery,
			evt.ID(),
			string(evt.Type()),
			string(ref.Type),
			ref.ID,
			evt.Version(),
			evt.SchemaVersion().Int(),
			evt.Payload(),
			string(evt.Encoding()),
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

// CheckVersionQuery is the SQL query template for checking aggregate version.
const CheckVersionQuery = `SELECT COALESCE(MAX(version), 0) FROM ` + TableEvents + ` WHERE aggregate_type = %s AND aggregate_id = %s`

// SharedCheckVersion is the common implementation for optimistic concurrency checks.
func SharedCheckVersion(
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

// SharedCheckpointLoad returns the last checkpoint for a projection.
func SharedCheckpointLoad(
	ctx context.Context,
	db *sql.DB,
	projectionName string,
	d Dialect,
) (event.Checkpoint, error) {
	query := "SELECT event_id, processed_at FROM " + TableCheckpoints + " WHERE projection_name = " + d.Placeholder(
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

// SharedCheckpointSave persists a checkpoint.
func SharedCheckpointSave(
	ctx context.Context,
	db *sql.DB,
	projectionName string,
	cp event.Checkpoint,
	d Dialect,
) error {
	query := fmt.Sprintf(
		"INSERT INTO "+TableCheckpoints+" (projection_name, event_id, processed_at) VALUES (%s, %s, %s) ON CONFLICT (projection_name) DO UPDATE SET event_id = EXCLUDED.event_id, processed_at = EXCLUDED.processed_at",
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
