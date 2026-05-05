package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// SQLiteOutboxSchema returns the SQL DDL for creating the outbox table in SQLite.
func SQLiteOutboxSchema() string {
	return `CREATE TABLE IF NOT EXISTS outbox (
    id          TEXT PRIMARY KEY,
    status      TEXT NOT NULL DEFAULT 'pending',
    events      TEXT NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_outbox_pending ON outbox(created_at) WHERE status = 'pending';`
}

// SQLiteOutbox persists events for reliable eventual publishing in a SQLite database.
type SQLiteOutbox struct {
	db *sql.DB
}

// NewSQLiteOutbox creates a new SQLite-backed outbox.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLiteOutbox(db *sql.DB) (*SQLiteOutbox, error) {
	if db == nil {
		return nil, fmt.Errorf("%w", ErrNilDB)
	}

	return &SQLiteOutbox{db: db}, nil
}

// Close is a no-op. The *sql.DB is borrowed from the caller, who owns its lifecycle.
func (o *SQLiteOutbox) Close() error { return nil }

const sqliteOutboxInsertSQL = `INSERT INTO outbox (id, status, events, created_at) VALUES (?, 'pending', ?, ?)`

const sqlitePollPendingQuery = `SELECT id, events FROM outbox
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT ?`

// Append writes events to the outbox in a single transaction.
func (o *SQLiteOutbox) Append(ctx context.Context, events []event.Event) error {
	if len(events) == 0 {
		return nil
	}

	serialized, err := marshalOutboxEvents(events)
	if err != nil {
		return fmt.Errorf("serialize outbox events: %w", err)
	}

	outboxID := events[0].ID()

	_, err = o.db.ExecContext(ctx, sqliteOutboxInsertSQL, outboxID, serialized, time.Now())
	if err != nil {
		return fmt.Errorf("insert outbox entry %s: %w", outboxID, err)
	}

	return nil
}

// PollPending returns unacknowledged outbox entries, oldest first.
func (o *SQLiteOutbox) PollPending(ctx context.Context, limit int) ([]event.OutboxEntry, error) {
	rows, err := o.db.QueryContext(ctx, sqlitePollPendingQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("poll pending outbox: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	return scanOutboxEntries(rows)
}

// Ack removes outbox entries by their IDs.
func (o *SQLiteOutbox) Ack(ctx context.Context, ids []event.OutboxID) error {
	if len(ids) == 0 {
		return nil
	}

	for start := 0; start < len(ids); start += maxAckBatchSize {
		end := min(start+maxAckBatchSize, len(ids))

		batch := ids[start:end]

		err := o.sqliteAckBatch(ctx, batch)
		if err != nil {
			return fmt.Errorf("ack outbox entries [%d:%d]: %w", start, end, err)
		}
	}

	return nil
}

func (o *SQLiteOutbox) sqliteAckBatch(ctx context.Context, ids []event.OutboxID) error {
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))

	for i, oid := range ids {
		placeholders[i] = "?" + strconv.Itoa(i+1)
		args[i] = string(oid)
	}

	query := fmt.Sprintf("DELETE FROM outbox WHERE id IN (%s)", strings.Join(placeholders, ", "))

	_, err := o.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("ack outbox entries: %w", err)
	}

	return nil
}

var _ event.Outbox = (*SQLiteOutbox)(nil)
