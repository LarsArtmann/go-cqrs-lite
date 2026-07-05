package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
	"github.com/larsartmann/go-cqrs-lite/scheduling/v3"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v3/sql"
)

// SQLTimerStore is a persistent [scheduling.TimerStore] backed by a SQL database.
// Payloads are JSON-encoded into a BLOB column. The store is dialect-agnostic:
// pass [sqlpkg.PostgresDialect] or [sqlpkg.SQLiteDialect] (or any custom
// [sqlpkg.Dialect]) at construction.
//
// Use NewSQLTimerStore / NewSQLiteTimerStore for the common cases, or
// NewSQLTimerStoreWithDialect to supply a custom dialect. The timers table is
// created automatically by SQLiteInitSchema / PostgresInitSchema; to add it to
// an existing database, run the DDL from TimerSchema / SQLiteTimerSchema.
type SQLTimerStore[P any] struct {
	sqlpkg.DBHandle
}

// NewSQLTimerStore creates a SQLTimerStore for PostgreSQL.
func NewSQLTimerStore[P any](db *sql.DB) (*SQLTimerStore[P], error) {
	return newSQLTimerStoreWithDialect[P](db, sqlpkg.PostgresDialect{})
}

// NewSQLiteTimerStore creates a SQLTimerStore for SQLite.
func NewSQLiteTimerStore[P any](db *sql.DB) (*SQLTimerStore[P], error) {
	return newSQLTimerStoreWithDialect[P](db, sqlpkg.SQLiteDialect{})
}

// NewSQLTimerStoreWithDialect creates a SQLTimerStore with a custom Dialect.
func NewSQLTimerStoreWithDialect[P any](
	db *sql.DB,
	d sqlpkg.Dialect,
) (*SQLTimerStore[P], error) {
	return newSQLTimerStoreWithDialect[P](db, d)
}

func newSQLTimerStoreWithDialect[P any](
	db *sql.DB,
	d sqlpkg.Dialect,
) (*SQLTimerStore[P], error) {
	handle, err := sqlpkg.NewDBHandle(db, d)
	if err != nil {
		return nil, err
	}

	return &SQLTimerStore[P]{DBHandle: handle}, nil
}

// TimerSchema returns the PostgreSQL DDL for the timers table.
func TimerSchema() string { return sqlpkg.PostgresDialect{}.TimerSchema() }

// SQLiteTimerSchema returns the SQLite DDL for the timers table.
func SQLiteTimerSchema() string { return sqlpkg.SQLiteDialect{}.TimerSchema() }

// Schedule records a timer. If a timer with the same ID already exists, the
// INSERT is ignored (idempotent scheduling, matching MemoryTimerStore).
func (s *SQLTimerStore[P]) Schedule(ctx context.Context, t scheduling.Timer[P]) error {
	ctx, span := s.startSpan(ctx, "timer.schedule", t.ID)
	defer span.End()

	payload, err := json.Marshal(t.Payload)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return event.WrapCorruption(err, "storage.schedule_timer",
			"marshal timer payload for "+t.ID)
	}

	// ON CONFLICT DO NOTHING makes scheduling idempotent: a retry of the same
	// timer ID (e.g. after a crash) is a no-op rather than a duplicate fire.
	// Both SQLite (>=3.24) and Postgres support this syntax.
	query := fmt.Sprintf(
		`INSERT INTO timers (id, fire_at, payload) VALUES (%s, %s, %s) ON CONFLICT(id) DO NOTHING`,
		s.Dialect.Placeholder(1), s.Dialect.Placeholder(2), s.Dialect.Placeholder(3),
	)

	if _, err := s.DB.ExecContext(
		ctx, query,
		t.ID, s.Dialect.FormatTime(t.FireAt), payload,
	); err != nil {
		cqrsotel.RecordError(span, err)

		return event.WrapInfrastructure(err, "storage.schedule_timer",
			"insert timer "+t.ID)
	}

	return nil
}

// Due returns timers whose FireAt is at or before now, ordered by FireAt
// ascending. This allows the scheduler to drain the backlog deterministically.
func (s *SQLTimerStore[P]) Due(ctx context.Context, now time.Time) ([]scheduling.Timer[P], error) {
	ctx, span := s.startSpan(ctx, "timer.due", "")
	defer span.End()

	query := fmt.Sprintf(
		`SELECT id, fire_at, payload FROM timers WHERE fire_at <= %s ORDER BY fire_at ASC`,
		s.Dialect.Placeholder(1),
	)

	rows, err := s.DB.QueryContext(ctx, query, s.Dialect.FormatTime(now))
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, event.WrapInfrastructure(err, "storage.due_timers",
			"query due timers")
	}
	defer func() { _ = rows.Close() }()

	var due []scheduling.Timer[P]

	for rows.Next() {
		var (
			t        scheduling.Timer[P]
			fireDest = s.Dialect.ScanTimeDest()
			payload  []byte
		)

		if err := rows.Scan(&t.ID, fireDest, &payload); err != nil {
			cqrsotel.RecordError(span, err)

			return nil, event.WrapCorruption(err, "storage.scan_timer",
				"scan due timer row")
		}

		fireAt, err := s.Dialect.ParseTime(fireDest)
		if err != nil {
			cqrsotel.RecordError(span, err)

			return nil, event.WrapCorruption(err, "storage.parse_fire_at",
				"parse fire_at for timer "+t.ID)
		}

		t.FireAt = fireAt

		if err := json.Unmarshal(payload, &t.Payload); err != nil {
			cqrsotel.RecordError(span, err)

			return nil, event.WrapCorruption(err, "storage.unmarshal_timer_payload",
				"unmarshal payload for timer "+t.ID)
		}

		due = append(due, t)
	}

	if err := rows.Err(); err != nil {
		cqrsotel.RecordError(span, err)

		return nil, event.WrapInfrastructure(err, "storage.iterate_timers",
			"iterate due timer rows")
	}

	span.SetAttributes(cqrsotel.AttrInt(cqrsotel.AttrEventCount, len(due)))

	return due, nil
}

// MarkFired removes a timer after it has been dispatched.
func (s *SQLTimerStore[P]) MarkFired(ctx context.Context, id scheduling.TimerID) error {
	ctx, span := s.startSpan(ctx, "timer.mark_fired", id)
	defer span.End()

	query := "DELETE FROM timers WHERE id = " + s.Dialect.Placeholder(1)

	if _, err := s.DB.ExecContext(ctx, query, id); err != nil {
		cqrsotel.RecordError(span, err)

		return event.WrapInfrastructure(err, "storage.mark_timer_fired",
			"delete timer "+id)
	}

	return nil
}

// Cancel removes a timer before it fires.
func (s *SQLTimerStore[P]) Cancel(ctx context.Context, id scheduling.TimerID) error {
	ctx, span := s.startSpan(ctx, "timer.cancel", id)
	defer span.End()

	query := "DELETE FROM timers WHERE id = " + s.Dialect.Placeholder(1)

	if _, err := s.DB.ExecContext(ctx, query, id); err != nil {
		cqrsotel.RecordError(span, err)

		return event.WrapInfrastructure(err, "storage.cancel_timer",
			"delete timer "+id)
	}

	return nil
}

func (s *SQLTimerStore[P]) startSpan(
	ctx context.Context,
	name, timerID string,
) (context.Context, cqrsotel.Span) {
	attrs := []cqrsotel.KeyValue{
		cqrsotel.AttrString("cqrs.timer.operation", name),
	}
	if timerID != "" {
		attrs = append(attrs, cqrsotel.AttrString("cqrs.timer.id", timerID))
	}

	return cqrsotel.StartSpan(ctx, sqlpkg.Tracer(), name, cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(attrs...))
}

var _ scheduling.TimerStore[any] = (*SQLTimerStore[any])(nil)
