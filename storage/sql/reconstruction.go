package sql

import (
	"database/sql"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
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
		return nil, event.WrapInfrastructure(err, "storage.iterate_rows",
			"iterate rows")
	}

	return result, nil
}

// ReconstructEvent rebuilds an event.ImmutableEvent from database row fields.
func ReconstructEvent(
	eventID id.EventID,
	eventType event.Type,
	aggType event.AggregateType,
	aggID id.AggregateID,
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

// MarshalMetadata serializes event metadata to JSON.
func MarshalMetadata(m event.Metadata) ([]byte, error) {
	return event.MarshalMetadataJSON(m, "storage.marshal_metadata")
}

// CommitTx commits a transaction, wrapping errors with infrastructure context.
func CommitTx(tx *sql.Tx) error {
	err := tx.Commit()
	if err != nil {
		return event.WrapInfrastructure(err, "storage.commit_tx",
			"commit transaction")
	}

	return nil
}
