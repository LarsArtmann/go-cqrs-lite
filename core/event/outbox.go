package event

import (
	"context"
	"io"
	"time"

	cbid "github.com/larsartmann/go-branded-id"
)

type outboxMarker struct{}

// OutboxID identifies an entry in the outbox.
type OutboxID = cbid.ID[outboxMarker, string]

// NewOutboxID creates a new OutboxID from a string.
func NewOutboxID(s string) OutboxID { return cbid.NewID[outboxMarker, string](s) }

// OutboxEntry represents a batch of events staged for reliable publishing.
type OutboxEntry struct {
	ID        OutboxID
	Events    []Event
	CreatedAt time.Time
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
