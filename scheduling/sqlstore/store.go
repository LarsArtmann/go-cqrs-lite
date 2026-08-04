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
	"encoding/json"
	"fmt"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
)

// Dialect selects SQL syntax for table creation and placeholders.
type Dialect int

const (
	// DialectSQLite uses ? placeholders and stores timestamps as RFC3339 text.
	DialectSQLite Dialect = iota
	// DialectPostgres uses $N placeholders and native TIMESTAMP WITH TIME ZONE.
	DialectPostgres
	// DialectMySQL uses ? placeholders and native DATETIME(3).
	DialectMySQL
)

// sqliteTimeFormat is a fixed-width RFC3339 variant that always emits 9
// fractional digits so lexicographic comparison matches chronological order.
const sqliteTimeFormat = "2006-01-02T15:04:05.000000000Z07:00"

type queries struct {
	ddl        string
	schedule   string
	due        string
	deleteByID string
}

func sqliteQueries() queries {
	return queries{
		ddl: `CREATE TABLE IF NOT EXISTS timers (
	id         TEXT PRIMARY KEY,
	fire_at    TEXT NOT NULL,
	payload    BLOB NOT NULL,
	created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_timers_fire_at ON timers(fire_at);`,
		schedule:   `INSERT INTO timers (id, fire_at, payload) VALUES (?, ?, ?) ON CONFLICT(id) DO NOTHING`,
		due:        `SELECT id, fire_at, payload FROM timers WHERE fire_at <= ? ORDER BY fire_at ASC`,
		deleteByID: `DELETE FROM timers WHERE id = ?`,
	}
}

func postgresQueries() queries {
	return queries{
		ddl: `CREATE TABLE IF NOT EXISTS timers (
	id         TEXT PRIMARY KEY,
	fire_at    TIMESTAMP WITH TIME ZONE NOT NULL,
	payload    BYTEA NOT NULL,
	created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_timers_fire_at ON timers(fire_at);`,
		schedule:   `INSERT INTO timers (id, fire_at, payload) VALUES ($1, $2, $3) ON CONFLICT(id) DO NOTHING`,
		due:        `SELECT id, fire_at, payload FROM timers WHERE fire_at <= $1 ORDER BY fire_at ASC`,
		deleteByID: `DELETE FROM timers WHERE id = $1`,
	}
}

func mysqlQueries() queries {
	return queries{
		ddl: `CREATE TABLE IF NOT EXISTS timers (
	id         VARCHAR(255) PRIMARY KEY,
	fire_at    DATETIME(3) NOT NULL,
	payload    BLOB NOT NULL,
	created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);`,
		schedule: "INSERT INTO timers (id, fire_at, payload) VALUES (?, ?, ?) " +
			"ON DUPLICATE KEY UPDATE id = id",
		due:        `SELECT id, fire_at, payload FROM timers WHERE fire_at <= ? ORDER BY fire_at ASC`,
		deleteByID: `DELETE FROM timers WHERE id = ?`,
	}
}

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
		return nil, fmt.Errorf("sqlstore: unknown dialect %d", d)
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
	payload, err := json.Marshal(t.Payload)
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
			"schedule timer "+t.ID,
		)
	}

	return nil
}

// Due returns timers whose FireAt is at or before now, ordered by FireAt
// ascending.
func (s *SQLTimerStore[P]) Due(ctx context.Context, now time.Time) ([]scheduling.Timer[P], error) {
	rows, err := s.db.QueryContext(ctx, s.q.due, s.formatTime(now))
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"scheduling.sqlstore.due",
			"query due timers",
		)
	}
	defer rows.Close()

	var timers []scheduling.Timer[P]

	for rows.Next() {
		var id string

		var payload []byte

		dest := s.scanTimeDest()

		if err := rows.Scan(&id, dest, &payload); err != nil {
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

		var p P

		if err := json.Unmarshal(payload, &p); err != nil {
			return nil, errorfamily.WrapCorruption(
				err,
				"scheduling.sqlstore.unmarshal_payload",
				"unmarshal timer payload for "+id,
			)
		}

		timers = append(timers, scheduling.Timer[P]{
			ID:      id,
			FireAt:  fireAt,
			Payload: p,
		})
	}

	return timers, rows.Err()
}

// MarkFired removes a timer after it has been dispatched.
func (s *SQLTimerStore[P]) MarkFired(ctx context.Context, id scheduling.TimerID) error {
	return s.deleteTimer(ctx, id, "mark_fired")
}

// Cancel removes a timer before it fires.
func (s *SQLTimerStore[P]) Cancel(ctx context.Context, id scheduling.TimerID) error {
	return s.deleteTimer(ctx, id, "cancel")
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
