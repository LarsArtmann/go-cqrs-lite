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
		idStr         string
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
		&idStr, &eventType, &aggType, &aggIDStr,
		&version, &schemaVersion, &payload, &metadataJSON,
		timeDest,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"scan event row for %s/%s v%d (schema v%d) event %s (id %s): %w",
			aggIDStr, aggType, version, schemaVersion, eventType, idStr, err,
		)
	}

	occurredAt, err := s.dialect.ParseTime(timeDest)
	if err != nil {
		return nil, fmt.Errorf(
			"parse occurred_at for %s/%s v%d (schema v%d) event %s (id %s): %w",
			aggIDStr, aggType, version, schemaVersion, eventType, idStr, err,
		)
	}

	return reconstructEvent(
		idStr, eventType, aggType, aggIDStr,
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
		`INSERT INTO events (id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at)
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
