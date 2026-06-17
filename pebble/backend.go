package pebble

import (
	"fmt"
	"log/slog"

	"github.com/cockroachdb/pebble"
)

// Backend is a facade that provides access to all Pebble-backed stores
// sharing a single *pebble.DB via disjoint key prefixes.
//
// Unlike SQLBackend (which borrows the *sql.DB), Backend OWNS the *pebble.DB.
// Calling Close() closes the database AND all stores.
type Backend struct {
	database *pebble.DB
	events   *EventStore
	snapshot *SnapshotStore
	checkpt  *CheckpointStore
}

// Open creates a new Backend by opening a Pebble database at the given directory.
// The Backend owns the *pebble.DB — Close() will close it.
func Open(dir string, opts *pebble.Options, logger *slog.Logger) (*Backend, error) {
	database, err := pebble.Open(dir, opts)
	if err != nil {
		return nil, fmt.Errorf("pebble: open backend: %w", err)
	}

	return &Backend{
		database: database,
		events:   NewStore(database, logger),
		snapshot: NewSnapshotStore(database, logger),
		checkpt:  NewCheckpointStore(database, logger),
	}, nil
}

// NewBackend wraps an existing *pebble.DB into a Backend.
// The Backend does NOT own the DB — the caller is responsible for closing it.
// Use Open() instead if you want the Backend to own the DB lifecycle.
func NewBackend(database *pebble.DB, logger *slog.Logger) *Backend {
	return &Backend{
		database: database,
		events:   NewStore(database, logger),
		snapshot: NewSnapshotStore(database, logger),
		checkpt:  NewCheckpointStore(database, logger),
	}
}

// EventStore returns the event store (Save, Load, Journal, SeekableJournal).
func (b *Backend) EventStore() *EventStore { return b.events }

// SnapshotStore returns the snapshot store.
func (b *Backend) SnapshotStore() *SnapshotStore { return b.snapshot }

// CheckpointStore returns the checkpoint store.
func (b *Backend) CheckpointStore() *CheckpointStore { return b.checkpt }

// Close closes all stores and the underlying *pebble.DB.
// After Close, all store operations will return ErrClosed.
func (b *Backend) Close() error {
	err := b.database.Close()
	if err != nil {
		return fmt.Errorf("pebble: close backend: %w", err)
	}

	return nil
}
