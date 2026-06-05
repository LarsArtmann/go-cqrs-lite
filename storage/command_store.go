package storage

import (
	"database/sql"
	"sync/atomic"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v2/sql"
)

// SQLCommandStore persists commands in a SQL database.
type SQLCommandStore struct {
	sqlpkg.Base

	ownDB  bool
	closed atomic.Bool
}

// NewSQLCommandStore creates a new PostgreSQL-backed command store.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLCommandStore(db *sql.DB) (*SQLCommandStore, error) {
	return newSQLCommandStoreWithDialect(db, sqlpkg.PostgresDialect{})
}

// NewSQLiteCommandStore creates a new SQLite-backed command store.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLiteCommandStore(db *sql.DB) (*SQLCommandStore, error) {
	return newSQLCommandStoreWithDialect(db, sqlpkg.SQLiteDialect{})
}

// NewSQLCommandStoreWithDialect creates a new SQL-backed command store with a custom dialect.
// This enables consumers to use any SQL backend (MySQL, CockroachDB, etc.) by implementing the Dialect interface.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLCommandStoreWithDialect(db *sql.DB, d sqlpkg.Dialect) (*SQLCommandStore, error) {
	return newSQLCommandStoreWithDialect(db, d)
}

func newSQLCommandStoreWithDialect(db *sql.DB, d sqlpkg.Dialect) (*SQLCommandStore, error) {
	base, err := sqlpkg.NewBase(db, d)
	if err != nil {
		return nil, err
	}

	return &SQLCommandStore{Base: base}, nil
}

// Close closes the store. If WithOwnership was set, also closes the underlying *sql.DB.
func (s *SQLCommandStore) Close() error {
	s.closed.Store(true)

	if s.ownDB {
		return s.DB.Close()
	}

	return nil
}

func (s *SQLCommandStore) checkClosed() error {
	if s.closed.Load() {
		return command.ErrStoreClosed
	}

	return nil
}

var _ command.Store = (*SQLCommandStore)(nil)
