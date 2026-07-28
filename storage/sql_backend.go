package storage

import (
	"database/sql"
	"errors"
	"sync"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4/eventstore"
	"github.com/larsartmann/go-cqrs-lite/storage/v4/readmodel"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// SQLBackend is a facade that provides access to all SQL-backed stores
// sharing a single database connection. All store accessors are goroutine-safe.
type SQLBackend struct {
	store *SQLEventStore

	cmdMu        sync.Mutex
	cmdStore     *SQLCommandStore
	queryMu      sync.Mutex
	queryStore   *SQLQueryStore
	snapMu       sync.Mutex
	snapStore    *SQLSnapshotStore
	checkpointMu sync.Mutex
	cpStore      *SQLCheckpointStore

	kvMu    sync.Mutex
	kvStore *SQLKVStore
}

func NewSQLBackend(db *sql.DB) (*SQLBackend, error) {
	return newSQLBackendWithDialect(db, sqlpkg.PostgresDialect{})
}

func NewSQLiteBackend(db *sql.DB) (*SQLBackend, error) {
	return newSQLBackendWithDialect(db, sqlpkg.SQLiteDialect{})
}

func NewDuckDBBackend(db *sql.DB) (*SQLBackend, error) {
	return newSQLBackendWithDialect(db, sqlpkg.DuckDBDialect{})
}

func NewSQLBackendWithDialect(db *sql.DB, d sqlpkg.Dialect) (*SQLBackend, error) {
	return newSQLBackendWithDialect(db, d)
}

func newSQLBackendWithDialect(db *sql.DB, d sqlpkg.Dialect) (*SQLBackend, error) {
	store, err := eventstore.NewSQLEventStoreWithDialect(db, d)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "backend.event_store", "event store")
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
		return nil, errorfamily.WrapInfrastructure(err, "backend.command_store", "command store")
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
		return nil, errorfamily.WrapInfrastructure(err, "backend.query_store", "query store")
	}

	b.queryStore = store

	return store, nil
}

// SnapshotStore returns the SQL snapshot store, creating it on first call.
// All calls return the same instance. Goroutine-safe.
func (b *SQLBackend) SnapshotStore() (*SQLSnapshotStore, error) {
	b.snapMu.Lock()
	defer b.snapMu.Unlock()

	if b.snapStore != nil {
		return b.snapStore, nil
	}

	store, err := eventstore.NewSQLSnapshotStoreWithDialect(b.store.DB, b.store.Dialect)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "backend.snapshot_store", "snapshot store")
	}

	b.snapStore = store

	return store, nil
}

// CheckpointStore returns the SQL checkpoint store, creating it on first call.
// All calls return the same instance. Goroutine-safe.
func (b *SQLBackend) CheckpointStore() (*SQLCheckpointStore, error) {
	b.checkpointMu.Lock()
	defer b.checkpointMu.Unlock()

	if b.cpStore != nil {
		return b.cpStore, nil
	}

	store, err := eventstore.NewSQLCheckpointStoreWithDialect(b.store.DB, b.store.Dialect)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"backend.checkpoint_store",
			"checkpoint store",
		)
	}

	b.cpStore = store

	return store, nil
}

// KVStore returns a [kv.Store] backed by the cqrs_kv table, creating it on
// first call. All calls return the same instance. Goroutine-safe.
//
// Use this as the read-model backend for SQL-backed presets so that read
// models persist across process restarts instead of living only in memory.
func (b *SQLBackend) KVStore() (kv.Store, error) {
	b.kvMu.Lock()
	defer b.kvMu.Unlock()

	if b.kvStore != nil {
		return b.kvStore, nil
	}

	store, err := readmodel.NewSQLKVStoreWithDialect(b.store.DB, b.store.Dialect)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "backend.kv_store", "kv store")
	}

	b.kvStore = store

	return store, nil
}

// Close closes all stores created through this backend.
// The underlying *sql.DB is NOT closed — it is borrowed from the caller.
func (b *SQLBackend) Close() error {
	var errs []error

	if b.cmdStore != nil {
		if err := b.cmdStore.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if b.queryStore != nil {
		if err := b.queryStore.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if b.snapStore != nil {
		if err := b.snapStore.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if b.cpStore != nil {
		if err := b.cpStore.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if b.kvStore != nil {
		if err := b.kvStore.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if err := b.store.Close(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}
