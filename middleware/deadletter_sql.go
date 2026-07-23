package middleware

import (
	"context"
	"database/sql"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

const (
	dialectPostgres = "postgres"
)

var ErrUnexpectedTimeType = errorfamily.NewCorruption(
	"middleware.deadletter.unexpected_time_type",
	"unexpected time type",
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
		return nil, errorfamily.WrapInfrastructure(err, "deadletter.migrate",
			"migrate dead-letter table")
	}

	return s, nil
}

func (s *SQLDeadLetterStore) migrate() error {
	ctx := context.Background()

	if _, err := s.db.ExecContext(ctx, s.schemaSQL()); err != nil {
		return errorfamily.WrapInfrastructure(err, "deadletter.create_table",
			"create dead_letters table")
	}

	s.migrateColumns(ctx)

	return nil
}

// Handle stores a dead-letter entry. Implements DeadLetterHandler.
func (s *SQLDeadLetterStore) Handle(ctx context.Context, entry DeadLetterEntry) {
	aggID := ""

	if !entry.StreamID.IsZero() {
		aggID = entry.StreamID.String()
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
		return nil, errorfamily.WrapTransient(err, "deadletter.query",
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
			return nil, errorfamily.WrapCorruption(err, "deadletter.scan",
				"scan dead-letter row")
		}

		entry := DeadLetterEntry{ //nolint:exhaustruct // StreamID/Error/FailedAt set below
			Kind:     kind,
			Type:     typ,
			Attempts: attempts,
		}

		if aggID.Valid && aggID.String != "" {
			entry.StreamID = idParseSafe(aggID.String)
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
		return nil, errorfamily.WrapTransient(err, "deadletter.rows_err",
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
		return 0, errorfamily.WrapTransient(err, "deadletter.count",
			"count dead-letter entries")
	}

	return count, nil
}

// Clear removes all dead-lettered messages.
func (s *SQLDeadLetterStore) Clear(ctx context.Context) error {
	query := "DELETE FROM " + tableDeadLetters

	if _, err := s.db.ExecContext(ctx, query); err != nil {
		return errorfamily.WrapInfrastructure(err, "deadletter.clear",
			"clear dead-letter table")
	}

	return nil
}
