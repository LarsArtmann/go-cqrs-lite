package eventstore

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/larsartmann/go-codec"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

func (s *SQLEventStore) scanEvents(rows *sql.Rows, capacityHint int) ([]event.Event, error) {
	return sqlpkg.ScanSlice(rows, s.scanEvent, capacityHint)
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
	parsedAggID, err := id.ParseStreamID(aggIDStr)
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, "storage.parse_aggregate_id",
			fmt.Sprintf("parse stream ID %q for %s v%d", aggIDStr, aggType, version))
	}
	return sqlpkg.ReconstructEvent(
		parsedEventID,
		event.Type(eventType),
		id.StreamType(aggType),
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
	ref id.StreamRef,
	events []event.Event,
) error {
	return sqlpkg.SharedBatchInsertEvents(ctx, tx, ref, events, s.Dialect, s.Dialect.FormatTime)
}
