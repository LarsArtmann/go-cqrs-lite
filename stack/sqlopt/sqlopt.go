// Package sqlopt provides shared helpers for converting a storage.SQLBackend
// into stack.Option values. It is the single home for the option-assembly
// logic used by every SQL-backed preset (sqlite, postgres, turso).
//
// This is a separate package within the stack module so that the base stack
// package does not acquire a storage dependency; consumers using non-SQL
// backends (memory, pebble) never import this package and never pull storage
// into their build.
package sqlopt

import (
	"database/sql"
	"fmt"
	"io"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
)

// OpenDBOrErr opens a SQL database and wraps any failure as an Infrastructure
// error. Collapses the repeated "sql.Open + WrapInfrastructure" boilerplate in
// preset openBackend functions. The driver and dsn are passed to sql.Open; the
// code is the stable errorfamily code (e.g. "sqlite_preset.open_primary").
func OpenDBOrErr(driver, dsn, code string) (*sql.DB, error) {
	sqlDB, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, code,
			fmt.Sprintf("open %s %q", driver, dsn))
	}

	return sqlDB, nil
}

// NewSecondaryBackend wraps the create-secondary-backend pattern shared by the
// postgres, sqlite, and turso presets: open the secondary DB via openDB,
// construct its backend via newBackend, and return both along with a closer
// that releases both. On any create failure the secondary DB is closed and an
// Infrastructure error is returned tagged with errCode.
//
// dsn is the connection string — purely diagnostic in the error message.
// openDB returns the opened *sql.DB (preset-specific: applies WAL/PRAGMA etc).
// newBackend converts the *sql.DB into a *storage.SQLBackend.
// errCode is the stable errorfamily code, e.g. "sqlite.create_backend".
func NewSecondaryBackend(
	dsn string,
	openDB func() (*sql.DB, error),
	newBackend func(*sql.DB) (*storage.SQLBackend, error),
	errCode string,
) (*storage.SQLBackend, io.Closer, error) {
	secDB, err := openDB()
	if err != nil {
		return nil, nil, err
	}

	secBackend, err := newBackend(secDB)
	if err != nil {
		_ = secDB.Close()

		return nil, nil, errorfamily.WrapInfrastructure(err, errCode,
			fmt.Sprintf("create backend for %q", dsn))
	}

	closer := stack.NewMultiCloser(secBackend, stack.NewFuncCloser(secDB.Close))

	return secBackend, closer, nil
}

// AllOptions assembles the full stack.Option set from a storage.SQLBackend.
// The event store is always present (eager); the lazy stores (command, query,
// snapshot, checkpoint) are included only when they construct successfully.
// This is the common case: one database backs every store.
func AllOptions(backend *storage.SQLBackend) []stack.Option {
	return append(EventStoreOptions(backend), QueryStoreOptions(backend)...)
}

// EventStoreOptions assembles options for the event-sourcing write model:
// the event store (eager), snapshots, and checkpoints. Used by multi-DB
// presets that isolate event stores on their own database.
func EventStoreOptions(backend *storage.SQLBackend) []stack.Option {
	opts := []stack.Option{stack.WithEventStore(backend.EventStore())}

	if snapStore, err := backend.SnapshotStore(); err == nil {
		opts = append(opts, stack.WithSnapshotStore(snapStore))
	}

	if cpStore, err := backend.CheckpointStore(); err == nil {
		opts = append(opts, stack.WithCheckpointStore(cpStore))
	}

	return opts
}

// QueryStoreOptions assembles options for the command and query audit stores.
// Used by multi-DB presets that isolate audit stores on their own database.
func QueryStoreOptions(backend *storage.SQLBackend) []stack.Option {
	var opts []stack.Option

	if cmdStore, err := backend.CommandStore(); err == nil {
		opts = append(opts, stack.WithCommandStore(cmdStore))
	}

	if queryStore, err := backend.QueryStore(); err == nil {
		opts = append(opts, stack.WithQueryStore(queryStore))
	}

	return opts
}

// SecondaryStoreOptions opens a secondary database backend (via openBackend)
// and converts it to stack.Options using optionBuilder. Returns nil when
// secondaryDSN is empty. On failure, cleanup is called so the caller's
// primary resources are released.
//
// dialect and label produce meaningful error locations, e.g.
// dialect="sqlite", label="event" → "sqlite_preset.open_event_db".
func SecondaryStoreOptions(
	secondaryDSN, dialect, label string,
	openBackend func(dsn string) (*storage.SQLBackend, io.Closer, error),
	optionBuilder func(*storage.SQLBackend) []stack.Option,
	cleanup func(),
) ([]stack.Option, error) {
	if secondaryDSN == "" {
		return nil, nil
	}

	backend, closer, err := openBackend(secondaryDSN)
	if err != nil {
		cleanup()

		return nil, errorfamily.WrapInfrastructure(err,
			dialect+"_preset.open_"+label+"_db", "open "+label+" database")
	}

	opts := optionBuilder(backend)
	opts = append(opts, stack.WithCloser(closer))

	return opts, nil
}

// MultiDBOverrides handles both event-DB and query-DB overrides in one call.
// It is the common multi-DB pattern shared by every SQL preset.
func MultiDBOverrides(
	eventDSN, queryDSN, dialect string,
	openBackend func(dsn string) (*storage.SQLBackend, io.Closer, error),
	cleanup func(),
) ([]stack.Option, error) {
	evtOpts, err := SecondaryStoreOptions(eventDSN, dialect, "event",
		openBackend, EventStoreOptions, cleanup)
	if err != nil {
		return nil, err
	}

	qOpts, err := SecondaryStoreOptions(queryDSN, dialect, "query",
		openBackend, QueryStoreOptions, cleanup)
	if err != nil {
		return nil, err
	}

	return append(evtOpts, qOpts...), nil
}

// InitStack opens the primary backend, assembles base options, and applies
// multi-DB overrides. Returns the assembled options, the primary backend and
// DB handle (for FinalizeBundle), and a closePrimary cleanup function.
func InitStack(
	dsn, dialect, eventDSN, queryDSN string,
	openBackend func(dsn string) (*sql.DB, *storage.SQLBackend, error),
	openSecondary func(dsn string) (*storage.SQLBackend, io.Closer, error),
) ([]stack.Option, *storage.SQLBackend, *sql.DB, func(), error) {
	sqlDB, backend, err := openBackend(dsn)
	if err != nil {
		return nil, nil, nil, nil, errorfamily.WrapInfrastructure(err,
			dialect+"_preset.open_backend", "open primary backend")
	}

	opts := AllOptions(backend)
	closePrimary := func() { _ = backend.Close(); _ = sqlDB.Close() }

	multiOpts, err := MultiDBOverrides(eventDSN, queryDSN, dialect,
		openSecondary, closePrimary)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	opts = append(opts, multiOpts...)

	return opts, backend, sqlDB, closePrimary, nil
}

// FinalizeBundle appends view options, lifecycle closers, and calls stack.New.
// Shared by SQL presets to eliminate the identical tail of newBundle.
func FinalizeBundle(
	stackOpts []stack.Option,
	backend *storage.SQLBackend,
	sqlDB *sql.DB,
	dialect, viewDSN string,
	openDB func(dsn string) (*sql.DB, error),
	newBackend func(db *sql.DB) (*storage.SQLBackend, error),
) (*stack.Bundle, error) {
	viewOpts, err := ViewOptions(viewDSN, dialect, backend, sqlDB, openDB, newBackend)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err,
			dialect+"_preset.view_options", "build view options")
	}

	stackOpts = append(stackOpts, viewOpts...)
	stackOpts = append(
		stackOpts,
		stack.WithDatabase(sqlDB),
		stack.WithCloser(backend),
		stack.WithCloser(stack.NewFuncCloser(sqlDB.Close)),
	)

	b, err := stack.New(stackOpts...)
	if err != nil {
		_ = backend.Close()
		_ = sqlDB.Close()

		return nil, errorfamily.WrapInfrastructure(err,
			dialect+"_preset.wire_bundle", "wire "+dialect+" bundle")
	}

	return b, nil
}

// ViewOptions builds read-model options from either a separate view database
// (when viewDSN is set) or the primary backend's KV store. The dialect
// parameter produces meaningful error locations. The openDB and newBackend
// callbacks are preset-specific (e.g. SQLite applies WAL/PRAGMA, Postgres
// applies schema migration).
func ViewOptions(
	viewDSN, dialect string,
	primary *storage.SQLBackend,
	sqlDB *sql.DB,
	openDB func(dsn string) (*sql.DB, error),
	newBackend func(db *sql.DB) (*storage.SQLBackend, error),
) ([]stack.Option, error) {
	if viewDSN == "" {
		kvStore, err := primary.KVStore()
		if err != nil {
			_ = primary.Close()
			_ = sqlDB.Close()

			return nil, errorfamily.WrapInfrastructure(err,
				dialect+".kv_store", "create KV store")
		}

		return []stack.Option{stack.WithReadModels(kvStore)}, nil
	}

	viewDB, err := openDB(viewDSN)
	if err != nil {
		_ = primary.Close()
		_ = sqlDB.Close()

		return nil, errorfamily.WrapInfrastructure(err,
			dialect+".open_view_db", "open view database")
	}

	viewBackend, err := newBackend(viewDB)
	if err != nil {
		_ = primary.Close()
		_ = sqlDB.Close()
		_ = viewDB.Close()

		return nil, errorfamily.WrapInfrastructure(err,
			dialect+".create_view_backend", "create view backend")
	}

	kvStore, err := viewBackend.KVStore()
	if err != nil {
		_ = viewBackend.Close()
		_ = primary.Close()
		_ = sqlDB.Close()

		return nil, errorfamily.WrapInfrastructure(err,
			dialect+".view_kv_store", "create KV store for view database")
	}

	return []stack.Option{
		stack.WithReadModels(kvStore),
		stack.WithCloser(viewBackend),
		stack.WithCloser(stack.NewFuncCloser(viewDB.Close)),
	}, nil
}
