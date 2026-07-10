package middleware

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

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

func (s *SQLDeadLetterStore) parseTime(src any) (time.Time, error) {
	if s.dialect == dialectPostgres {
		if t, ok := src.(time.Time); ok {
			return t, nil
		}

		return time.Time{}, errorfamily.WrapCorruption(errUnexpectedTimeTypeDL,
			"middleware.deadletter_sql.unexpected_time_type",
			fmt.Sprintf("expected time.Time, got %T", src))
	}

	str, ok := src.(string)
	if !ok {
		return time.Time{}, errorfamily.WrapCorruption(errUnexpectedTimeTypeDL,
			"middleware.deadletter_sql.unexpected_string_type",
			fmt.Sprintf("expected string, got %T", src))
	}

	t, err := time.Parse(time.RFC3339Nano, str)
	if err != nil {
		return time.Time{}, errorfamily.WrapCorruption(err,
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
	family := familyToWire(errorfamily.Classify(err))

	code := ""

	if ce, ok := errors.AsType[*errorfamily.Error](err); ok {
		code = ce.Code()
	}

	return code, family
}

// familyToWire maps a taxonomy family to its lowercase wire name.
func familyToWire(f errorfamily.Family) string {
	switch f {
	case errorfamily.Rejection:
		return "rejection"
	case errorfamily.Conflict:
		return "conflict"
	case errorfamily.Transient:
		return "transient"
	case errorfamily.Corruption:
		return "corruption"
	case errorfamily.Infrastructure:
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
