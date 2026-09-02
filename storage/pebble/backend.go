package pebble

import (
	"context"
	"log/slog"

	"github.com/cockroachdb/pebble"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// Backend is a facade that provides access to all Pebble-backed stores
// sharing a single *pebble.DB via disjoint key prefixes.
//
// Unlike SQLBackend (which borrows the *sql.DB), Backend OWNS the *pebble.DB.
// Calling Close() closes the database AND all stores.
//
// By default every store writes with pebble.Sync (fsync per write). Pass
// [WithBackendAsyncWrites] to construct all stores with asynchronous writes.
type Backend struct {
	database *pebble.DB
	events   *EventStore
	commands *CommandStore
	queries  *QueryStore
	snapshot *SnapshotStore
	checkpt  *CheckpointStore
	readMods kv.Store
}

// BackendOption configures how a [Backend] constructs its stores.
type BackendOption func(*backendConfig)

type backendConfig struct {
	asyncWrites bool
}

// WithBackendAsyncWrites constructs every Backend store — events, commands,
// queries, snapshots, checkpoints, and the shared read-model KV store — with
// asynchronous writes: each write is appended to the write-ahead log in the
// page cache but returns before the fsync. Writes survive an application
// crash; a kernel or power crash may lose the most recent ones.
//
// This is the per-store [WithAsyncWrites] / [WithCommandAsyncWrites] / ...
// family applied to every store at once, typically driven by a durability
// tier. The default Backend keeps synchronous
// writes (pebble.Sync per write).
func WithBackendAsyncWrites() BackendOption {
	return func(cfg *backendConfig) { cfg.asyncWrites = true }
}

// Open creates a new Backend by opening a Pebble database at the given directory.
// The Backend owns the *pebble.DB — Close() will close it.
func Open(
	dir string,
	opts *pebble.Options,
	logger *slog.Logger,
	backendOpts ...BackendOption,
) (*Backend, error) {
	database, err := pebble.Open(dir, opts)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "pebble.open_backend",
			"open pebble database")
	}

	return newBackend(database, logger, backendOpts...)
}

// NewBackend wraps an existing *pebble.DB into a Backend.
// The Backend does NOT own the DB — the caller is responsible for closing it.
// Use Open() instead if you want the Backend to own the DB lifecycle.
// Returns ErrNilDatabase if database is nil.
func NewBackend(
	database *pebble.DB,
	logger *slog.Logger,
	backendOpts ...BackendOption,
) (*Backend, error) {
	return newBackend(database, logger, backendOpts...)
}

func newBackend(
	database *pebble.DB,
	logger *slog.Logger,
	backendOpts ...BackendOption,
) (*Backend, error) {
	cfg := backendConfig{}
	for _, opt := range backendOpts {
		opt(&cfg)
	}

	var storeOpts []StoreOption
	var commandOpts []CommandStoreOption
	var queryOpts []QueryStoreOption
	var snapshotOpts []SnapshotOption
	var checkpointOpts []CheckpointOption

	kvOpts := []KVOption{WithBorrowedDB()}

	if cfg.asyncWrites {
		storeOpts = append(storeOpts, WithAsyncWrites())
		commandOpts = append(commandOpts, WithCommandAsyncWrites())
		queryOpts = append(queryOpts, WithQueryAsyncWrites())
		snapshotOpts = append(snapshotOpts, WithSnapshotAsyncWrites())
		checkpointOpts = append(checkpointOpts, WithCheckpointAsyncWrites())
	} else {
		// Standalone KVAdapter defaults to async writes; the Backend's shared
		// read-model store keeps its historical synchronous behavior.
		kvOpts = append(kvOpts, WithKVSyncWrites())
	}

	events, err := NewStore(database, logger, storeOpts...)
	if err != nil {
		return nil, err
	}

	commands, err := NewCommandStore(database, logger, commandOpts...)
	if err != nil {
		return nil, err
	}

	queries, err := NewQueryStore(database, logger, queryOpts...)
	if err != nil {
		return nil, err
	}

	snapshot, err := NewSnapshotStore(database, logger, snapshotOpts...)
	if err != nil {
		return nil, err
	}

	checkpt, err := NewCheckpointStore(database, logger, checkpointOpts...)
	if err != nil {
		return nil, err
	}

	readMods, err := NewKVStore(database, kvOpts...)
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
	return closeAndWrap(b.database, "pebble.close_backend", "close pebble database")
}

// GracefulClose is like Close but bounded by the given context. If the
// context is cancelled before Close finishes, the context error is returned
// and the close continues in the background. Use this in shutdown handlers
// to avoid hanging on slow flushes:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	err := backend.GracefulClose(ctx)
func (b *Backend) GracefulClose(ctx context.Context) error {
	done := make(chan error, 1)

	go func() { done <- b.Close() }()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
