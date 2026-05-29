package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// Schema returns the SQL DDL for creating the events and outbox tables.
func Schema() string {
	pg := PostgresDialect{}
	return pg.EventSchema() + "\n" + pg.OutboxSchema()
}

// SQLiteSchema returns the SQL DDL for creating the events and outbox tables in SQLite.
func SQLiteSchema() string {
	sqlite := SQLiteDialect{}
	return sqlite.EventSchema() + "\n" + sqlite.OutboxSchema()
}

// scanSlice is a generic helper that deduplicates event scanning.
func scanSlice[T any](rows *sql.Rows, fn func(*sql.Rows) (T, error)) ([]T, error) {
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

func reconstructEvent(
	eventID id.EventID,
	eventType, aggType string,
	aggID id.AggregateID,
	version, schemaVersion int,
	payload, metadataJSON []byte,
	occurredAt time.Time,
) (event.Event, error) {
	metaOpts, err := unmarshalEventMetadata(metadataJSON, eventType)
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

func unmarshalEventMetadata(data []byte, eventType string) ([]event.Option, error) {
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

func marshalMetadata(m *event.Metadata) ([]byte, error) {
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

func commitTx(tx *sql.Tx) error {
	err := tx.Commit()
	if err != nil {
		return event.WrapInfrastructure(err, "storage.commit_tx",
			"commit transaction")
	}

	return nil
}

// saveWithOutboxTx is the shared implementation for SaveWithOutbox.
// It performs version checking, event insertion, and outbox append in a single transaction.
func saveWithOutboxTx(
	ctx context.Context,
	db *sql.DB,
	ref event.AggregateRef,
	events []event.Event,
	expectedVersion event.Version,
	checkVersionFn func(context.Context, *sql.Tx, event.AggregateType, id.AggregateID, event.Version) error,
	insertEventsFn func(context.Context, *sql.Tx, event.AggregateType, id.AggregateID, []event.Event) error,
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

	err = checkVersionFn(ctx, tx, aggregateType, aggregateID, expectedVersion)
	if err != nil {
		return event.WrapInfrastructure(err, "storage.check_version",
			fmt.Sprintf("check version for %s %s", aggregateType, aggregateID))
	}

	err = insertEventsFn(ctx, tx, aggregateType, aggregateID, events)
	if err != nil {
		return event.WrapInfrastructure(err, "storage.insert_events",
			fmt.Sprintf("insert %d events for %s %s", len(events), aggregateType, aggregateID))
	}

	err = appendOutboxFn(ctx, tx, events)
	if err != nil {
		return event.WrapInfrastructure(err, "storage.append_outbox",
			fmt.Sprintf("append %d events to outbox", len(events)))
	}

	return commitTx(tx)
}
