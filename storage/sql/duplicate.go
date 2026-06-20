package sql

import (
	"errors"
	"strings"
)

// pgErrorCode is the PostgreSQL SQLSTATE for unique constraint violation.
const pgDuplicateCode = "23505"

// sqliteExtendedCode is the SQLite extended result code for SQLITE_CONSTRAINT_UNIQUE.
const sqliteExtendedCode = 2067

// IsDuplicateKeyError returns true if the error is a unique constraint violation
// from either SQLite ("UNIQUE constraint failed") or PostgreSQL
// ("duplicate key value violates unique constraint").
//
// It first checks for typed error codes (PG SQLSTATE 23505, SQLite extended
// code 2067) via interface assertions, then falls back to string matching
// for drivers that don't expose typed errors.
func IsDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}

	// Typed check: PostgreSQL pgconn.PgError or any error with Code field.
	if hasDuplicateCode(err) {
		return true
	}

	// Typed check: SQLite extended error code (modernc.org/sqlite).
	if hasSQLiteUniqueCode(err) {
		return true
	}

	// String fallback for drivers without typed errors.
	msg := err.Error()

	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "duplicate key value violates unique constraint")
}

// pgCodeError is an interface satisfied by pgconn.PgError and similar types
// that expose a SQLSTATE code string.
type pgCodeError interface {
	Code() string
}

// hasDuplicateCode checks for PostgreSQL SQLSTATE 23505 via typed interface.
func hasDuplicateCode(err error) bool {
	// Try errors.As with the Code() interface.
	var ce pgCodeError
	if errors.As(err, &ce) {
		return ce.Code() == pgDuplicateCode
	}

	// Also check via reflection-free field access for common PG error types.
	type codeGetter interface {
		GetCode() string
	}

	var cg codeGetter
	if errors.As(err, &cg) {
		return cg.GetCode() == pgDuplicateCode
	}

	return false
}

// sqliteCodeError is an interface satisfied by modernc.org/sqlite Error
// types that expose a numeric result code.
type sqliteCodeError interface {
	Code() int
}

// hasSQLiteUniqueCode checks for SQLite SQLITE_CONSTRAINT_UNIQUE (2067).
func hasSQLiteUniqueCode(err error) bool {
	var ce sqliteCodeError
	if errors.As(err, &ce) {
		return ce.Code() == sqliteExtendedCode
	}

	return false
}
