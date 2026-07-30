package sql

import (
	"database/sql"
	"encoding/json/v2"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// Schema returns the SQL DDL for creating the events table (PostgreSQL).
func Schema() string {
	pg := PostgresDialect{}
	return pg.EventSchema()
}

// SQLiteSchema returns the SQL DDL for creating the events table (SQLite).
func SQLiteSchema() string {
	sqlite := SQLiteDialect{}
	return sqlite.EventSchema()
}

// CloseRows closes *sql.Rows, suppressing the always-nil error. Intended for
// use with defer: `defer sqlpkg.CloseRows(rows)`. Replaces the verbose
// `defer func() { _ = rows.Close() }()` idiom.
func CloseRows(rows *sql.Rows) {
	//cqrs-lint:ignore(C023) helper exists to suppress the error — see doc comment
	//cqrs-lint:ignore(C015) library code or intentional pattern
	_ = rows.Close()
}

// AppendLimit appends a " LIMIT <placeholder>" clause to the query and the
// limit value to args when limit > 0. Returns the query and args unchanged
// when limit is zero or negative.
func AppendLimit(query string, args []any, limit int, placeholder string) (string, []any) {
	if limit > 0 {
		return query + " LIMIT " + placeholder, append(args, limit)
	}

	return query, args
}

// ScanSlice is a generic helper that deduplicates event scanning.
func ScanSlice[T any](rows *sql.Rows, fn func(*sql.Rows) (T, error)) ([]T, error) {
	result := make([]T, 0, 64)

	for rows.Next() {
		item, err := fn(rows)
		if err != nil {
			return nil, err
		}

		result = append(result, item)
	}

	err := rows.Err()
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "storage.iterate_rows",
			"iterate rows")
	}

	return result, nil
}

// ReconstructEvent rebuilds an event.ImmutableEvent from database row fields.
func ReconstructEvent(
	eventID id.EventID,
	eventType event.Type,
	aggType id.StreamType,
	aggID id.StreamID,
	version, schemaVersion int,
	payload, metadataJSON []byte,
	occurredAt time.Time,
	encoding codec.Encoding,
) (event.Event, error) {
	return event.ReconstructEventFromFields(
		eventID, eventType, aggType, aggID,
		version, schemaVersion, payload, metadataJSON,
		occurredAt, encoding, "storage",
	)
}

// UnmarshalEventMetadata parses metadata JSON into event options.
func UnmarshalEventMetadata(data []byte, eventType string) ([]event.Option, error) {
	return event.UnmarshalMetadataJSON(data, "storage.unmarshal_metadata", eventType)
}

// MarshalMetadata serializes any metadata value to JSON.
// Accepts event.Metadata, command.Metadata, or query.Metadata — all share the
// embedded Tracing + Custom JSON shape, so the SQL layer need not depend on
// any one module's concrete type (ADR-0031).
func MarshalMetadata(m any) ([]byte, error) {
	data, err := json.Marshal(m, json.Deterministic(true))
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, "storage.marshal_metadata", "marshal metadata")
	}

	return data, nil
}

// CommitTx commits a transaction, wrapping errors with infrastructure context.
func CommitTx(tx *sql.Tx) error {
	err := tx.Commit()
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "storage.commit_tx",
			"commit transaction")
	}

	return nil
}
