package pebble

import (
	"fmt"
	"log/slog"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/stack/v2"
	cqrspebble "github.com/larsartmann/go-cqrs-lite/storage/pebble/v2"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v2"
)

// Option configures the Pebble preset.
type Option func(*config)

type config struct {
	pebbleOpts *pebble.Options
	logger     *slog.Logger
}

func defaultConfig() config {
	return config{
		pebbleOpts: &pebble.Options{}, //nolint:exhaustruct // intentionally empty; override via WithPebbleOptions
		logger:     slog.Default(),
	}
}

// WithPebbleOptions overrides the default PebbleDB options (empty Options{}).
// Use pebble.DefaultOptions() for recommended production settings (bloom
// filters, concurrent compactions).
func WithPebbleOptions(opts *pebble.Options) Option {
	return func(c *config) { c.pebbleOpts = opts }
}

// WithLogger sets the slog.Logger used by PebbleDB stores. Defaults to slog.Default().
func WithLogger(logger *slog.Logger) Option {
	return func(c *config) { c.logger = logger }
}

// New opens a PebbleDB database at dir and returns a fully-wired [stack.Bundle].
//
// dir is the filesystem path for the PebbleDB database. The directory is
// created if it does not exist.
//
// Events, commands, queries, snapshots, and checkpoints are persisted to disk
// with CBOR envelopes. Read models use the same *pebble.DB via a shared
// kv.Store (KVAdapter) — use kv.WithTypedKeyPrefix to namespace each read
// model type. The event bus uses watermill.EventBus (GoChannel, in-process).
//
// The returned Bundle owns the *pebble.DB; Close releases it along with all stores.
// On any setup failure the database is closed before the error is returned.
func New(dir string, opts ...Option) (*stack.Bundle, error) {
	cfg := defaultConfig()

	for _, opt := range opts {
		opt(&cfg)
	}

	backend, err := cqrspebble.Open(dir, cfg.pebbleOpts, cfg.logger)
	if err != nil {
		return nil, fmt.Errorf("pebble preset: open backend: %w", err)
	}

	b, err := stack.New(
		stack.WithEventStore(backend.EventStore()),
		stack.WithCommandStore(backend.CommandStore()),
		stack.WithQueryStore(backend.QueryStore()),
		stack.WithSnapshotStore(backend.SnapshotStore()),
		stack.WithCheckpointStore(backend.CheckpointStore()),
		stack.WithReadModels(backend.ReadModels()),
		stack.WithBus(cqrswatermill.NewEventBus()),
		stack.WithCloser(backend),
	)
	if err != nil {
		return nil, fmt.Errorf("pebble preset: wire bundle: %w", err)
	}

	return b, nil
}
