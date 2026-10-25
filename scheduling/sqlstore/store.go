// Package sqlstore provides a SQL-backed [scheduling.TimerStore] for durable
// deadline timers that survive process restarts.
//
// Deprecated: this module rides satellite-owned SQL and is retired by the
// all-durable-I/O-through-metaengine rule (ADR-0142 amendment, 2026-10-24):
// use scheduling/engine.NewTimerStore over any metaengine engine implementing
// metaengine.DueClaimer — timers become claims on the ONE claim stack
// (sqlite/postgres/mysql natively via claimkit; KV engines degraded).
// Nothing is removed in v4; the module is deleted at v5.0.0.
//
// Unlike [scheduling.MemoryTimerStore] (in-memory, lost on restart), a SQL
// store persists timers to disk. This is essential for production sagas like
// "cancel order after 30 min unpaid" — if the process crashes before the timer
// fires, the timer must still be present on restart.
//
// The caller owns the *sql.DB — [SQLTimerStore.Close] is a no-op. The timers
// table is created automatically by the constructors (CREATE TABLE IF NOT
// EXISTS), matching the schema embedded in the storage module's migrations so
// consumers who use both see no conflicts.
package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	codec "github.com/larsartmann/go-codec"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
)

// SQLTimerStore is a SQL-backed [scheduling.TimerStore]. Payloads are
// serialized to JSON and stored in the payload column. The caller owns the
// *sql.DB; [SQLTimerStore.Close] is a no-op.
//
// Deprecated: use scheduling/engine.NewTimerStore over a metaengine.DueClaimer
// engine (ADR-0142 I/O rule); removed at v5.0.0.
type SQLTimerStore[P any] struct {
	db      *sql.DB
	dialect Dialect
	q       queries
}

// NewSQLiteStore creates a SQLite-backed timer store and creates the timers
// table if it does not exist. The caller retains ownership of db.
//
// Deprecated: use scheduling/engine.NewTimerStore over a metaengine.DueClaimer
// engine (ADR-0142 I/O rule); removed at v5.0.0.
func NewSQLiteStore[P any](ctx context.Context, db *sql.DB) (*SQLTimerStore[P], error) {
	return newStore[P](ctx, db, DialectSQLite)
}

// NewPostgresStore creates a Postgres-backed timer store and creates the
// timers table if it does not exist. The caller retains ownership of db.
//
// Deprecated: use scheduling/engine.NewTimerStore over a metaengine.DueClaimer
// engine (ADR-0142 I/O rule); removed at v5.0.0.
func NewPostgresStore[P any](ctx context.Context, db *sql.DB) (*SQLTimerStore[P], error) {
	return newStore[P](ctx, db, DialectPostgres)
}

// NewMySQLStore creates a MySQL-backed timer store and creates the timers
// table if it does not exist. The caller retains ownership of db.
//
// Deprecated: use scheduling/engine.NewTimerStore over a metaengine.DueClaimer
// engine (ADR-0142 I/O rule); removed at v5.0.0.
func NewMySQLStore[P any](ctx context.Context, db *sql.DB) (*SQLTimerStore[P], error) {
	return newStore[P](ctx, db, DialectMySQL)
}

func newStore[P any](ctx context.Context, db *sql.DB, d Dialect) (*SQLTimerStore[P], error) {
	var q queries

	switch d {
	case DialectSQLite:
		q = sqliteQueries()
	case DialectPostgres:
		q = postgresQueries()
	case DialectMySQL:
		q = mysqlQueries()
	case DialectDuckDB:
		// No timer-store query set exists for DuckDB (claim-core-only).
		fallthrough
	default:
		return nil, fmt.Errorf("%w: %d", ErrUnknownDialect, d)
	}

	if _, err := db.ExecContext(ctx, q.ddl); err != nil {
		return nil, fmt.Errorf("sqlstore: create timers table: %w", err)
	}

	return &SQLTimerStore[P]{db: db, dialect: d, q: q}, nil
}

// Close is a no-op; the caller owns the *sql.DB.
func (s *SQLTimerStore[P]) Close() error { return nil }

// Schedule records a timer. If a timer with the same ID already exists, it is
// a no-op (idempotent scheduling).
func (s *SQLTimerStore[P]) Schedule(ctx context.Context, t scheduling.Timer[P]) error {
	// The envelope keeps the actor in its self-describing wire form
	// ("kind:raw", "" when unset) — exactly the shape earlier versions
	// persisted, so existing rows stay readable without migration.
	envelope := timerEnvelope[P]{
		Version: timerEnvelopeVersion,
		Actor:   t.Actor.PrefixedString(),
		Payload: t.Payload,
	}

	// gcdc envelope (ADR-0044): the timer envelope rides the shared
	// codec-stamped wrapper — one envelope system with the blind stores.
	// Legacy rows (v1 hand envelope, v0 bare payload) decode through
	// UnwrapDecode's fallback unchanged.
	payload, err := codec.WrapEncode(envelope, codec.JSONCodec{})
	if err != nil {
		return errorfamily.WrapCorruption(
			err,
			"scheduling.sqlstore.marshal_payload",
			"marshal timer payload to JSON",
		)
	}

	if _, err := s.db.ExecContext(
		ctx,
		s.q.schedule,
		t.ID,
		s.formatTime(t.FireAt),
		payload,
	); err != nil {
		return errorfamily.WrapInfrastructure(
			err,
			"scheduling.sqlstore.schedule",
			"schedule timer "+t.ID.String(),
		)
	}

	return nil
}

// Due returns timers whose FireAt is at or before now, ordered by FireAt
// ascending.
// A row whose payload, timer ID, or actor cannot be decoded (stored-data
// corruption) is skipped, not fatal: the decodable timers are returned
// alongside a joined Corruption error describing every skipped row, so one
// rotten row cannot block dispatch of every other due timer forever. The
// corrupt timers stay in the table and are re-reported each poll until an
// operator removes them.
func (s *SQLTimerStore[P]) Due(ctx context.Context, now time.Time) ([]scheduling.Timer[P], error) {
	rows, err := s.db.QueryContext( //nolint:sqlclosecheck // closed below via record.DeferClose (ADR-0144 idiom; linter cannot see through the helper)
		ctx,
		s.q.due,
		s.formatTime(now),
	)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"scheduling.sqlstore.due",
			"query due timers",
		)
	}
	defer record.DeferClose(rows)

	var timers []scheduling.Timer[P]

	var corrupt []error

	for rows.Next() {
		var rawID string

		var payload []byte

		dest := s.scanTimeDest()

		if err := rows.Scan(&rawID, dest, &payload); err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"scheduling.sqlstore.scan",
				"scan due timer row",
			)
		}

		fireAt, err := s.parseTime(dest)
		if err != nil {
			return nil, err
		}

		timer, err := decodeDueTimer[P](rawID, payload, fireAt)
		if err != nil {
			corrupt = append(corrupt, err)

			continue
		}

		timers = append(timers, timer)
	}

	if err := rows.Err(); err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"scheduling.sqlstore.due_iter",
			"iterate due timers",
		)
	}

	return timers, errors.Join(corrupt...)
}

// MarkFired removes a timer after it has been dispatched.
func (s *SQLTimerStore[P]) MarkFired(ctx context.Context, id scheduling.TimerID) error {
	return s.deleteTimer(ctx, id.Get(), "mark_fired")
}

// Cancel removes a timer before it fires.
func (s *SQLTimerStore[P]) Cancel(ctx context.Context, id scheduling.TimerID) error {
	return s.deleteTimer(ctx, id.Get(), "cancel")
}

func (s *SQLTimerStore[P]) deleteTimer(ctx context.Context, id, op string) error {
	if _, err := s.db.ExecContext(ctx, s.q.deleteByID, id); err != nil {
		return errorfamily.WrapInfrastructure(
			err,
			"scheduling.sqlstore."+op,
			op+" timer "+id,
		)
	}

	return nil
}

func (s *SQLTimerStore[P]) formatTime(t time.Time) any {
	if s.dialect == DialectSQLite {
		return t.UTC().Format(sqliteTimeFormat)
	}

	return t
}

func (s *SQLTimerStore[P]) scanTimeDest() any {
	if s.dialect == DialectSQLite {
		return new(string)
	}

	return new(time.Time)
}

func (s *SQLTimerStore[P]) parseTime(src any) (time.Time, error) {
	switch v := src.(type) {
	case *time.Time:
		return *v, nil
	case *string:
		t, err := time.Parse(sqliteTimeFormat, *v)
		if err != nil {
			return time.Time{}, errorfamily.WrapCorruption(
				err,
				"scheduling.sqlstore.parse_time",
				"parse SQLite timestamp "+*v,
			)
		}

		return t, nil
	default:
		return time.Time{}, errorfamily.NewCorruption(
			"scheduling.sqlstore.unexpected_time_type",
			fmt.Sprintf("sqlstore: unexpected scan destination type %T", src),
		)
	}
}

var _ scheduling.TimerStore[any] = (*SQLTimerStore[any])(nil)
