package storage

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Dialect abstracts SQL differences between database backends (PostgreSQL, SQLite).
// Each store method delegates placeholder formatting and time handling to a Dialect,
// eliminating the duplicated PostgreSQL/SQLite store pairs.
type Dialect interface {
	// Placeholder returns the positional placeholder for the given 1-based index.
	// PostgreSQL: "$1", "$2", etc.
	// SQLite: "?" for all positions.
	Placeholder(index int) string

	// FormatTime converts a time.Time to the driver-compatible representation.
	// PostgreSQL: time.Time directly.
	// SQLite: RFC3339Nano string.
	FormatTime(t time.Time) any

	// ScanTimeDest returns a destination for scanning a timestamp column.
	// PostgreSQL: *time.Time.
	// SQLite: *string (parsed after scan).
	ScanTimeDest() any

	// ParseTime converts the scanned value back to time.Time.
	// PostgreSQL: identity (already time.Time).
	// SQLite: parse the string.
	ParseTime(src any) (time.Time, error)
}

// PostgresDialect is the Dialect for PostgreSQL databases.
type PostgresDialect struct{}

func (PostgresDialect) Placeholder(index int) string {
	return "$" + strconv.Itoa(index)
}

func (PostgresDialect) FormatTime(t time.Time) any { return t }

func (PostgresDialect) ScanTimeDest() any {
	return new(time.Time)
}

func (PostgresDialect) ParseTime(src any) (time.Time, error) {
	tp, ok := src.(*time.Time)
	if !ok {
		return time.Time{}, fmt.Errorf("postgres dialect: expected *time.Time, got %T: %w", src, ErrUnexpectedTimeType) //nolint:err113
	}

	return *tp, nil
}

// SQLiteDialect is the Dialect for SQLite databases.
type SQLiteDialect struct{}

func (SQLiteDialect) Placeholder(_ int) string { return "?" }

func (SQLiteDialect) FormatTime(t time.Time) any {
	return t.Format(time.RFC3339Nano)
}

func (SQLiteDialect) ScanTimeDest() any {
	return new(string)
}

func (SQLiteDialect) ParseTime(src any) (time.Time, error) {
	sp, ok := src.(*string)
	if !ok {
		return time.Time{}, fmt.Errorf("sqlite dialect: expected *string, got %T: %w", src, ErrUnexpectedTimeType) //nolint:err113
	}

	return parseSQLiteTimestamp(*sp)
}

// placeholders returns a comma-separated list of placeholders for the given count.
func placeholders(d Dialect, count, offset int) string {
	parts := make([]string, count)

	for i := range count {
		parts[i] = d.Placeholder(offset + i + 1)
	}

	return strings.Join(parts, ", ")
}
