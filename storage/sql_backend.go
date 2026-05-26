package storage

import (
	"database/sql"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// SQLBackend provides a unified entry point for SQL-backed CQRS storage.
//
// It wires SQLEventStore and SQLOutbox to share the same *sql.DB, ensuring
// that SaveWithOutbox operates atomically within a single transaction.
//
// Usage:
//
//	backend, err := storage.NewSQLBackend(db)
//	if err != nil { ... }
//
//	store  := backend.TransactionalStore() // atomic save + outbox
//	outbox := backend.Outbox()             // for OutboxPoller
//	go storage.NewOutboxPoller(outbox, bus).Start(ctx)
type SQLBackend struct {
	store  *SQLEventStore
	outbox *SQLOutbox
	tx     *SQLTransactionalStore
}

// NewSQLBackend creates a SQLBackend using PostgreSQL dialect.
// Returns an error if db is nil.
func NewSQLBackend(db *sql.DB) (*SQLBackend, error) {
	return newSQLBackendWithDialect(db, PostgresDialect{})
}

// NewSQLiteBackend creates a SQLBackend using SQLite dialect.
// Returns an error if db is nil.
func NewSQLiteBackend(db *sql.DB) (*SQLBackend, error) {
	return newSQLBackendWithDialect(db, SQLiteDialect{})
}

// NewSQLBackendWithDialect creates a SQLBackend with a custom dialect.
// Returns an error if db is nil.
func NewSQLBackendWithDialect(db *sql.DB, d Dialect) (*SQLBackend, error) {
	return newSQLBackendWithDialect(db, d)
}

func newSQLBackendWithDialect(db *sql.DB, d Dialect) (*SQLBackend, error) {
	store, err := newSQLEventStoreWithDialect(db, d)
	if err != nil {
		return nil, fmt.Errorf("event store: %w", err)
	}

	outbox, err := newSQLOutboxWithDialect(db, d)
	if err != nil {
		return nil, fmt.Errorf("outbox: %w", err)
	}

	tx, err := NewSQLTransactionalStore(store, outbox)
	if err != nil {
		return nil, fmt.Errorf("transactional store: %w", err)
	}

	return &SQLBackend{
		store:  store,
		outbox: outbox,
		tx:     tx,
	}, nil
}

// EventStore returns the underlying SQLEventStore.
func (b *SQLBackend) EventStore() *SQLEventStore {
	return b.store
}

// Outbox returns the underlying SQLOutbox.
func (b *SQLBackend) Outbox() *SQLOutbox {
	return b.outbox
}

// TransactionalStore returns the atomic save+outbox store.
func (b *SQLBackend) TransactionalStore() event.TransactionalStore {
	return b.tx
}
