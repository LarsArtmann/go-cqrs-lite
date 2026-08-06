package bbolt

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	bolt "go.etcd.io/bbolt"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

// Backend is a facade providing access to all bbolt-backed stores sharing a
// single *bbolt.DB via disjoint buckets. The Backend OWNS the *bbolt.DB.
type Backend struct {
	db         *bolt.DB
	events     *EventStore
	snapshot   *SnapshotStore
	checkpoint *CheckpointStore
	readModels kv.Store
}

// Open creates a new Backend by opening a bbolt database at the given path.
// The Backend owns the *bbolt.DB — Close will close it.
func Open(path string, logger *slog.Logger) (*Backend, error) {
	return OpenWith(path, &bolt.Options{Timeout: 5 * time.Second}, logger)
}

// OpenWith creates a new Backend with custom bbolt.Options.
// Use this for fine-grained control over sync behavior, timeout, etc.
// The Backend owns the *bbolt.DB — Close will close it.
func OpenWith(path string, opts *bolt.Options, logger *slog.Logger) (*Backend, error) {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}

	if opts == nil {
		opts = &bolt.Options{Timeout: 5 * time.Second}
	}

	db, err := bolt.Open(path, 0o600, opts)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "bbolt.open_backend",
			"open bbolt database at "+path)
	}

	return newBackend(db, logger)
}

// NewBackend wraps an existing *bbolt.DB. The Backend does NOT own the DB.
func NewBackend(database *bolt.DB, logger *slog.Logger) (*Backend, error) {
	return newBackend(database, logger)
}

func newBackend(database *bolt.DB, logger *slog.Logger) (*Backend, error) {
	if err := createBuckets(database); err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "bbolt.create_buckets",
			"create CQRS buckets")
	}

	events, err := NewStore(database, logger)
	if err != nil {
		return nil, err
	}

	snap, err := NewSnapshotStore(database, logger)
	if err != nil {
		return nil, err
	}

	checkpt, err := NewCheckpointStore(database, logger)
	if err != nil {
		return nil, err
	}

	readMods, err := NewKVStore(database)
	if err != nil {
		return nil, err
	}

	return &Backend{
		db:         database,
		events:     events,
		snapshot:   snap,
		checkpoint: checkpt,
		readModels: readMods,
	}, nil
}

func (b *Backend) EventStore() *EventStore           { return b.events }
func (b *Backend) SnapshotStore() *SnapshotStore     { return b.snapshot }
func (b *Backend) CheckpointStore() *CheckpointStore { return b.checkpoint }
func (b *Backend) ReadModels() kv.Store              { return b.readModels }
func (b *Backend) DB() *bolt.DB                      { return b.db }

// Close closes the underlying *bbolt.DB.
func (b *Backend) Close() error {
	return wrapBucketErr(b.db.Close(), "bbolt.close_backend", "close bbolt database")
}

// GracefulClose is like Close but bounded by the context.
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

// DiskUsage estimates the on-disk size of the bbolt file.
func (b *Backend) DiskUsage() (uint64, error) {
	st, err := os.Stat(b.db.Path())
	if err != nil {
		return 0, fmt.Errorf("stat bbolt file: %w", err)
	}

	return uint64(st.Size()), nil
}
