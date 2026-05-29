package sql

import (
	"database/sql"
)

// Base holds the shared database connection and dialect for all SQL stores.
// Embed this struct in store implementations to avoid duplicating the DB and Dialect fields.
type Base struct {
	DB      *sql.DB
	Dialect Dialect
}

// NewBase creates a Base with the given DB and Dialect, returning ErrNilDB if db is nil.
func NewBase(db *sql.DB, d Dialect) (Base, error) {
	if db == nil {
		return Base{}, ErrNilDB
	}

	return Base{DB: db, Dialect: d}, nil
}

// Close is a no-op for Base (the DB connection lifetime is managed externally).
func (Base) Close() error { return nil }
