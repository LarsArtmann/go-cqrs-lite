package middleware

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

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
		return nil, fmt.Errorf("sql dead letter store: migrate: %w", err)
	}

	return s, nil
}

func (s *SQLDeadLetterStore) migrate() error {
	ddl := s.schemaSQL()

	_, err := s.db.Exec(ddl)
	if err != nil {
		return fmt.Errorf("create table %s: %w", tableDeadLetters, err)
	}

	return nil
}

func (s *SQLDeadLetterStore) schemaSQL() string {
	if s.dialect == "postgres" {
		return `CREATE TABLE IF NOT EXISTS ` + tableDeadLetters + ` (
    id          SERIAL PRIMARY KEY,
    kind        VARCHAR(50) NOT NULL,
    type        VARCHAR(255) NOT NULL,
    aggregate_id TEXT,
    error_text  TEXT,
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
    attempts    INTEGER NOT NULL DEFAULT 0,
    failed_at   TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_dead_letters_kind ON ` + tableDeadLetters + `(kind);
CREATE INDEX IF NOT EXISTS idx_dead_letters_type ON ` + tableDeadLetters + `(type);`
}

func (s *SQLDeadLetterStore) placeholder(idx int) string {
	if s.dialect == "postgres" {
		return "$" + strconv.Itoa(idx)
	}

	return "?"
}

func (s *SQLDeadLetterStore) formatTime(t time.Time) any {
	if s.dialect == "postgres" {
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

	if entry.Error != nil {
		errText = entry.Error.Error()
	}

	failedAt := entry.FailedAt
	if failedAt.IsZero() {
		failedAt = time.Now()
	}

	p1 := s.placeholder(1)
	p2 := s.placeholder(2)
	p3 := s.placeholder(3)
	p4 := s.placeholder(4)
	p5 := s.placeholder(5)
	p6 := s.placeholder(6)

	query := fmt.Sprintf(
		"INSERT INTO %s (kind, type, aggregate_id, error_text, attempts, failed_at) VALUES (%s, %s, %s, %s, %s, %s)",
		tableDeadLetters,
		p1,
		p2,
		p3,
		p4,
		p5,
		p6,
	)

	_, _ = s.db.ExecContext(ctx, query,
		entry.Kind, entry.Type, aggID, errText, entry.Attempts, s.formatTime(failedAt))
}

// Entries returns all dead-lettered messages, ordered by insertion time.
func (s *SQLDeadLetterStore) Entries(ctx context.Context) ([]DeadLetterEntry, error) {
	query := fmt.Sprintf(
		"SELECT kind, type, aggregate_id, error_text, attempts, failed_at FROM %s ORDER BY id",
		tableDeadLetters,
	)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sql dead letter store: query: %w", err)
	}

	defer func() { _ = rows.Close() }()

	var entries []DeadLetterEntry

	for rows.Next() {
		var (
			kind        string
			typ         string
			aggID       sql.NullString
			errText     sql.NullString
			attempts    int
			failedAtRaw any
		)

		if err := rows.Scan(&kind, &typ, &aggID, &errText, &attempts, &failedAtRaw); err != nil {
			return nil, fmt.Errorf("sql dead letter store: scan: %w", err)
		}

		entry := DeadLetterEntry{
			Kind:     kind,
			Type:     typ,
			Attempts: attempts,
		}

		if aggID.Valid && aggID.String != "" {
			entry.AggregateID = idParseSafe(aggID.String)
		}

		if errText.Valid && errText.String != "" {
			entry.Error = strError(errText.String)
		}

		entry.FailedAt, _ = s.parseTime(failedAtRaw)
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// Count returns the number of dead-lettered messages.
func (s *SQLDeadLetterStore) Count(ctx context.Context) (int, error) {
	var count int

	query := "SELECT COUNT(*) FROM " + tableDeadLetters

	err := s.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("sql dead letter store: count: %w", err)
	}

	return count, nil
}

// Clear removes all dead-lettered messages.
func (s *SQLDeadLetterStore) Clear(ctx context.Context) error {
	query := "DELETE FROM " + tableDeadLetters

	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("sql dead letter store: clear: %w", err)
	}

	return nil
}

func (s *SQLDeadLetterStore) parseTime(src any) (time.Time, error) {
	if s.dialect == "postgres" {
		if t, ok := src.(time.Time); ok {
			return t, nil
		}

		return time.Time{}, fmt.Errorf("expected time.Time, got %T", src)
	}

	str, ok := src.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("expected string, got %T", src)
	}

	return time.Parse(time.RFC3339Nano, str)
}

type strErrorType string

func (e strErrorType) Error() string { return string(e) }

func strError(s string) error { return strErrorType(s) }

func idParseSafe(s string) id.AggregateID {
	aid, err := id.ParseAggregateID(s)
	if err != nil {
		return id.AggregateID{}
	}

	return aid
}
