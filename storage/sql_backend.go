package storage

import (
	"database/sql"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v2/sql"
)

// SQLBackend is a facade that provides access to all SQL-backed stores
// sharing a single database connection. All store accessors are goroutine-safe.
type SQLBackend struct {
	store *SQLEventStore

	cmdMu      sync.Mutex
	cmdStore   *SQLCommandStore
	queryMu    sync.Mutex
	queryStore *SQLQueryStore
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

// EventStore returns the SQL event store.
func (b *SQLBackend) EventStore() *SQLEventStore { return b.store }

// CommandStore returns the SQL command store, creating it on first call.
// All calls return the same instance. Goroutine-safe.
func (b *SQLBackend) CommandStore() (*SQLCommandStore, error) {
	b.cmdMu.Lock()
	defer b.cmdMu.Unlock()

	if b.cmdStore != nil {
		return b.cmdStore, nil
	}

	store, err := newSQLCommandStoreWithDialect(b.store.DB, b.store.Dialect)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "backend.command_store", "command store")
	}

	b.cmdStore = store

	return store, nil
}

// QueryStore returns the SQL query store, creating it on first call.
// All calls return the same instance. Goroutine-safe.
func (b *SQLBackend) QueryStore() (*SQLQueryStore, error) {
	b.queryMu.Lock()
	defer b.queryMu.Unlock()

	if b.queryStore != nil {
		return b.queryStore, nil
	}

	store, err := newSQLQueryStoreWithDialect(b.store.DB, b.store.Dialect)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "backend.query_store", "query store")
	}

	b.queryStore = store

	return store, nil
}
