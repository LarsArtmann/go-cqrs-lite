package sql

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec"
	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/id"
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
	var result []T

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
	eventType, aggType string,
	aggID id.AggregateID,
	version, schemaVersion int,
	payload, metadataJSON []byte,
	occurredAt time.Time,
	encoding codec.Encoding,
) (event.Event, error) {
	metaOpts, err := UnmarshalEventMetadata(metadataJSON, eventType)
	if err != nil {
		return nil, event.WrapCorruption(
			err,
			"storage.metadata_unmarshal",
			fmt.Sprintf(
				"metadata for %s/%s v%d (schema v%d)",
				aggType,
				eventType,
				version,
				schemaVersion,
			),
		)
	}

	opts := make([]event.Option, 0, 3+len(metaOpts))

	opts = append(opts, event.WithEventID(eventID), event.WithOccurredAt(occurredAt))
	if schemaVersion > 0 {
		opts = append(opts, event.WithSchemaVersion(event.SchemaVersion(schemaVersion)))
	}

	opts = append(opts, metaOpts...)

	if encoding != "" {
		opts = append(opts, event.WithEncoding(encoding))
	}

	evt, err := event.NewEvent(
		event.Type(eventType),
		aggID,
		event.AggregateType(aggType),
		event.Version(version),
		payload,
		opts...,
	)
	if err != nil {
		return nil, event.WrapCorruption(err, "storage.reconstruct_event",
			"reconstruct event "+eventType)
	}

	return evt, nil
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
