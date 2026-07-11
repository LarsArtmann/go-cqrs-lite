package projectionhost

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

var (
	_ DeadLetterStore      = (*SQLiteDeadLetterStore)(nil)
	_ DeadLetterStoreAdmin = (*SQLiteDeadLetterStore)(nil)
)

// intFromVersion converts an event.Version (uint64) to int for SQL storage.
// Event versions are small sequential integers that never approach int32 max.
func intFromVersion(v event.Version) int {
	return int(v) //nolint:gosec // G115: versions are small sequential integers
}

// sqliteDLQSchema defines the projection dead-letter table and its indexes.
//
// Index audit (ADR feedback): three indexes cover all access patterns:
//   - UNIQUE(projection_name, event_id): point lookups for Store (INSERT OR
//     REPLACE conflict) and Delete.
//   - idx_pdl_projection_time(projection_name, failed_at): filtered listings
//     (List, ListPaged, Purge by projection). projection_name is the leftmost
//     column so single-column projection lookups also use this index.
//   - idx_pdl_failed_at(failed_at): global ORDER BY and time-bounded PurgeBefore.
//
// failed_at is TEXT (RFC3339Nano) — lexicographic order matches chronological
// order, so index range scans are valid.
const sqliteDLQSchema = `CREATE TABLE IF NOT EXISTS projection_dead_letters (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    projection_name  TEXT NOT NULL,
    event_id         TEXT NOT NULL,
    event_type       TEXT NOT NULL,
    aggregate_type   TEXT NOT NULL DEFAULT '',
    aggregate_id     TEXT NOT NULL,
    version          INTEGER NOT NULL DEFAULT 0,
    schema_version   INTEGER NOT NULL DEFAULT 1,
    payload          BLOB,
    payload_encoding TEXT NOT NULL DEFAULT 'json',
    metadata         TEXT,
    occurred_at      TEXT NOT NULL,
    error_text       TEXT,
    error_code       TEXT,
    error_family     TEXT,
    failed_at        TEXT NOT NULL,
    UNIQUE(projection_name, event_id)
);
CREATE INDEX IF NOT EXISTS idx_pdl_projection_time ON projection_dead_letters(projection_name, failed_at);
CREATE INDEX IF NOT EXISTS idx_pdl_failed_at ON projection_dead_letters(failed_at);`

// SQLiteDeadLetterStore is a persistent DeadLetterStore backed by SQLite.
// It survives restarts, unlike MemoryDeadLetterStore.
// The *sql.DB is NOT closed by Close — the caller owns it.
type SQLiteDeadLetterStore struct {
	db *sql.DB
}

// NewSQLiteDeadLetterStore creates a SQLite-backed dead-letter store.
// The database must already be open; the caller owns its lifecycle. The table
// is created if it does not exist. The initial schema bootstrap uses ctx.
func NewSQLiteDeadLetterStore(
	ctx context.Context,
	database *sql.DB,
) (*SQLiteDeadLetterStore, error) {
	if database == nil {
		return nil, errorfamily.NewRejection(
			"projectionhost.nil_db", "database handle is required",
		)
	}

	if _, err := database.ExecContext(ctx, sqliteDLQSchema); err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err, "projectionhost.dlq_schema",
			"create projection_dead_letters table",
		)
	}

	return &SQLiteDeadLetterStore{db: database}, nil
}

func (s *SQLiteDeadLetterStore) Store(ctx context.Context, entry DeadLetterEntry) error {
	evt := entry.Event

	var payload, metadataJSON []byte

	var aggType string

	if evt != nil {
		var err error

		metadataJSON, err = event.MarshalMetadataJSON(
			evt.Metadata(), "projectionhost.dlq_marshal",
		)
		if err != nil {
			return errorfamily.WrapInfrastructure(
				err, "projectionhost.dlq_store",
				"marshal metadata for dead-letter entry "+entry.EventID,
			)
		}

		payload = event.PayloadReadOnly(evt)
		aggType = string(evt.AggregateType())
	}

	version, schemaVersion := 0, 1
	encoding, occurredAt := "json", ""

	if evt != nil {
		version = intFromVersion(evt.Version())
		schemaVersion = evt.SchemaVersion().Int()
		encoding = string(evt.Encoding())
		occurredAt = evt.OccurredAt().Format(time.RFC3339Nano)
	}

	_, err := s.db.ExecContext(
		ctx, `INSERT OR REPLACE INTO projection_dead_letters
        (projection_name, event_id, event_type, aggregate_type, aggregate_id,
         version, schema_version, payload, payload_encoding, metadata, occurred_at,
         error_text, error_code, error_family, failed_at)
        VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		entry.ProjectionName,
		entry.EventID,
		entry.EventType,
		aggType,
		entry.AggregateID,
		version,
		schemaVersion,
		payload,
		encoding,
		metadataJSON,
		occurredAt,
		entry.Error,
		entry.ErrorCode,
		entry.ErrorFamily,
		entry.FailedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return errorfamily.WrapInfrastructure(
			err, "projectionhost.dlq_store",
			fmt.Sprintf(
				"store dead-letter entry %s/%s",
				entry.ProjectionName, entry.EventID,
			),
		)
	}

	return nil
}

func (s *SQLiteDeadLetterStore) List(
	ctx context.Context,
	projectionName string,
) ([]DeadLetterEntry, error) {
	query := `SELECT projection_name, event_id, event_type, aggregate_type, aggregate_id,
        version, schema_version, payload, payload_encoding, metadata, occurred_at,
        error_text, error_code, error_family, failed_at
        FROM projection_dead_letters`

	var (
		rows *sql.Rows

		err error
	)

	if projectionName == "" {
		query += " ORDER BY failed_at"
		rows, err = s.db.QueryContext(ctx, query)
	} else {
		query += " WHERE projection_name = ? ORDER BY failed_at"
		rows, err = s.db.QueryContext(ctx, query, projectionName)
	}

	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err, "projectionhost.dlq_list",
			"query dead-letter entries",
		)
	}

	defer func() { _ = rows.Close() }()

	var result []DeadLetterEntry

	for rows.Next() {
		entry, err := scanDLQRow(rows)
		if err != nil {
			return nil, err
		}

		result = append(result, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dead-letter rows: %w", err)
	}

	return result, nil
}

func (s *SQLiteDeadLetterStore) Delete(
	ctx context.Context,
	projectionName, eventID string,
) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM projection_dead_letters WHERE projection_name = ? AND event_id = ?",
		projectionName, eventID)
	if err != nil {
		return errorfamily.WrapInfrastructure(
			err, "projectionhost.dlq_delete",
			fmt.Sprintf("delete dead-letter entry %s/%s", projectionName, eventID),
		)
	}

	return nil
}

func (s *SQLiteDeadLetterStore) Purge(
	ctx context.Context,
	projectionName string,
) error {
	var (
		res sql.Result

		err error
	)

	if projectionName == "" {
		res, err = s.db.ExecContext(ctx, "DELETE FROM projection_dead_letters")
	} else {
		res, err = s.db.ExecContext(ctx,
			"DELETE FROM projection_dead_letters WHERE projection_name = ?",
			projectionName)
	}

	if err != nil {
		return errorfamily.WrapInfrastructure(
			err, "projectionhost.dlq_purge",
			"purge dead-letter entries for "+projectionName,
		)
	}

	_ = res

	return nil
}

func scanDLQRow(rows *sql.Rows) (DeadLetterEntry, error) {
	var (
		entry                      DeadLetterEntry
		aggType                    string
		version, schemaVersion     int
		payload, metadataJSON      []byte
		encoding                   string
		occurredAtStr, failedAtStr string
	)

	err := rows.Scan(
		&entry.ProjectionName,
		&entry.EventID,
		&entry.EventType,
		&aggType,
		&entry.AggregateID,
		&version,
		&schemaVersion,
		&payload,
		&encoding,
		&metadataJSON,
		&occurredAtStr,
		&entry.Error,
		&entry.ErrorCode,
		&entry.ErrorFamily,
		&failedAtStr,
	)
	if err != nil {
		return DeadLetterEntry{}, errorfamily.WrapInfrastructure(
			err, "projectionhost.dlq_scan",
			"scan dead-letter row",
		)
	}

	failedAt, err := time.Parse(time.RFC3339Nano, failedAtStr)
	if err != nil {
		failedAt = time.Time{}
	}

	entry.FailedAt = failedAt

	if len(payload) > 0 || entry.EventID != "" {
		occurredAt, _ := time.Parse(time.RFC3339Nano, occurredAtStr)

		eventID, idErr := id.ParseEventID(entry.EventID)
		if idErr != nil {
			eventID = id.NewEventID()
		}

		aggID, _ := id.ParseAggregateID(entry.AggregateID)

		evt, reconstructErr := event.ReconstructEventFromFields(
			eventID,
			event.Type(entry.EventType),
			id.AggregateType(aggType),
			aggID,
			version,
			schemaVersion,
			payload,
			metadataJSON,
			occurredAt,
			codec.Encoding(encoding),
			"projectionhost",
		)
		if reconstructErr != nil {
			return DeadLetterEntry{}, errorfamily.WrapCorruption(
				reconstructErr,
				"projectionhost.dlq_reconstruct",
				"reconstruct event for dead-letter entry "+entry.EventID,
			)
		}

		entry.Event = evt
	}

	return entry, nil
}
