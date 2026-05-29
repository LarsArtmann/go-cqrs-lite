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
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/sql"
)

// SQLOutbox persists events for reliable eventual publishing in a SQL database.
type SQLOutbox struct {
	sqlpkg.Base
}

func NewSQLOutbox(db *sql.DB) (*SQLOutbox, error) {
	return newSQLOutboxWithDialect(db, sqlpkg.PostgresDialect{})
}

func NewSQLiteOutbox(db *sql.DB) (*SQLOutbox, error) {
	return newSQLOutboxWithDialect(db, sqlpkg.SQLiteDialect{})
}

func NewSQLOutboxWithDialect(db *sql.DB, d sqlpkg.Dialect) (*SQLOutbox, error) {
	return newSQLOutboxWithDialect(db, d)
}

func newSQLOutboxWithDialect(db *sql.DB, d sqlpkg.Dialect) (*SQLOutbox, error) {
	base, err := sqlpkg.NewBase(db, d)
	if err != nil {
		return nil, err
	}

	return &SQLOutbox{Base: base}, nil
}

func OutboxSchema() string  { return sqlpkg.PostgresDialect{}.OutboxSchema() }
func SQLiteOutboxSchema() string { return sqlpkg.SQLiteDialect{}.OutboxSchema() }

func (o *SQLOutbox) Append(ctx context.Context, events []event.Event) error {
	if err := ctx.Err(); err != nil {
		return event.WrapInfrastructure(err, "storage.outbox_context_cancelled",
			"context cancelled")
	}

	if len(events) == 0 {
		return nil
	}

	ctx, span := cqrsotel.StartSpan(
		ctx, sqlpkg.Tracer(), "outbox.append",
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

	insertSQL := sqlpkg.OutboxInsertSQL(o.Dialect)

	_, err = o.DB.ExecContext(
		ctx,
		insertSQL,
		outboxID,
		string(sqlpkg.OutboxStatusPending),
		serialized,
		o.Dialect.FormatTime(time.Now()),
	)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return event.WrapInfrastructure(err, "storage.insert_outbox",
			"insert outbox entry "+outboxID.String())
	}

	return nil
}

func (o *SQLOutbox) PollPending(ctx context.Context, limit int) ([]event.OutboxEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, event.WrapInfrastructure(err, "storage.outbox_context_cancelled",
			"context cancelled")
	}

	ctx, span := cqrsotel.StartSpan(
		ctx, sqlpkg.Tracer(), "outbox.poll",
		trace.SpanKindClient,
		trace.WithAttributes(
			attribute.Int("cqrs.outbox.limit", limit),
		),
	)
	defer span.End()

	p1, p2 := o.Dialect.Placeholder(1), o.Dialect.Placeholder(2)

	query := fmt.Sprintf(`SELECT id, events, created_at FROM `+sqlpkg.TableOutbox+`
		WHERE status = %s
		ORDER BY created_at ASC
		LIMIT %s`, p1, p2)

	rows, err := queryContextWithError(ctx, span, o.DB, query,
		"storage.poll_pending_outbox",
		fmt.Sprintf("poll pending outbox (limit %d)", limit),
		string(sqlpkg.OutboxStatusPending), limit)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = rows.Close()
	}()

	entries, err := scanOutboxEntries(rows, o.Dialect)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, fmt.Errorf("scan outbox entries (limit=%d): %w", limit, err)
	}

	return entries, nil
}

const maxAckBatchSize = 500

func (o *SQLOutbox) Ack(ctx context.Context, ids []event.OutboxID) error {
	if err := ctx.Err(); err != nil {
		return event.WrapInfrastructure(err, "storage.outbox_context_cancelled",
			"context cancelled")
	}

	if len(ids) == 0 {
		return nil
	}

	ctx, span := cqrsotel.StartSpan(
		ctx, sqlpkg.Tracer(), "outbox.ack",
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

func (o *SQLOutbox) ackBatch(ctx context.Context, ids []event.OutboxID) error {
	return sqlpkg.SharedAckBatch(ctx, o.DB, ids, o.Dialect)
}

var _ event.Outbox = (*SQLOutbox)(nil)
