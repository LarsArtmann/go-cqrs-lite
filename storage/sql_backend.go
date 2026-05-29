package storage

import (
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/sql"
)

type SQLBackend struct {
	store  *SQLEventStore
	outbox *SQLOutbox
	tx     *SQLTransactionalStore
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
	outbox, err := newSQLOutboxWithDialect(db, d)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "backend.outbox", "outbox")
	}
	tx, err := NewSQLTransactionalStore(store, outbox)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "backend.transactional_store", "transactional store")
	}
	return &SQLBackend{store: store, outbox: outbox, tx: tx}, nil
}

func (b *SQLBackend) EventStore() *SQLEventStore        { return b.store }
func (b *SQLBackend) Outbox() *SQLOutbox                 { return b.outbox }
func (b *SQLBackend) TransactionalSink() event.TransactionalSink { return b.tx }
