// Package sqlstore provides a SQL-backed [scheduling.TimerStore] for durable
// deadline timers that survive process restarts.
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
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
)

// SQLTimerStore is a SQL-backed [scheduling.TimerStore]. Payloads are
// serialized to JSON and stored in the payload column. The caller owns the
// *sql.DB; [SQLTimerStore.Close] is a no-op.
type SQLTimerStore[P any] struct {
	db      *sql.DB
	dialect Dialect
	q       queries
}

// NewSQLiteStore creates a SQLite-backed timer store and creates the timers
// table if it does not exist. The caller retains ownership of db.
func NewSQLiteStore[P any](ctx context.Context, db *sql.DB) (*SQLTimerStore[P], error) {
	return newStore[P](ctx, db, DialectSQLite)
}

// NewPostgresStore creates a Postgres-backed timer store and creates the
// timers table if it does not exist. The caller retains ownership of db.
func NewPostgresStore[P any](ctx context.Context, db *sql.DB) (*SQLTimerStore[P], error) {
	return newStore[P](ctx, db, DialectPostgres)
}

// NewMySQLStore creates a MySQL-backed timer store and creates the timers
// table if it does not exist. The caller retains ownership of db.
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

// timerEnvelopeVersion is the payload-column format version. v1 wraps the
// payload so [scheduling.Timer.Actor] survives SQL persistence; v0 (legacy)
// rows hold the bare JSON of P and decode with an empty actor.
const timerEnvelopeVersion = 1

// timerEnvelope is the versioned payload-column format (ADR-0044 pattern):
//
//	{"v":1,"actor":"user:01JXYZ...","payload":<P>}
//
// Detection requires BOTH a "v":1 integer AND a "payload" key, so a legacy
// payload whose own JSON happens to carry a "v" field is not misread — only
// a legacy payload that is itself an object with exactly these two keys
// (v=1 + payload) would be, which is outside the contract.
type timerEnvelope[P any] struct {
	Version int    `json:"v"`
	Actor   string `json:"actor,omitzero"`
	Payload P      `json:"payload"`
}

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

	payload, err := json.Marshal(envelope)
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
	rows, err := s.db.QueryContext(ctx, s.q.due, s.formatTime(now))
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"scheduling.sqlstore.due",
			"query due timers",
		)
	}
	defer func() { _ = rows.Close() }()

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

// decodeDueTimer reconstructs one Timer from its stored row. A decode failure
// is Corruption in the stored data, not a reason to drop the whole batch.
func decodeDueTimer[P any](
	rawID string,
	payload []byte,
	fireAt time.Time,
) (scheduling.Timer[P], error) {
	envelope, err := decodeTimerPayload[P](rawID, payload)
	if err != nil {
		return scheduling.Timer[P]{}, err
	}

	timerID, err := scheduling.ParseTimerID(rawID)
	if err != nil {
		return scheduling.Timer[P]{}, errorfamily.WrapCorruption(
			err,
			"scheduling.sqlstore.parse_timer_id",
			"parse timer ID "+rawID,
		)
	}

	actor, err := id.ParseActorID(envelope.Actor)
	if err != nil {
		return scheduling.Timer[P]{}, errorfamily.WrapCorruption(
			err,
			"scheduling.sqlstore.parse_actor",
			"parse actor for timer "+rawID,
		)
	}

	return scheduling.Timer[P]{
		ID:      timerID,
		FireAt:  fireAt,
		Payload: envelope.Payload,
		Actor:   actor,
	}, nil
}

// MarkFired removes a timer after it has been dispatched.
func (s *SQLTimerStore[P]) MarkFired(ctx context.Context, id scheduling.TimerID) error {
	return s.deleteTimer(ctx, id.String(), "mark_fired")
}

// Cancel removes a timer before it fires.
func (s *SQLTimerStore[P]) Cancel(ctx context.Context, id scheduling.TimerID) error {
	return s.deleteTimer(ctx, id.String(), "cancel")
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

// decodeTimerPayload decodes a payload-column blob for the timer with the
// given ID (used for error context). v1 rows carry the full envelope (actor +
// payload); legacy rows hold the bare JSON of P and decode with an empty
// actor. A probe requiring BOTH a "v":1 integer AND a "payload" key
// distinguishes the shapes; non-object legacy payloads (strings, arrays) fail
// the probe and decode directly as P. Decode failures are classified as
// corruption — a row whose payload no longer matches P is a corrupt row.
func decodeTimerPayload[P any](id string, data []byte) (timerEnvelope[P], error) {
	var probe struct {
		Version *int            `json:"v"`
		Payload *jsontext.Value `json:"payload"`
	}

	if err := json.Unmarshal(data, &probe); err == nil &&
		probe.Version != nil && *probe.Version == timerEnvelopeVersion && probe.Payload != nil {
		var env timerEnvelope[P]
		if err := json.Unmarshal(data, &env); err != nil {
			return timerEnvelope[P]{}, errorfamily.WrapCorruption(
				err,
				"scheduling.sqlstore.unmarshal_envelope",
				"unmarshal timer envelope for "+id,
			)
		}

		return env, nil
	}

	var p P

	if err := json.Unmarshal(data, &p); err != nil {
		return timerEnvelope[P]{}, errorfamily.WrapCorruption(
			err,
			"scheduling.sqlstore.unmarshal_legacy_payload",
			"unmarshal legacy timer payload for "+id,
		)
	}

	return timerEnvelope[P]{Version: 0, Actor: "", Payload: p}, nil
}
