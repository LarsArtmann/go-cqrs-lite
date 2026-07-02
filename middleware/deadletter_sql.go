package middleware

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

const (
	dialectPostgres = "postgres"
)

var errUnexpectedTimeTypeDL = errors.New("unexpected time type")

// SQLDeadLetterStore is a persistent dead-letter handler backed by a SQL
// database. It stores [DeadLetterEntry] values in a `dead_letters` table,
// making them queryable across process restarts — the production counterpart
// to [MemoryDeadLetterStore].
//
// The store auto-creates the table on construction. It works with both SQLite
// and PostgreSQL: pass "sqlite" or "postgres" as the dialect argument.
//
// Usage:
//
//	db, _ := sql.Open("sqlite", "dead_letters.db")
//	store, _ := middleware.NewSQLDeadLetterStore(db, "sqlite")
//	config := middleware.RetryConfig{
//	    MaxAttempts:  3,
//	    OnDeadLetter: store.Handle,
//	}
//	// ... run commands/events through the retry middleware ...
//	entries, _ := store.Entries(context.Background()) // inspect dead-lettered messages
type SQLDeadLetterStore struct {
	db      *sql.DB
	dialect string
}

const tableDeadLetters = "dead_letters"

// NewSQLDeadLetterStore creates a SQL-backed dead-letter store and auto-creates
// the dead_letters table. The dialect must be "sqlite" or "postgres".
func NewSQLDeadLetterStore(db *sql.DB, dialect string) (*SQLDeadLetterStore, error) {
	s := &SQLDeadLetterStore{db: db, dialect: dialect}

	if err := s.migrate(); err != nil {
			return nil, event.WrapInfrastructure(err, "deadletter.migrate",
			"migrate dead-letter table")
	}

	return s, nil
}

func (s *SQLDeadLetterStore) migrate() error {
	ctx := context.Background()

	if _, err := s.db.ExecContext(ctx, s.schemaSQL()); err != nil {
		return event.WrapInfrastructure(err, "deadletter.create_table",
			"create dead_letters table")
	}

	s.migrateColumns(ctx)

	return nil
}

// migrateColumns adds error_code and error_family to tables created by older
// versions that lacked those columns. Best-effort: ignores errors (column
// already exists, or the database is too old for ALTER TABLE ADD COLUMN).
func (s *SQLDeadLetterStore) migrateColumns(ctx context.Context) {
	for _, col := range []string{"error_code", "error_family"} {
		stmt := fmt.Sprintf(
			"ALTER TABLE %s ADD COLUMN %s TEXT",
			tableDeadLetters,
			col,
		)
		if s.dialect == dialectPostgres {
			stmt += " IF NOT EXISTS"
		}

		_, _ = s.db.ExecContext(ctx, stmt)
	}
}

func (s *SQLDeadLetterStore) schemaSQL() string {
	if s.dialect == dialectPostgres {
		return `CREATE TABLE IF NOT EXISTS ` + tableDeadLetters + ` (
    id          SERIAL PRIMARY KEY,
    kind        VARCHAR(50) NOT NULL,
    type        VARCHAR(255) NOT NULL,
    aggregate_id TEXT,
    error_text  TEXT,
    error_code  TEXT,
    error_family TEXT,
    attempts    INTEGER NOT NULL DEFAULT 0,
    failed_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dead_letters_kind ON ` + tableDeadLetters + `(kind);
CREATE INDEX IF NOT EXISTS idx_dead_letters_type ON ` + tableDeadLetters + `(type);`
	}

	return `CREATE TABLE IF NOT EXISTS ` + tableDeadLetters + ` (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    kind        TEXT NOT NULL,
    type        TEXT NOT NULL,
    aggregate_id TEXT,
    error_text  TEXT,
    error_code  TEXT,
    error_family TEXT,
    attempts    INTEGER NOT NULL DEFAULT 0,
    failed_at   TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_dead_letters_kind ON ` + tableDeadLetters + `(kind);
CREATE INDEX IF NOT EXISTS idx_dead_letters_type ON ` + tableDeadLetters + `(type);`
}

func (s *SQLDeadLetterStore) placeholder(idx int) string {
	if s.dialect == dialectPostgres {
		return "$" + strconv.Itoa(idx)
	}

	return "?"
}

func (s *SQLDeadLetterStore) placeholders(n int) string {
	parts := make([]string, n)
	for i := range n {
		parts[i] = s.placeholder(i + 1)
	}

	return strings.Join(parts, ", ")
}

func (s *SQLDeadLetterStore) formatTime(t time.Time) any {
	if s.dialect == dialectPostgres {
		return t
	}

	return t.Format(time.RFC3339Nano)
}

// Handle stores a dead-letter entry. Implements DeadLetterHandler.
func (s *SQLDeadLetterStore) Handle(ctx context.Context, entry DeadLetterEntry) {
	aggID := ""

	if !entry.AggregateID.IsZero() {
		aggID = entry.AggregateID.String()
	}

	errText := ""
	code := entry.ErrorCode
	family := entry.ErrorFamily

	if entry.Error != nil {
		errText = entry.Error.Error()
		if code == "" {
			code, family = classifyDeadLetterError(entry.Error)
		}
	}

	failedAt := entry.FailedAt
	if failedAt.IsZero() {
		failedAt = time.Now()
	}

	query := "INSERT INTO " + tableDeadLetters + //nolint:gosec // G202: constant concat
		" (kind, type, aggregate_id, error_text, error_code, error_family, attempts, failed_at) VALUES (" +
		s.placeholders(8) + ")" //nolint:mnd // 8 columns

	_, _ = s.db.ExecContext(
		ctx,
		query,
		entry.Kind,
		entry.Type,
		aggID,
		errText,
		code,
		family,
		entry.Attempts,
		s.formatTime(failedAt),
	)
}

// Entries returns all dead-lettered messages, ordered by insertion time.
func (s *SQLDeadLetterStore) Entries(ctx context.Context) ([]DeadLetterEntry, error) {
	query := "SELECT kind, type, aggregate_id, error_text, error_code, error_family, attempts, failed_at FROM " +
		tableDeadLetters + " ORDER BY id"

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, event.WrapTransient(err, "deadletter.query",
			"query dead-letter entries")
	}

	defer func() { _ = rows.Close() }()

	var entries []DeadLetterEntry

	for rows.Next() {
		var (
			kind        string
			typ         string
			aggID       sql.NullString
			errText     sql.NullString
			errCode     sql.NullString
			errFamily   sql.NullString
			attempts    int
			failedAtRaw any
		)

		if err := rows.Scan(
			&kind,
			&typ,
			&aggID,
			&errText,
			&errCode,
			&errFamily,
			&attempts,
			&failedAtRaw,
		); err != nil {
			return nil, event.WrapCorruption(err, "deadletter.scan",
				"scan dead-letter row")
		}

		entry := DeadLetterEntry{ //nolint:exhaustruct // AggregateID/Error/FailedAt set below
			Kind:     kind,
			Type:     typ,
			Attempts: attempts,
		}

		if aggID.Valid && aggID.String != "" {
			entry.AggregateID = idParseSafe(aggID.String)
		}

		if errText.Valid && errText.String != "" {
			entry.Error = deadLetterError(errText.String)
		}

		if errCode.Valid {
			entry.ErrorCode = errCode.String
		}

		if errFamily.Valid {
			entry.ErrorFamily = errFamily.String
		}

		entry.FailedAt, _ = s.parseTime(failedAtRaw)
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, event.WrapTransient(err, "deadletter.rows_err",
			"dead-letter rows iteration")
	}

	return entries, nil
}

// Count returns the number of dead-lettered messages.
func (s *SQLDeadLetterStore) Count(ctx context.Context) (int, error) {
	var count int

	query := "SELECT COUNT(*) FROM " + tableDeadLetters

	err := s.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, event.WrapTransient(err, "deadletter.count",
			"count dead-letter entries")
	}

	return count, nil
}

// Clear removes all dead-lettered messages.
func (s *SQLDeadLetterStore) Clear(ctx context.Context) error {
	query := "DELETE FROM " + tableDeadLetters

	if _, err := s.db.ExecContext(ctx, query); err != nil {
		return event.WrapInfrastructure(err, "deadletter.clear",
			"clear dead-letter table")
	}

	return nil
}

func (s *SQLDeadLetterStore) parseTime(src any) (time.Time, error) {
	if s.dialect == dialectPostgres {
		if t, ok := src.(time.Time); ok {
			return t, nil
		}

		return time.Time{}, event.WrapCorruption(errUnexpectedTimeTypeDL,
			"middleware.deadletter_sql.unexpected_time_type",
			fmt.Sprintf("expected time.Time, got %T", src))
	}

	str, ok := src.(string)
	if !ok {
		return time.Time{}, event.WrapCorruption(errUnexpectedTimeTypeDL,
			"middleware.deadletter_sql.unexpected_string_type",
			fmt.Sprintf("expected string, got %T", src))
	}

	t, err := time.Parse(time.RFC3339Nano, str)
	if err != nil {
		return time.Time{}, event.WrapCorruption(err,
			"middleware.deadletter_sql.parse_time", "parse sqlite time")
	}

	return t, nil
}

type storeError string

func (e storeError) Error() string { return string(e) }

func deadLetterError(s string) error { return storeError(s) }

// classifyDeadLetterError extracts the machine-readable code and lowercase
// family name from err using the CQRS taxonomy. Used when storing dead-letter
// entries so the classification survives the SQL round-trip.
func classifyDeadLetterError(err error) (string, string) {
	family := familyToWire(event.Classify(err))

	code := ""

	if ce, ok := errors.AsType[*event.Error](err); ok {
		code = ce.Code()
	}

	return code, family
}

// familyToWire maps a taxonomy family to its lowercase wire name.
func familyToWire(f event.Family) string {
	switch f {
	case event.Rejection:
		return "rejection"
	case event.Conflict:
		return "conflict"
	case event.Transient:
		return "transient"
	case event.Corruption:
		return "corruption"
	case event.Infrastructure:
		return "infrastructure"
	default:
		return ""
	}
}

func idParseSafe(s string) id.AggregateID {
	aid, err := id.ParseAggregateID(s)
	if err != nil {
		return id.AggregateID{}
	}

	return aid
}
