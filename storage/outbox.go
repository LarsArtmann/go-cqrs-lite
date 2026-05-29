package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
)

// SQLOutbox persists events for reliable eventual publishing in a SQL database.
type SQLOutbox struct {
	sqlBase
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

// NewSQLOutboxWithDialect creates a new SQL-backed outbox with a custom dialect.
// This enables consumers to use any SQL backend by implementing the Dialect interface.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLOutboxWithDialect(db *sql.DB, d Dialect) (*SQLOutbox, error) {
	return newSQLOutboxWithDialect(db, d)
}

func newSQLOutboxWithDialect(db *sql.DB, d Dialect) (*SQLOutbox, error) {
	base, err := newSQLBase(db, d)
	if err != nil {
		return nil, err
	}

	return &SQLOutbox{sqlBase: base}, nil
}

// OutboxSchema returns the SQL DDL for creating the outbox table.
func OutboxSchema() string { return PostgresDialect{}.OutboxSchema() }

// SQLiteOutboxSchema returns the SQL DDL for creating the outbox table (SQLite variant).
func SQLiteOutboxSchema() string { return SQLiteDialect{}.OutboxSchema() }

// Append writes events to the outbox in a single transaction.
func (o *SQLOutbox) Append(ctx context.Context, events []event.Event) error {
	if err := ctx.Err(); err != nil {
		return event.WrapInfrastructure(err, "storage.outbox_context_cancelled",
			"context cancelled")
	}

	if len(events) == 0 {
		return nil
	}

	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "outbox.append",
		trace.SpanKindClient,
		trace.WithAttributes(
			attribute.Int(cqrsotel.AttrEventCount, len(events)),
		),
	)
	defer span.End()

	serialized, err := marshalOutboxEvents(events)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return event.WrapCorruption(err, "storage.serialize_outbox",
			"serialize outbox events")
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
		cqrsotel.RecordError(span, err)

		return event.WrapInfrastructure(err, "storage.insert_outbox",
			"insert outbox entry "+outboxID.String())
	}

	return nil
}

// PollPending returns unacknowledged outbox entries, oldest first.
func (o *SQLOutbox) PollPending(ctx context.Context, limit int) ([]event.OutboxEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, event.WrapInfrastructure(err, "storage.outbox_context_cancelled",
			"context cancelled")
	}

	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "outbox.poll",
		trace.SpanKindClient,
		trace.WithAttributes(
			attribute.Int("cqrs.outbox.limit", limit),
		),
	)
	defer span.End()

	p1, p2 := o.dialect.Placeholder(1), o.dialect.Placeholder(2)

	query := fmt.Sprintf(`SELECT id, events, created_at FROM `+tableOutbox+`
		WHERE status = %s
		ORDER BY created_at ASC
		LIMIT %s`, p1, p2)

	rows, err := o.db.QueryContext(ctx, query, string(OutboxStatusPending), limit)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, event.WrapInfrastructure(err, "storage.poll_pending_outbox",
			fmt.Sprintf("poll pending outbox (limit %d)", limit))
	}

	defer func() {
		_ = rows.Close()
	}()

	entries, err := scanOutboxEntries(rows, o.dialect)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, fmt.Errorf("scan outbox entries (limit=%d): %w", limit, err)
	}

	return entries, nil
}

// Ack removes outbox entries by their IDs.
// Deletes in batches of maxAckBatchSize to avoid exceeding PostgreSQL's
// parameter limit (65535).
func (o *SQLOutbox) Ack(ctx context.Context, ids []event.OutboxID) error {
	if err := ctx.Err(); err != nil {
		return event.WrapInfrastructure(err, "storage.outbox_context_cancelled",
			"context cancelled")
	}

	if len(ids) == 0 {
		return nil
	}

	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "outbox.ack",
		trace.SpanKindClient,
		trace.WithAttributes(
			attribute.Int(cqrsotel.AttrOutboxEntryCount, len(ids)),
		),
	)
	defer span.End()

	for start := 0; start < len(ids); start += maxAckBatchSize {
		end := min(start+maxAckBatchSize, len(ids))

		batch := ids[start:end]

		err := o.ackBatch(ctx, batch)
		if err != nil {
			cqrsotel.RecordError(span, err)

			return event.WrapInfrastructure(err, "storage.ack_outbox_batch",
				fmt.Sprintf("ack outbox entries [%d:%d]", start, end))
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
