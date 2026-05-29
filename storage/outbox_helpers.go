package storage

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/sql"
)

// outboxEvent represents an outbox entry for JSON serialization.
type outboxEvent struct {
	ID            id.EventID          `json:"id"`
	Type          string              `json:"type"`
	AggregateType string              `json:"aggregate_type"`
	AggregateID   id.AggregateID      `json:"aggregate_id"`
	Version       event.Version       `json:"version"`
	SchemaVersion event.SchemaVersion `json:"schema_version,omitempty"`
	Payload       []byte              `json:"payload"`
	Metadata      *event.Metadata     `json:"metadata,omitempty"`
	OccurredAt    time.Time           `json:"occurred_at"`
	Encoding      string              `json:"encoding,omitempty"`
}

func marshalOutboxEvents(events []event.Event) ([]byte, error) {
	rows := make([]outboxEvent, len(events))

	for i, evt := range events {
		rows[i] = outboxEvent{
			ID:            evt.ID(),
			Type:          string(evt.Type()),
			AggregateType: string(evt.AggregateType()),
			AggregateID:   evt.AggregateID(),
			Version:       evt.Version(),
			SchemaVersion: evt.SchemaVersion(),
			Payload:       evt.Payload(),
			Metadata:      evt.Metadata(),
			OccurredAt:    evt.OccurredAt(),
			Encoding:      string(evt.Encoding()),
		}
	}

	data, err := json.Marshal(rows)
	if err != nil {
		return nil, event.WrapCorruption(err, "storage.marshal_outbox",
			"marshal outbox events")
	}

	return data, nil
}

func unmarshalOutboxEvents(data []byte) ([]event.Event, error) {
	var rows []outboxEvent

	err := json.Unmarshal(data, &rows)
	if err != nil {
		return nil, event.WrapCorruption(err, "storage.unmarshal_outbox",
			"unmarshal outbox events")
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
	metadataJSON, _ := sqlpkg.MarshalMetadata(row.Metadata)

	return sqlpkg.ReconstructEvent(
		row.ID, row.Type, row.AggregateType, row.AggregateID,
		row.Version.Int(), row.SchemaVersion.Int(),
		row.Payload, metadataJSON,
		row.OccurredAt,
		codec.Encoding(row.Encoding),
	)
}

func scanOutboxEntries(rows *sql.Rows, d sqlpkg.Dialect) ([]event.OutboxEntry, error) {
	var entries []event.OutboxEntry

	for rows.Next() {
		var (
			idStr       string
			eventsBytes []byte
			timeDest    = d.ScanTimeDest()
		)

		err := rows.Scan(&idStr, &eventsBytes, timeDest)
		if err != nil {
			return nil, event.WrapInfrastructure(err, "storage.scan_outbox",
				"scan outbox row")
		}

		createdAt, err := d.ParseTime(timeDest)
		if err != nil {
			return nil, event.WrapCorruption(err, "storage.parse_outbox_created_at",
				"parse created_at for outbox entry "+idStr)
		}

		events, err := unmarshalOutboxEvents(eventsBytes)
		if err != nil {
			return nil, event.WrapCorruption(err, "storage.unmarshal_outbox_entry",
				"unmarshal outbox entry "+idStr)
		}

		entries = append(entries, event.OutboxEntry{
			ID:        event.NewOutboxID(idStr),
			Events:    events,
			CreatedAt: createdAt,
		})
	}

	err := rows.Err()
	if err != nil {
		return nil, event.WrapInfrastructure(err, "storage.iterate_outbox",
			"iterate outbox rows")
	}

	return entries, nil
}
