package sql

import (
	"fmt"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// Dialect abstracts SQL differences between database backends (PostgreSQL, SQLite,
// MySQL, DuckDB). Each store method delegates placeholder formatting, time handling,
// and upsert clause generation to a Dialect, eliminating the duplicated per-backend
// store pairs.
//
// The concrete dialects live in per-backend files:
//
//	PostgresDialect  — dialect_postgres.go
//	MySQLDialect     — dialect_mysql.go
//	SQLiteDialect    — dialect_sqlite.go
//	DuckDBDialect    — dialect_duckdb.go
type Dialect interface { //nolint:interfacebloat // each method returns a distinct schema DDL
	Placeholder(index int) string
	FormatTime(t time.Time) any
	ScanTimeDest() any
	ParseTime(src any) (time.Time, error)
	EventSchema() string
	CommandSchema() string
	QuerySchema() string
	SnapshotSchema() string
	CheckpointSchema() string
	KVSchema() string
	TimerSchema() string

	// ExcludedRef returns the SQL reference to the "excluded" (would-be-inserted)
	// value of a column in an upsert SET clause.
	//
	// PostgreSQL/SQLite/DuckDB return "excluded.col".
	// MySQL returns "VALUES(col)".
	ExcludedRef(col string) string

	// OnConflictDoNothing generates the clause appended to an INSERT statement
	// for insert-or-ignore-on-conflict semantics (no update on conflict).
	//
	// noOpCol is a guaranteed-present column that MySQL uses for its
	// self-assignment no-op trick (ON DUPLICATE KEY UPDATE col = col).
	// PostgreSQL/SQLite/DuckDB ignore it and emit "ON CONFLICT DO NOTHING".
	OnConflictDoNothing(noOpCol string) string

	// OnConflictDoUpdate generates the clause appended to an INSERT statement
	// for insert-or-update-on-conflict semantics.
	//
	// conflictCols are the unique-constraint target columns (empty for MySQL,
	// which infers the conflict target from unique keys automatically).
	// setExprs are the SET assignments — build each with [Dialect.ExcludedRef]
	// (e.g. "version = excluded.version" or "version = VALUES(version)").
	//
	// PostgreSQL/SQLite/DuckDB: "ON CONFLICT(cols) DO UPDATE SET exprs".
	// MySQL: "ON DUPLICATE KEY UPDATE exprs".
	OnConflictDoUpdate(conflictCols []string, setExprs []string) string

	// QuoteIdentifier quotes a SQL identifier if the dialect requires it.
	// MySQL backtick-quotes reserved-word columns like `key`; all other dialects
	// return the name unchanged.
	QuoteIdentifier(name string) string
}

// parseTimePointer asserts src is a *time.Time, wraps the failure with a
// dialect-specific code/message, and returns the dereferenced value. Shared
// between Postgres and DuckDB (both native-time dialects); SQLite overrides
// ParseTime because it scans timestamp strings instead of *time.Time.
func parseTimePointer(src any, dialect string) (time.Time, error) {
	tp, ok := src.(*time.Time)
	if !ok {
		return time.Time{}, errorfamily.WrapCorruption(
			ErrUnexpectedTimeType,
			"storage.unexpected_time_type",
			fmt.Sprintf("%s dialect: expected *time.Time, got %T", dialect, src),
		)
	}

	return *tp, nil
}

// Placeholders returns a comma-separated list of placeholders for the given count.
func Placeholders(d Dialect, count, offset int) string {
	parts := make([]string, count)

	for i := range count {
		parts[i] = d.Placeholder(offset + i + 1)
	}

	return strings.Join(parts, ", ")
}
