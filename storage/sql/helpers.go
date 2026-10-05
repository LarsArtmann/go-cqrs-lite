package sql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// DeleteByStream is the shared implementation for Delete methods on stores
// whose tables use the v5 stream column vocabulary (snapshots).
func DeleteByStream(
	db *sql.DB,
	ctx context.Context,
	ref id.StreamRef,
	table string,
	placeholder1 string,
	placeholder2 string,
	what string,
) error {
	query := fmt.Sprintf(
		"DELETE FROM %s WHERE stream_type = %s AND stream_id = %s",
		table, placeholder1, placeholder2,
	)

	_, err := db.ExecContext(ctx, query, string(ref.Type), ref.ID)
	if err != nil {
		return errorfamily.WrapInfrastructure(
			err,
			"storage.delete_by_stream",
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
	ref id.StreamRef,
	events []event.Event,
	sqlQuery string,
	formatTime func(time.Time) any,
) error {
	for _, evt := range events {
		metadata, err := MarshalMetadata(evt.Metadata())
		if err != nil {
			return errorfamily.WrapCorruption(err, "storage.marshal_metadata",
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
			event.PayloadReadOnly(evt),
			string(evt.Encoding()),
			metadata,
			formatTime(evt.OccurredAt()),
		)
		if err != nil {
			return errorfamily.WrapInfrastructure(err, "storage.insert_event",
				"insert event "+string(evt.Type()))
		}
	}

	return nil
}

const eventColumnsPerRow = 10

// maxSQLiteParameters is SQLite's hard per-statement host-parameter limit.
const maxSQLiteParameters = 999

// maxNonSQLiteParameters bounds bound parameters per statement for dialects
// without SQLite's 999 limit. PostgreSQL (extended protocol) and MySQL
// (prepared statements) both accept 65535 placeholders; DuckDB has no
// documented hard limit. 32767 stays comfortably inside every protocol limit
// while bounding single-statement parse cost and memory.
const maxNonSQLiteParameters = 32767

// MaxParametersForDialect returns the maximum number of bound parameters a
// single statement may carry for the given dialect. SQLite is capped at 999;
// PostgreSQL, MySQL, and DuckDB get 32767. Unknown dialects keep the
// conservative SQLite limit so custom Dialect implementations stay correct by
// default (custom dialects can inherit the wider limit by embedding one of
// the known dialects).
func MaxParametersForDialect(dialect Dialect) int {
	switch dialect.(type) {
	case SQLiteDialect:
		return maxSQLiteParameters
	case PostgresDialect, MySQLDialect, DuckDBDialect:
		return maxNonSQLiteParameters
	default:
		return maxSQLiteParameters
	}
}

// SharedBatchInsertEvents inserts multiple events using a single multi-VALUES
// INSERT statement, reducing network round-trips for batch writes.
// Events are chunked to respect BOTH the dialect's bound-parameter limit
// (999 for SQLite, 32767 for PostgreSQL/MySQL/DuckDB) and an estimated
// serialized-statement byte cap (MaxStatementBytes), so large payloads shrink
// chunks instead of exceeding MariaDB/MySQL max_allowed_packet.
func SharedBatchInsertEvents(
	ctx context.Context,
	tx *sql.Tx,
	ref id.StreamRef,
	events []event.Event,
	dialect Dialect,
	formatTime func(time.Time) any,
) error {
	if len(events) == 0 {
		return nil
	}

	metadataJSON, err := marshalAllEventMetadata(events)
	if err != nil {
		return err
	}

	maxPerBatch := MaxParametersForDialect(dialect) / eventColumnsPerRow

	for start := 0; start < len(events); {
		rows := RowsWithinByteCap(start, len(events), maxPerBatch, func(i int) int {
			return len(
				event.PayloadReadOnly(events[i]),
			) + len(
				metadataJSON[i],
			) + bytesPerEventOverhead
		})

		err := insertMultiValues(
			ctx,
			tx,
			ref,
			events[start:start+rows],
			metadataJSON[start:start+rows],
			dialect,
			formatTime,
		)
		if err != nil {
			return err
		}
		start += rows
	}

	return nil
}

func marshalAllEventMetadata(events []event.Event) ([][]byte, error) {
	marshaled := make([][]byte, len(events))
	for i, evt := range events {
		metadata, err := MarshalMetadata(evt.Metadata())
		if err != nil {
			return nil, errorfamily.WrapCorruption(err, "storage.marshal_metadata",
				"marshal metadata for event "+string(evt.Type()))
		}
		marshaled[i] = metadata
	}

	return marshaled, nil
}

func insertMultiValues(
	ctx context.Context,
	tx *sql.Tx,
	ref id.StreamRef,
	events []event.Event,
	metadataJSON [][]byte,
	dialect Dialect,
	formatTime func(time.Time) any,
) error {
	n := len(events)
	valueGroups := make([]string, n)
	args := make([]any, 0, n*eventColumnsPerRow)

	for i, evt := range events {
		offset := i * eventColumnsPerRow
		valueGroups[i] = "(" + Placeholders(dialect, eventColumnsPerRow, offset) + ")"

		args = append(
			args,
			evt.ID(),
			string(evt.Type()),
			string(ref.Type),
			ref.ID,
			evt.Version(),
			evt.SchemaVersion().Int(),
			event.PayloadReadOnly(evt),
			string(evt.Encoding()),
			metadataJSON[i],
			formatTime(evt.OccurredAt()),
		)
	}

	query := fmt.Sprintf(
		`INSERT INTO %s (id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, payload_encoding, metadata, occurred_at) VALUES %s`,
		TableEvents,
		strings.Join(valueGroups, ", "),
	)

	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "storage.batch_insert_events",
			fmt.Sprintf("batch insert %d events", n))
	}

	return nil
}

// CheckVersionQuery is the SQL query template for checking aggregate version.
const CheckVersionQuery = `SELECT COALESCE(MAX(version), 0) FROM ` + TableEvents + ` WHERE aggregate_type = %s AND aggregate_id = %s`

// SharedCheckVersion is the common implementation for optimistic concurrency checks.
func SharedCheckVersion(
	ctx context.Context,
	tx *sql.Tx,
	ref id.StreamRef,
	expectedVersion event.Version,
	query string,
) error {
	var currentVersion int

	err := tx.QueryRowContext(ctx, query, string(ref.Type), ref.ID).
		Scan(&currentVersion)
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "storage.check_version",
			"check current version")
	}

	if currentVersion != expectedVersion.Int() {
		return errorfamily.WrapConflict(ErrConcurrencyConflict, "storage.version_mismatch",
			fmt.Sprintf("expected version %d, got %d for %s %s",
				expectedVersion.Int(), currentVersion, ref.Type, ref.ID))
	}

	return nil
}

// Deprecated: use DeleteByStream.
func DeleteByAggregate(
	db *sql.DB,
	ctx context.Context,
	ref id.StreamRef,
	table string,
	placeholder1 string,
	placeholder2 string,
	what string,
) error {
	return DeleteByStream(db, ctx, ref, table, placeholder1, placeholder2, what)
}
