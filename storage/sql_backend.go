package storage

import (
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/event"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/sql"
)

type SQLBackend struct {
	store *SQLEventStore
}

func NewSQLBackend(db *sql.DB) (*SQLBackend, error) {
	return newSQLBackendWithDialect(db, sqlpkg.PostgresDialect{})
}

func NewSQLiteBackend(db *sql.DB) (*SQLBackend, error) {
	return newSQLBackendWithDialect(db, sqlpkg.SQLiteDialect{})
}

func NewSQLBackendWithDialect(db *sql.DB, d sqlpkg.Dialect) (*SQLBackend, error) {
	return newSQLBackendWithDialect(db, d)
}

func newSQLBackendWithDialect(db *sql.DB, d sqlpkg.Dialect) (*SQLBackend, error) {
	store, err := newSQLEventStoreWithDialect(db, d)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "backend.event_store", "event store")
	}
	return &SQLBackend{store: store}, nil
}

func (b *SQLBackend) EventStore() *SQLEventStore { return b.store }
