package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// SQLOutbox persists events for reliable eventual publishing in a SQL database.
type SQLOutbox struct {
	db      *sql.DB
	dialect Dialect
}

// NewSQLOutbox creates a new SQL-backed outbox using PostgreSQL dialect.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLOutbox(db *sql.DB) (*SQLOutbox, error) {
	return newSQLOutboxWithDialect(db, PostgresDialect{})
}

// NewSQLiteOutbox creates a new SQLite-backed outbox.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLiteOutbox(db *sql.DB) (*SQLOutbox, error) {
	return newSQLOutboxWithDialect(db, SQLiteDialect{})
}

func newSQLOutboxWithDialect(db *sql.DB, d Dialect) (*SQLOutbox, error) {
	if db == nil {
		return nil, fmt.Errorf("%w", ErrNilDB)
	}

	return &SQLOutbox{db: db, dialect: d}, nil
}

// Close is a no-op. The *sql.DB is borrowed from the caller, who owns its lifecycle.
func (o *SQLOutbox) Close() error { return nil }

// OutboxSchema returns the SQL DDL for creating the outbox table.
func OutboxSchema() string {
	return `CREATE TABLE IF NOT EXISTS outbox (
    id          TEXT PRIMARY KEY,
    status      TEXT NOT NULL DEFAULT '` + string(OutboxStatusPending) + `'` + `,
    events      JSONB NOT NULL,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_outbox_pending ON outbox(created_at) WHERE status = '` + string(OutboxStatusPending) + `'` + `;`
}

// SQLiteOutboxSchema returns the SQL DDL for creating the outbox table in SQLite.
func SQLiteOutboxSchema() string {
	return `CREATE TABLE IF NOT EXISTS outbox (
    id          TEXT PRIMARY KEY,
    status      TEXT NOT NULL DEFAULT '` + string(OutboxStatusPending) + `'` + `,
    events      TEXT NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_outbox_pending ON outbox(created_at) WHERE status = '` + string(OutboxStatusPending) + `'` + `;`
}

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

	insertSQL := outboxInsertSQL(o.dialect)

	_, err = o.db.ExecContext(
		ctx,
		insertSQL,
		outboxID,
		string(OutboxStatusPending),
		serialized,
		o.dialect.FormatTime(time.Now()),
	)
	if err != nil {
		return fmt.Errorf("insert outbox entry %s: %w", outboxID, err)
	}

	return nil
}

// PollPending returns unacknowledged outbox entries, oldest first.
func (o *SQLOutbox) PollPending(ctx context.Context, limit int) ([]event.OutboxEntry, error) {
	p1, p2 := o.dialect.Placeholder(1), o.dialect.Placeholder(2)

	query := fmt.Sprintf(`SELECT id, events FROM outbox
		WHERE status = %s
		ORDER BY created_at ASC
		LIMIT %s`, p1, p2)

	rows, err := o.db.QueryContext(ctx, query, string(OutboxStatusPending), limit)
	if err != nil {
		return nil, fmt.Errorf("poll pending outbox (limit %d): %w", limit, err)
	}

	defer func() {
		_ = rows.Close()
	}()

	return scanOutboxEntries(rows)
}

// Ack removes outbox entries by their IDs.
// Deletes in batches of maxAckBatchSize to avoid exceeding PostgreSQL's
// parameter limit (65535).
func (o *SQLOutbox) Ack(ctx context.Context, ids []event.OutboxID) error {
	if len(ids) == 0 {
		return nil
	}

	for start := 0; start < len(ids); start += maxAckBatchSize {
		end := min(start+maxAckBatchSize, len(ids))

		batch := ids[start:end]

		err := o.ackBatch(ctx, batch)
		if err != nil {
			return fmt.Errorf("ack outbox entries [%d:%d]: %w", start, end, err)
		}
	}

	return nil
}

// maxAckBatchSize limits the number of IDs in a single DELETE to avoid
// exceeding PostgreSQL's 65535 parameter limit.
const maxAckBatchSize = 500

func (o *SQLOutbox) ackBatch(ctx context.Context, ids []event.OutboxID) error {
	return sharedAckBatch(ctx, o.db, ids, o.dialect)
}

var _ event.Outbox = (*SQLOutbox)(nil)
