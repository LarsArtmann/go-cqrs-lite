package sql

import "strings"

// IsDuplicateKeyError returns true if the error is a unique constraint violation
// from either SQLite ("UNIQUE constraint failed") or PostgreSQL
// ("duplicate key value violates unique constraint").
func IsDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()

	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "duplicate key value violates unique constraint")
}
