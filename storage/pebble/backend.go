package pebble

import (
	"fmt"
	"log/slog"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/kv/v3"
)

// Backend is a facade that provides access to all Pebble-backed stores
// sharing a single *pebble.DB via disjoint key prefixes.
//
// Unlike SQLBackend (which borrows the *sql.DB), Backend OWNS the *pebble.DB.
// Calling Close() closes the database AND all stores.
type Backend struct {
	database *pebble.DB
	events   *EventStore
	commands *CommandStore
	queries  *QueryStore
	snapshot *SnapshotStore
	checkpt  *CheckpointStore
	readMods kv.Store
}

// Open creates a new Backend by opening a Pebble database at the given directory.
// The Backend owns the *pebble.DB — Close() will close it.
func Open(dir string, opts *pebble.Options, logger *slog.Logger) (*Backend, error) {
	database, err := pebble.Open(dir, opts)
	if err != nil {
		return nil, fmt.Errorf("pebble: open backend: %w", err)
	}

	return newBackend(database, logger)
}

// NewBackend wraps an existing *pebble.DB into a Backend.
// The Backend does NOT own the DB — the caller is responsible for closing it.
// Use Open() instead if you want the Backend to own the DB lifecycle.
// Returns ErrNilDatabase if database is nil.
func NewBackend(database *pebble.DB, logger *slog.Logger) (*Backend, error) {
	return newBackend(database, logger)
}

func newBackend(database *pebble.DB, logger *slog.Logger) (*Backend, error) {
	events, err := NewStore(database, logger)
	if err != nil {
		return nil, err
	}

	commands, err := NewCommandStore(database, logger)
	if err != nil {
		return nil, err
	}

	queries, err := NewQueryStore(database, logger)
	if err != nil {
		return nil, err
	}

	snapshot, err := NewSnapshotStore(database, logger)
	if err != nil {
		return nil, err
	}

	checkpt, err := NewCheckpointStore(database, logger)
	if err != nil {
		return nil, err
	}

	readMods, err := NewKVStore(database, WithBorrowedDB(), WithKVSyncWrites())
	if err != nil {
		return nil, err
	}

	return &Backend{
		database: database,
		events:   events,
		commands: commands,
		queries:  queries,
		snapshot: snapshot,
		checkpt:  checkpt,
		readMods: readMods,
	}, nil
}

// EventStore returns the event store (Save, Load, Journal, SeekableJournal).
func (b *Backend) EventStore() *EventStore { return b.events }

// CommandStore returns the command store (Save, Load, CommandJournal, SeekableCommandJournal).
func (b *Backend) CommandStore() *CommandStore { return b.commands }

// QueryStore returns the query store (SaveQuery, LoadQueries, QueryJournal, SeekableQueryJournal).
func (b *Backend) QueryStore() *QueryStore { return b.queries }

// SnapshotStore returns the snapshot store.
func (b *Backend) SnapshotStore() *SnapshotStore { return b.snapshot }

// CheckpointStore returns the checkpoint store.
func (b *Backend) CheckpointStore() *CheckpointStore { return b.checkpt }

// ReadModels returns the shared key-value store for read models.
// Each kv.TypedStore should use WithKeyPrefix to avoid key collisions
// between different read model types sharing this store.
// The returned kv.Store does NOT own the *pebble.DB — Backend.Close() handles cleanup.
func (b *Backend) ReadModels() kv.Store { return b.readMods }

// Close closes all stores and the underlying *pebble.DB.
// After Close, all store operations will return ErrClosed.
func (b *Backend) Close() error {
	err := b.database.Close()
	if err != nil {
		return fmt.Errorf("pebble: close backend: %w", err)
	}

	return nil
}
