package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// outboxEvent represents an outbox entry for JSON serialization.
type outboxEvent struct {
	ID            string           `json:"id"`
	Type          string           `json:"type"`
	AggregateType string           `json:"aggregate_type"`
	AggregateID   string           `json:"aggregate_id"`
	Version       event.Version    `json:"version"`
	SchemaVersion event.SchemaVersion `json:"schema_version,omitempty"`
	Payload       []byte           `json:"payload"`
	Metadata      *event.Metadata  `json:"metadata,omitempty"`
	OccurredAt    time.Time        `json:"occurred_at"`
}

func marshalOutboxEvents(events []event.Event) ([]byte, error) {
	rows := make([]outboxEvent, len(events))

	for i, evt := range events {
		rows[i] = outboxEvent{
			ID:            evt.ID().String(),
			Type:          string(evt.Type()),
			AggregateType: string(evt.AggregateType()),
			AggregateID:   evt.AggregateID().String(),
			Version:       evt.Version(),
			SchemaVersion: evt.SchemaVersion(),
			Payload:       evt.Payload(),
			Metadata:      evt.Metadata(),
			OccurredAt:    evt.OccurredAt(),
		}
	}

	data, err := json.Marshal(rows)
	if err != nil {
		return nil, fmt.Errorf("marshal outbox events: %w", err)
	}

	return data, nil
}

func unmarshalOutboxEvents(data []byte) ([]event.Event, error) {
	var rows []outboxEvent

	err := json.Unmarshal(data, &rows)
	if err != nil {
		return nil, fmt.Errorf("unmarshal outbox events: %w", err)
	}

	events := make([]event.Event, 0, len(rows))

	for _, row := range rows {
		evt, err := reconstructOutboxEvent(row)
		if err != nil {
			return nil, err
		}

		events = append(events, evt)
	}

	return events, nil
}

func reconstructOutboxEvent(row outboxEvent) (event.Event, error) {
	metadataJSON, _ := marshalMetadata(row.Metadata)

	parsedEventID, err := id.ParseEventID(row.ID)
	if err != nil {
		return nil, fmt.Errorf("parse event ID %q: %w", row.ID, err)
	}

	r, err := id.ParseAggregateID(row.AggregateID)
		if err != nil {
			return nil, fmt.Errorf("parse aggregate ID %q: %w", row.AggregateID, err)
		}

		metaOpts, err := unmarshalEventMetadata(metadataJSON, row.Type)
		if err != nil {
			return nil, err
		}

		opts := make([]event.Option, 0, 3+len(metaOpts))
		opts = append(opts, event.WithEventID(parsedEventID), event.WithOccurredAt(row.OccurredAt))

		if row.SchemaVersion.Int() > 0 {
			opts = append(opts, event.WithSchemaVersion(row.SchemaVersion))
		}

		opts = append(opts, metaOpts...)

		return event.NewEvent(
			event.Type(row.Type),
			r,
			event.AggregateType(row.AggregateType),
			row.Version,
			row.Payload,
			opts...,
		)
}

func scanOutboxEntries(rows *sql.Rows) ([]event.OutboxEntry, error) {
	var entries []event.OutboxEntry

	for rows.Next() {
		var (
			idStr       string
			eventsBytes []byte
		)

		err := rows.Scan(&idStr, &eventsBytes)
		if err != nil {
			return nil, fmt.Errorf("scan outbox row: %w", err)
		}

		events, err := unmarshalOutboxEvents(eventsBytes)
		if err != nil {
			return nil, fmt.Errorf("unmarshal outbox entry %s: %w", idStr, err)
		}

		entries = append(entries, event.OutboxEntry{
			ID:     event.OutboxID(idStr),
			Events: events,
		})
	}

	err := rows.Err()
	if err != nil {
		return nil, fmt.Errorf("iterate outbox rows: %w", err)
	}

	return entries, nil
}
