package storage

import (
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v2/sql"
)

// SQLCommandStore persists commands in a SQL database.
type SQLCommandStore struct {
	*sqlpkg.OwnedDBHandle
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
	handle, err := sqlpkg.NewBorrowedDBHandle(db, d)
	if err != nil {
		return nil, err
	}

	return &SQLCommandStore{OwnedDBHandle: handle}, nil
}

func (s *SQLCommandStore) checkClosed() error {
	return s.CheckClosed(command.ErrStoreClosed)
}

var (
	_ command.Store                  = (*SQLCommandStore)(nil)
	_ command.CommandJournal         = (*SQLCommandStore)(nil)
	_ command.SeekableCommandJournal = (*SQLCommandStore)(nil)
)
