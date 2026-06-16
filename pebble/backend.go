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
	db       *pebble.DB
	events   *EventStore
	snapshot *SnapshotStore   //nolint:unused // lazy-initialized on first call
	checkpt  *CheckpointStore //nolint:unused // lazy-initialized on first call
	// snapshot and checkpoint are created eagerly in Open() — they share the db.
	// The nolint:unused suppressions above are placeholders in case future
	// refactoring moves to lazy initialization.
}

// Open creates a new Backend by opening a Pebble database at the given directory.
// The Backend owns the *pebble.DB — Close() will close it.
func Open(dir string, opts *pebble.Options, logger *slog.Logger) (*Backend, error) {
	db, err := pebble.Open(dir, opts)
	if err != nil {
		return nil, fmt.Errorf("pebble: open backend: %w", err)
	}

	return &Backend{
		db:       db,
		events:   NewStore(db, logger),
		snapshot: NewSnapshotStore(db, logger),
		checkpt:  NewCheckpointStore(db, logger),
	}, nil
}

// NewBackend wraps an existing *pebble.DB into a Backend.
// The Backend does NOT own the DB — the caller is responsible for closing it.
// Use Open() instead if you want the Backend to own the DB lifecycle.
func NewBackend(db *pebble.DB, logger *slog.Logger) *Backend {
	return &Backend{
		db:       db,
		events:   NewStore(db, logger),
		snapshot: NewSnapshotStore(db, logger),
		checkpt:  NewCheckpointStore(db, logger),
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
	// Pebble stores don't hold external resources beyond the DB handle.
	// Closing the DB invalidates all iterators and future operations.
	return b.db.Close()
}
