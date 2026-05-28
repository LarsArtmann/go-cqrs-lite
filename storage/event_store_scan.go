package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

const eventColumnCount = 9

func (s *SQLEventStore) scanEvents(rows *sql.Rows) ([]event.Event, error) {
	return scanSlice(rows, s.scanEvent)
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
		metadataJSON  []byte
	)

	timeDest := s.dialect.ScanTimeDest()

	err := rows.Scan(
		&eventIDStr, &eventType, &aggType, &aggIDStr,
		&version, &schemaVersion, &payload, &metadataJSON,
		timeDest,
	)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "storage.scan_event",
			fmt.Sprintf("scan event row for %s/%s v%d (schema v%d) event %s (id %s)",
				aggIDStr, aggType, version, schemaVersion, eventType, eventIDStr))
	}

	occurredAt, err := s.dialect.ParseTime(timeDest)
	if err != nil {
		return nil, event.WrapCorruption(err, "storage.parse_occurred_at",
			fmt.Sprintf("parse occurred_at for %s/%s v%d (schema v%d) event %s (id %s)",
				aggIDStr, aggType, version, schemaVersion, eventType, eventIDStr))
	}

	parsedEventID, err := id.ParseEventID(eventIDStr)
	if err != nil {
		return nil, event.WrapCorruption(err, "storage.parse_event_id",
			fmt.Sprintf("parse event ID %q for %s v%d", eventIDStr, aggType, version))
	}

	parsedAggID, err := id.ParseAggregateID(aggIDStr)
	if err != nil {
		return nil, event.WrapCorruption(err, "storage.parse_aggregate_id",
			fmt.Sprintf("parse aggregate ID %q for %s v%d", aggIDStr, aggType, version))
	}

	return reconstructEvent(
		parsedEventID, eventType, aggType, parsedAggID,
		version, schemaVersion, payload, metadataJSON,
		occurredAt,
	)
}

func (s *SQLEventStore) insertEvents(
	ctx context.Context,
	tx *sql.Tx,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	events []event.Event,
) error {
	ph := make([]string, eventColumnCount)

	for i := range eventColumnCount {
		ph[i] = s.dialect.Placeholder(i + 1)
	}

	insertSQL := fmt.Sprintf(
		`INSERT INTO `+tableEvents+` (id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)`,
		ph[0],
		ph[1],
		ph[2],
		ph[3],
		ph[4],
		ph[5],
		ph[6],
		ph[7],
		ph[8],
	)

	return sharedInsertEvents(
		ctx, tx, aggregateType, aggregateID, events,
		insertSQL, s.dialect.FormatTime,
	)
}
