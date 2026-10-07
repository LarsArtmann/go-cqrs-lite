package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

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

		return event.Checkpoint{}, errorfamily.WrapInfrastructure(err, "storage.load_checkpoint",
			"load checkpoint for projection "+projectionName)
	}

	parsed, err := id.ParseEventID(eventIDStr)
	if err != nil {
		return event.Checkpoint{}, errorfamily.WrapCorruption(err, "storage.parse_event_id",
			fmt.Sprintf("parse event ID %q for projection %q", eventIDStr, projectionName))
	}

	processedAt, err := d.ParseTime(processedAtDest)
	if err != nil {
		return event.Checkpoint{}, errorfamily.WrapCorruption(err, "storage.parse_processed_at",
			fmt.Sprintf("parse processed_at for projection %q", projectionName))
	}

	return event.Checkpoint{EventID: parsed, ProcessedAt: processedAt}, nil
}

// SharedCheckpointSave persists a checkpoint.
//
// A zero checkpoint (no prior progress) DELETES the stored row instead of
// inserting: the schema declares event_id NOT NULL, so a zero EventID —
// which serializes to SQL NULL — can never be stored. Row absence is the
// canonical "no progress" state and SharedCheckpointLoad already maps
// ErrNoRows to a zero checkpoint, so the round-trip stays consistent.
// projectionhost.Host.Reset relies on this to clear a checkpoint.
func SharedCheckpointSave(
	ctx context.Context,
	db *sql.DB,
	projectionName string,
	cp event.Checkpoint,
	d Dialect,
) error {
	if cp.IsZero() {
		query := "DELETE FROM " + TableCheckpoints + " WHERE projection_name = " + d.Placeholder(1)

		if _, err := db.ExecContext(ctx, query, projectionName); err != nil {
			return errorfamily.WrapInfrastructure(err, "storage.save_checkpoint",
				"clear checkpoint for projection "+projectionName)
		}

		return nil
	}

	setExprs := []string{
		"event_id = " + d.ExcludedRef("event_id"),
		"processed_at = " + d.ExcludedRef("processed_at"),
	}
	query := fmt.Sprintf(
		"INSERT INTO "+TableCheckpoints+" (projection_name, event_id, processed_at) VALUES (%s, %s, %s) %s",
		d.Placeholder(1),
		d.Placeholder(2),
		d.Placeholder(3),
		d.OnConflictDoUpdate([]string{"projection_name"}, setExprs),
	)

	_, err := db.ExecContext(ctx, query, projectionName, cp.EventID, d.FormatTime(cp.ProcessedAt))
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "storage.save_checkpoint",
			"save checkpoint for projection "+projectionName)
	}

	return nil
}
