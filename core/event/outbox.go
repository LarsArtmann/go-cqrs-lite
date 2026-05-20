package event

import (
	"context"
	"io"
)

// OutboxID identifies an entry in the outbox.
type OutboxID string

// String returns the underlying string value.
func (id OutboxID) String() string { return string(id) }

// IsZero returns true if the outbox ID is zero-valued.
func (id OutboxID) IsZero() bool { return id == "" }

// OutboxEntry represents a batch of events staged for reliable publishing.
type OutboxEntry struct {
	ID     OutboxID
	Events []Event
}

// Outbox persists events for reliable eventual publishing.
// All implementations must support lifecycle management via io.Closer.
//
// Implementations MUST guarantee that Append returns successfully only
// when the events are durably stored. For SQL-backed implementations,
// this means Append runs inside the same transaction as the event store
// Save operation.
//
// The typical lifecycle:
//  1. Repository.Save calls Outbox.Append(events) inside the store tx.
//  2. A background OutboxPublisher polls Outbox.PollPending.
//  3. Publisher calls Bus.Publish for each entry.
//  4. On success, publisher calls Outbox.Ack(entry.ID).
type Outbox interface {
	io.Closer

	// Append writes events to the outbox.
	Append(ctx context.Context, events []Event) error

	// PollPending returns unacknowledged outbox entries, oldest first.
	PollPending(ctx context.Context, limit int) ([]OutboxEntry, error)

	// Ack marks entries as successfully published.
	Ack(ctx context.Context, ids []OutboxID) error
}
