package storage

import (
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/query/v3"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v3/sql"
)

// SQLQueryStore persists queries in a SQL database for audit purposes.
type SQLQueryStore struct {
	*sqlpkg.OwnedDBHandle
}

// NewSQLQueryStore creates a new PostgreSQL-backed query store.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLQueryStore(db *sql.DB) (*SQLQueryStore, error) {
	return newSQLQueryStoreWithDialect(db, sqlpkg.PostgresDialect{})
}

// NewSQLiteQueryStore creates a new SQLite-backed query store.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLiteQueryStore(db *sql.DB) (*SQLQueryStore, error) {
	return newSQLQueryStoreWithDialect(db, sqlpkg.SQLiteDialect{})
}

// NewSQLQueryStoreWithDialect creates a new SQL-backed query store with a custom dialect.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLQueryStoreWithDialect(db *sql.DB, d sqlpkg.Dialect) (*SQLQueryStore, error) {
	return newSQLQueryStoreWithDialect(db, d)
}

func newSQLQueryStoreWithDialect(db *sql.DB, d sqlpkg.Dialect) (*SQLQueryStore, error) {
	handle, err := sqlpkg.NewBorrowedDBHandle(db, d)
	if err != nil {
		return nil, err
	}

	return &SQLQueryStore{OwnedDBHandle: handle}, nil
}

func (s *SQLQueryStore) checkClosed() error {
	return s.CheckClosed(query.ErrStoreClosed)
}

var (
	_ query.QueryStore           = (*SQLQueryStore)(nil)
	_ query.QueryJournal         = (*SQLQueryStore)(nil)
	_ query.SeekableQueryJournal = (*SQLQueryStore)(nil)
)
