package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	json "github.com/go-json-experiment/json"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// OutboxSchema returns the SQL DDL for creating the outbox table.
func OutboxSchema() string {
	return `CREATE TABLE IF NOT EXISTS outbox (
    id          TEXT PRIMARY KEY,
    status      TEXT NOT NULL DEFAULT 'pending',
    events      JSONB NOT NULL,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_outbox_pending ON outbox(created_at) WHERE status = 'pending';`
}

// SQLOutbox persists events for reliable eventual publishing in a SQL database.
type SQLOutbox struct {
	db *sql.DB
}

// NewSQLOutbox creates a new SQL-backed outbox.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLOutbox(db *sql.DB) (*SQLOutbox, error) {
	if db == nil {
		return nil, fmt.Errorf("%w", ErrNilDB)
	}

	return &SQLOutbox{db: db}, nil
}

// Close is a no-op. The *sql.DB is borrowed from the caller, who owns its lifecycle.
func (o *SQLOutbox) Close() error { return nil }

const outboxInsertSQL = `INSERT INTO outbox (id, status, events, created_at) VALUES ($1, 'pending', $2, $3)`

// Append writes events to the outbox in a single transaction.
func (o *SQLOutbox) Append(ctx context.Context, events []event.Event) error {
	if len(events) == 0 {
		return nil
	}

	serialized, err := marshalOutboxEvents(events)
	if err != nil {
		return fmt.Errorf("serialize outbox events: %w", err)
	}

	outboxID := events[0].ID()

	_, err = o.db.ExecContext(ctx, outboxInsertSQL, outboxID, serialized, time.Now())
	if err != nil {
		return fmt.Errorf("insert outbox entry %s: %w", outboxID, err)
	}

	return nil
}

// PollPending returns unacknowledged outbox entries, oldest first.
func (o *SQLOutbox) PollPending(ctx context.Context, limit int) ([]event.OutboxEntry, error) {
	query := `SELECT id, events FROM outbox
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT $1`

	rows, err := o.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("poll pending outbox: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	return scanOutboxEntries(rows)
}

// Ack removes outbox entries by their IDs.
func (o *SQLOutbox) Ack(ctx context.Context, ids []event.OutboxID) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))

	for i, oid := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = string(oid)
	}

	query := fmt.Sprintf("DELETE FROM outbox WHERE id IN (%s)", strings.Join(placeholders, ", "))

	_, err := o.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("ack outbox entries: %w", err)
	}

	return nil
}

var _ event.Outbox = (*SQLOutbox)(nil)

type outboxEvent struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   string          `json:"aggregate_id"`
	Version       int             `json:"version"`
	Payload       []byte          `json:"payload"`
	Metadata      *event.Metadata `json:"metadata,omitempty"`
	OccurredAt    time.Time       `json:"occurred_at"`
}

func marshalOutboxEvents(events []event.Event) ([]byte, error) {
	rows := make([]outboxEvent, len(events))

	for i, evt := range events {
		rows[i] = outboxEvent{
			ID:            evt.ID().String(),
			Type:          string(evt.Type()),
			AggregateType: string(evt.AggregateType()),
			AggregateID:   evt.AggregateID().String(),
			Version:       evt.Version().Int(),
			Payload:       evt.Payload(),
			Metadata:      evt.Metadata(),
			OccurredAt:    evt.OccurredAt(),
		}
	}

	data, err := json.Marshal(rows)
	if err != nil {
		return nil, errors.Wrap(err, "marshal outbox events")
	}

	return data, nil
}

func unmarshalOutboxEvents(data []byte) ([]event.Event, error) {
	var rows []outboxEvent

	if err := json.Unmarshal(data, &rows); err != nil {
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
	aggID, err := id.ParseAggregateID(row.AggregateID)
	if err != nil {
		return nil, fmt.Errorf("parse aggregate ID %q: %w", row.AggregateID, err)
	}

	eventID, err := id.ParseEventID(row.ID)
	if err != nil {
		return nil, fmt.Errorf("parse event ID %q: %w", row.ID, err)
	}

	opts := []event.Option{
		event.WithEventID(eventID),
		event.WithOccurredAt(row.OccurredAt),
	}

	if row.Metadata != nil {
		opts = append(opts, event.WithMetadata(row.Metadata))
	}

	evt, err := event.NewEvent(
		event.Type(row.Type),
		aggID,
		event.AggregateType(row.AggregateType),
		row.Version,
		row.Payload,
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("reconstruct outbox event %s: %w", row.Type, err)
	}

	return evt, nil
}

func scanOutboxEntries(rows *sql.Rows) ([]event.OutboxEntry, error) {
	var entries []event.OutboxEntry

	for rows.Next() {
		var (
			idStr       string
			eventsBytes []byte
		)

		if err := rows.Scan(&idStr, &eventsBytes); err != nil {
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

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate outbox rows: %w", err)
	}

	return entries, nil
}
