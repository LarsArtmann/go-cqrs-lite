// Package bbolt provides a [stack.Bundle] preset backed by an embedded bbolt
// (B+tree) database. bbolt is a pure-Go key-value store maintained by the etcd
// team, using a single-writer B+tree model.
//
// Unlike Pebble (LSM tree, concurrent writes, bloom filters), bbolt uses a
// B+tree with a single-writer lock. This makes writes fully serialized but
// provides excellent point-read performance and predictable latency. The
// single-writer model eliminates the need for per-stream locking — version
// checks and event writes happen atomically inside one transaction.
package bbolt

import (
	"log/slog"
	"math"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	cqrsbbolt "github.com/larsartmann/go-cqrs-lite/storage/bbolt/v4"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// Option configures the bbolt preset.
type Option func(*config)

type config struct {
	logger *slog.Logger
}

func defaultConfig() config {
	return config{logger: slog.Default()}
}

// WithLogger sets the slog.Logger used by bbolt stores.
func WithLogger(logger *slog.Logger) Option {
	return func(c *config) { c.logger = logger }
}

// New opens a bbolt database at path and returns a fully-wired Bundle.
//
// path is the filesystem path for the bbolt database file. The file is created
// if it does not exist. bbolt obtains an exclusive file lock — only one process
// can open the database at a time.
//
// Events, snapshots, and checkpoints are persisted with CBOR envelopes. Read
// models use the same *bbolt.DB via a shared kv.Store (cqrs_kv bucket). The
// event bus uses watermill.EventBus (GoChannel, in-process).
//
// The returned Bundle owns the *bbolt.DB; Close releases it along with all
// stores. On any setup failure the database is closed before the error is
// returned.
func New(path string, opts ...Option) (*stack.Bundle, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	backend, err := cqrsbbolt.Open(path, cfg.logger)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "bbolt_preset.open_backend",
			"open bbolt backend")
	}

	b, err := stack.New(
		stack.WithEventStore(backend.EventStore()),
		stack.WithSnapshotStore(backend.SnapshotStore()),
		stack.WithCheckpointStore(backend.CheckpointStore()),
		stack.WithReadModels(backend.ReadModels()),
		stack.WithBus(cqrswatermill.NewEventBus()),
		stack.WithCloser(backend),
		stack.WithDiskSize(func() int64 {
			size, _ := backend.DiskUsage()

			return safeInt64(size)
		}),
		stack.WithDurability(stack.DurabilityStrict),
		stack.WithCapabilities(stack.Capabilities{
			Backend:    "bbolt",
			Persistent: true,
			Embedded:   true,
			DurabilityRange: []stack.DurabilityTier{
				stack.DurabilityStrict,
			},
		}),
	)
	if err != nil {
		_ = backend.Close()

		return nil, errorfamily.WrapInfrastructure(err, "bbolt_preset.wire_bundle",
			"wire bbolt bundle")
	}

	return b, nil
}

func safeInt64(v uint64) int64 {
	if v > math.MaxInt64 {
		return math.MaxInt64
	}

	return int64(v)
}
