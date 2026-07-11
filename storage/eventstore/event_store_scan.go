package eventstore

import (
	"context"
	"database/sql"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

const eventColumnCount = 10

func (s *SQLEventStore) scanEvents(rows *sql.Rows) ([]event.Event, error) {
	return sqlpkg.ScanSlice(rows, s.scanEvent)
}

func (s *SQLEventStore) scanEvent(rows *sql.Rows) (event.Event, error) {
	var (
		eventIDStr    string
		eventType     string
		aggType       string
		aggIDStr      string
		version       int
		schemaVersion int
		payload       []byte
		encoding      string
		metadataJSON  []byte
	)
	timeDest := s.Dialect.ScanTimeDest()
	err := rows.Scan(
		&eventIDStr,
		&eventType,
		&aggType,
		&aggIDStr,
		&version,
		&schemaVersion,
		&payload,
		&encoding,
		&metadataJSON,
		timeDest,
	)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "storage.scan_event",
			fmt.Sprintf("scan event row for %s/%s v%d (schema v%d) event %s (id %s)",
				aggIDStr, aggType, version, schemaVersion, eventType, eventIDStr))
	}
	occurredAt, err := s.Dialect.ParseTime(timeDest)
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, "storage.parse_occurred_at",
			fmt.Sprintf("parse occurred_at for %s/%s v%d (schema v%d) event %s (id %s)",
				aggIDStr, aggType, version, schemaVersion, eventType, eventIDStr))
	}
	parsedEventID, err := id.ParseEventID(eventIDStr)
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, "storage.parse_event_id",
			fmt.Sprintf("parse event ID %q for %s v%d", eventIDStr, aggType, version))
	}
	parsedAggID, err := id.ParseAggregateID(aggIDStr)
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, "storage.parse_aggregate_id",
			fmt.Sprintf("parse aggregate ID %q for %s v%d", aggIDStr, aggType, version))
	}
	return sqlpkg.ReconstructEvent(
		parsedEventID,
		event.Type(eventType),
		id.AggregateType(aggType),
		parsedAggID,
		version,
		schemaVersion,
		payload,
		metadataJSON,
		occurredAt,
		codec.Encoding(encoding),
	)
}

func (s *SQLEventStore) insertEvents(
	ctx context.Context,
	tx *sql.Tx,
	ref id.AggregateRef,
	events []event.Event,
) error {
	return sqlpkg.SharedBatchInsertEvents(ctx, tx, ref, events, s.Dialect, s.Dialect.FormatTime)
}

func buildInsertEventSQL(d sqlpkg.Dialect) string {
	ph := make([]string, eventColumnCount)
	for i := range eventColumnCount {
		ph[i] = d.Placeholder(i + 1)
	}

	return fmt.Sprintf(
		`INSERT INTO `+sqlpkg.TableEvents+` (id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, payload_encoding, metadata, occurred_at)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)`,
		ph[0],
		ph[1],
		ph[2],
		ph[3],
		ph[4],
		ph[5],
		ph[6],
		ph[7],
		ph[8],
		ph[9],
	)
}
