package sql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// Schema returns the SQL DDL for creating the events and outbox tables (PostgreSQL).
func Schema() string {
	pg := PostgresDialect{}
	return pg.EventSchema() + "\n" + pg.OutboxSchema()
}

// SQLiteSchema returns the SQL DDL for creating the events and outbox tables (SQLite).
func SQLiteSchema() string {
	sqlite := SQLiteDialect{}
	return sqlite.EventSchema() + "\n" + sqlite.OutboxSchema()
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
	if len(data) == 0 {
		return nil, nil
	}

	var meta event.Metadata

	err := json.Unmarshal(data, &meta)
	if err != nil {
		return nil, event.WrapCorruption(err, "storage.unmarshal_metadata",
			"unmarshal metadata for event "+eventType)
	}

	return []event.Option{event.WithMetadata(&meta)}, nil
}

// MarshalMetadata serializes event metadata to JSON.
func MarshalMetadata(m *event.Metadata) ([]byte, error) {
	if m == nil {
		return nil, nil
	}

	data, err := json.Marshal(m)
	if err != nil {
		return nil, event.WrapCorruption(err, "storage.marshal_metadata",
			"marshal metadata")
	}

	return data, nil
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

// SaveWithOutboxTx performs version checking, event insertion, and outbox append in a single transaction.
func SaveWithOutboxTx(
	ctx context.Context,
	db *sql.DB,
	ref event.AggregateRef,
	events []event.Event,
	expectedVersion event.Version,
	checkVersionFn func(context.Context, *sql.Tx, event.AggregateRef, event.Version) error,
	insertEventsFn func(context.Context, *sql.Tx, event.AggregateRef, []event.Event) error,
	appendOutboxFn func(context.Context, *sql.Tx, []event.Event) error,
) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return event.WrapInfrastructure(err, "storage.begin_tx",
			"begin transaction")
	}

	defer func() {
		_ = tx.Rollback()
	}()

	err = checkVersionFn(ctx, tx, ref, expectedVersion)
	if err != nil {
		return event.WrapInfrastructure(err, "storage.check_version",
			fmt.Sprintf("check version for %s %s", ref.Type, ref.ID))
	}

	err = insertEventsFn(ctx, tx, ref, events)
	if err != nil {
		return event.WrapInfrastructure(err, "storage.insert_events",
			fmt.Sprintf("insert %d events for %s %s", len(events), ref.Type, ref.ID))
	}

	err = appendOutboxFn(ctx, tx, events)
	if err != nil {
		return event.WrapInfrastructure(err, "storage.append_outbox",
			fmt.Sprintf("append %d events to outbox", len(events)))
	}

	return CommitTx(tx)
}
