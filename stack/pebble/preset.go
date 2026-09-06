package pebble

import (
	"context"
	"log/slog"
	"math"

	"github.com/cockroachdb/pebble"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	cqrspebble "github.com/larsartmann/go-cqrs-lite/storage/pebble/v4"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// Option configures the Pebble preset.
type Option func(*config)

type config struct {
	pebbleOpts *pebble.Options
	logger     *slog.Logger
	durability stack.DurabilityTier
	blockCache *pebble.Cache // created by WithBlockCacheSize; Unref'd after Open
}

func defaultConfig() config {
	return config{
		pebbleOpts: cqrspebble.DefaultOptions(),
		logger:     slog.Default(),
		durability: stack.DurabilityNormal,
	}
}

// WithDurability sets the durability tier for the Pebble backend. This maps
// to Pebble's WAL and per-write sync settings (see [tierToSettings]):
//
//   - [stack.DurabilityStrict]  → WAL enabled, sync writes (fsync per write)
//   - [stack.DurabilityNormal]  → WAL enabled, async writes (no per-write
//     fsync — safe against app crash, not kernel crash). The default.
//   - [stack.DurabilityRelaxed] → DisableWAL=true + async writes (writes go
//     to memtable only; data loss on crash)
//
// The chosen tier is recorded on the Bundle via [stack.WithDurability] so
// benchmark tools can compare backends at the same durability level.
func WithDurability(tier stack.DurabilityTier) Option {
	return func(c *config) { c.durability = tier }
}

// tierToSettings translates a durability tier to Pebble's WAL and per-write
// sync settings:
//
//   - Strict (and any unrecognized value): WAL enabled, sync writes — the
//     safest interpretation.
//   - Normal: WAL enabled, async writes (no per-write fsync).
//   - Relaxed: DisableWAL + async writes. Both flags matter: with the WAL
//     disabled, pebble.Sync degrades to a memtable flush per write — the
//     slowest possible path — so the "fast" tier must also drop the sync.
func tierToSettings(tier stack.DurabilityTier) (bool, bool) {
	switch tier {
	case stack.DurabilityStrict:
		return false, false
	case stack.DurabilityNormal:
		return false, true
	case stack.DurabilityRelaxed:
		return true, true
	default:
		return false, false
	}
}

// WithPebbleOptions overrides the default PebbleDB options. The preset ships
// with [cqrspebble.DefaultOptions] (bloom filters at 10 bits/key for ~1% FPR
// on point reads, MaxConcurrentCompactions=4 for write throughput). Callers
// who need different settings pass a fully constructed *pebble.Options.
func WithPebbleOptions(opts *pebble.Options) Option {
	return func(c *config) {
		if opts != nil {
			c.pebbleOpts = opts
		}
	}
}

// WithMemTableSize sets the Pebble write-buffer (memtable) size. Larger
// memtables batch more writes in memory before flushing, trading RAM for
// write throughput. Zero or negative leaves the pebble default.
//
// Operator knob for the deploy-time vision: read it from config/env and pass
// it here instead of hand-building *pebble.Options.
func WithMemTableSize(bytes int64) Option {
	return func(c *config) {
		if bytes > 0 {
			c.pebbleOpts.MemTableSize = uint64(bytes)
		}
	}
}

// WithBlockCacheSize sets the shared block-cache size (uncompressed sstable
// blocks). Bigger caches speed repeated reads. Zero or negative leaves the
// pebble default. The cache created here is released after Open; the database
// keeps its own reference for its lifetime.
func WithBlockCacheSize(bytes int64) Option {
	return func(c *config) {
		if bytes <= 0 {
			return
		}

		if c.blockCache != nil {
			c.blockCache.Unref() // replaced by a later WithBlockCacheSize
		}

		c.blockCache = pebble.NewCache(bytes)
		c.pebbleOpts.Cache = c.blockCache
	}
}

// WithWALBytesPerSync batches write-ahead-log fsyncs by bytes written
// (background sync every N bytes instead of relying on Sync writes alone).
// Zero or negative leaves the pebble default (0 = no background syncing).
func WithWALBytesPerSync(bytes int64) Option {
	return func(c *config) {
		if bytes > 0 {
			c.pebbleOpts.WALBytesPerSync = int(bytes)
		}
	}
}

// WithPebbleCompression sets sstable block compression for every level
// (e.g. pebble.NoCompression, pebble.SnappyCompression, pebble.ZstdCompression).
// Compression trades CPU for disk space; event payloads that are already
// compressed (encrypted/CBOR) may prefer NoCompression.
func WithPebbleCompression(compression pebble.Compression) Option {
	return func(c *config) {
		for i := range c.pebbleOpts.Levels {
			c.pebbleOpts.Levels[i].Compression = compression
		}
	}
}

// WithLogger sets the slog.Logger used by PebbleDB stores. Defaults to slog.Default().
func WithLogger(logger *slog.Logger) Option {
	return func(c *config) { c.logger = logger }
}

// Bundle wraps [stack.Bundle] with Pebble-specific backup and observability
// capabilities. It embeds *stack.Bundle, so all Bundle fields and methods are
// available directly.
//
// Deprecated: removed in v5 (ADR-0123): system.System is the single
// composition root. Migrate to system.New with DeploymentConfig before v5.
type Bundle struct {
	*stack.Bundle

	backend *cqrspebble.Backend
}

// Checkpoint creates a point-in-time physical snapshot of the PebbleDB
// database at dir. The target directory must not already exist.
// Use this for backups before migrations or for disaster recovery.
func (b *Bundle) Checkpoint(dir string) error {
	return b.backend.Checkpoint(dir)
}

// NewSnapshot returns a consistent read view of the database at the current
// moment. Useful for reading events while writes are in-flight without
// blocking writers. The caller must call Close on the returned snapshot.
func (b *Bundle) NewSnapshot() *pebble.Snapshot {
	return b.backend.NewSnapshot()
}

// Flush writes all buffered writes to stable storage. Call this before a
// Checkpoint to ensure the backup includes all recent writes.
func (b *Bundle) Flush() error {
	return b.backend.Flush()
}

// Metrics returns PebbleDB LSM-tree metrics for health checks and dashboards.
// Use BlockCacheHitRate() to monitor cache effectiveness.
func (b *Bundle) Metrics() cqrspebble.PebbleMetrics {
	return b.backend.Metrics()
}

// GracefulClose is like Close but bounded by the given context. If the
// context is cancelled before Close finishes, the context error is returned
// and the close continues in the background.
func (b *Bundle) GracefulClose(ctx context.Context) error {
	return b.backend.GracefulClose(ctx)
}

// New opens a PebbleDB database at dir and returns a fully-wired [Bundle].
//
// dir is the filesystem path for the PebbleDB database. The directory is
// created if it does not exist.
//
// Events, commands, queries, snapshots, and checkpoints are persisted to disk
// with CBOR envelopes. Read models use the same *pebble.DB via a shared
// kv.Store (KVAdapter) — use kv.WithTypedKeyPrefix to namespace each read
// model type. The event bus uses watermill.EventBus (GoChannel, in-process).
//
// The returned Bundle owns the *pebble.DB; Close releases it along with all
// stores. Use Checkpoint(dir) for point-in-time backups.
// On any setup failure the database is closed before the error is returned.
//
// Deprecated: removed in v5 (ADR-0123): system.System is the single
// composition root. Migrate to system.New with DeploymentConfig before v5.
func New(dir string, opts ...Option) (*Bundle, error) {
	cfg := defaultConfig()

	for _, opt := range opts {
		opt(&cfg)
	}

	// Translate durability tier to Pebble WAL and per-write sync settings.
	disableWAL, asyncStores := tierToSettings(cfg.durability)
	if disableWAL {
		cfg.pebbleOpts.DisableWAL = true
	}

	var backendOpts []cqrspebble.BackendOption
	if asyncStores {
		backendOpts = append(backendOpts, cqrspebble.WithBackendAsyncWrites())
	}

	backend, err := cqrspebble.Open(dir, cfg.pebbleOpts, cfg.logger, backendOpts...)
	if err != nil {
		if cfg.blockCache != nil {
			cfg.blockCache.Unref()
		}

		return nil, errorfamily.WrapInfrastructure(err, "pebble_preset.open_backend",
			"open pebble backend")
	}

	// The DB holds its own cache reference; drop ours.
	if cfg.blockCache != nil {
		cfg.blockCache.Unref()
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
		stack.WithDiskSize(func() int64 { return safeInt64(backend.DiskUsage()) }),
		stack.WithDurability(cfg.durability),
		stack.WithCapabilities(stack.Capabilities{
			Backend:    "pebble",
			Persistent: true,
			Embedded:   true,
			DurabilityRange: []stack.DurabilityTier{
				stack.DurabilityStrict,
				stack.DurabilityNormal,
				stack.DurabilityRelaxed,
			},
		}),
	)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "pebble_preset.wire_bundle",
			"wire pebble bundle")
	}

	return &Bundle{Bundle: b, backend: backend}, nil
}

func safeInt64(v uint64) int64 {
	if v > math.MaxInt64 {
		return math.MaxInt64
	}

	return int64(v)
}
