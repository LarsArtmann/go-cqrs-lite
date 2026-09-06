// Package badgerengine implements a Badger-backed metaengine Engine.
//
// Badger v4 is a pure-Go LSM-tree key-value store. It offers a genuinely
// different cost profile from Pebble and SQLite:
//   - Map/Set point lookups: O(1) LSM point read (comparable to Pebble)
//   - Counter: O(1) increment via read-modify-write, O(N) CounterGet (prefix scan)
//   - SortedMap scan: O(N) prefix scan + Go sort (degraded — no secondary indexes)
//   - Graph: O(degree^depth) BFS via prefix seeks on dual adjacency indexes
//     (forward + reverse marker keys)
//   - Vector: O(N·D) brute-force scan (degraded — no ANN index)
//
// This module exists OUTSIDE the zero-dependency metaengine core (ADR-0062)
// because it requires the dgraph-io/badger/v4 dependency.
package badgerengine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/dgraph-io/badger/v4"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/keycodec"
)

// sep is the key separator. Null byte sorts before all printable characters,
// preventing collisions between collection names and keys.
const sep = "\x00"

// BadgerNsPerOp is the estimated per-operation cost for the Badger engine.
// Badger's LSM-tree architecture provides fast point reads but higher write
// costs due to LSM compaction overhead. Calibrated via BenchmarkCalibration_BadgerSet
// (2026-08-06: ~4300 ns/op median).
const BadgerNsPerOp = 4300.0

// BadgerNsPerRead is the measured per-READ-operation cost (LSM point lookup +
// JSON decode). It feeds ReadCosts.NsPerPointLookup in Profile.
// Re-calibrated 2026-08-30 via BenchmarkCalibration_BadgerGet (~1085 ns/op
// median of 3; prior 2026-08-06 measurement was ~1200).
const BadgerNsPerRead = 1100.0

// BadgerNsPerWrite is the measured per-WRITE-operation cost.
// Calibrated via BenchmarkCalibration_BadgerSet (2026-08-06: ~4300 ns/op median).
// Counter operations are even higher (~5800 ns/op) due to read-modify-write.
const BadgerNsPerWrite = 4300.0

type badgerEngine struct {
	metaengine.Calibration

	db          *badger.DB
	ownsDB      bool
	syncWrites  bool
	inMemory    bool
	persistence metaengine.Persistence
	mu          sync.Mutex // guards MapUpdate, StreamAppend, StreamAppendExpected
	logSeq      sync.Map   // collection → *atomic.Int64 (log sequence counter)
	mmSeq       sync.Map   // collection → *atomic.Int64 (multimap sequence counter)
	streamSeq   sync.Map   // "col\x00sid" → *atomic.Int64 (per-stream sequence)
	journalSeq  sync.Map   // collection → *atomic.Int64 (global journal sequence)
}

// Option configures a Badger engine at construction time.
type Option func(*engineConfig)

type engineConfig struct {
	syncWrites bool
}

// WithAsyncWrites skips the per-write fsync (badger SyncWrites=false): each
// write returns before the value-log sync. Survives an application crash
// (the value log is still written and replayed on open); a kernel or power
// crash may lose the most recent writes. The default engine fsyncs every
// write.
func WithAsyncWrites() Option {
	return func(cfg *engineConfig) { cfg.syncWrites = false }
}

// NewBadgerEngine creates a Badger-backed metaengine engine. If dir is empty,
// an in-memory database is used (for testing); otherwise dir is the on-disk
// database directory (persisted across opens). The caller owns the returned
// Engine and must call Close.
func NewBadgerEngine(dir string, engineOpts ...Option) (metaengine.Engine, error) {
	cfg := engineConfig{syncWrites: true}
	for _, opt := range engineOpts {
		opt(&cfg)
	}

	opts := badger.DefaultOptions(dir)
	persistence := metaengine.PersistencePersistent

	if dir == "" {
		opts = badger.DefaultOptions("").WithInMemory(true)
		persistence = metaengine.PersistenceVolatile
	}

	// Suppress Badger's default logger noise in production.
	opts = opts.WithLogger(nil).WithSyncWrites(cfg.syncWrites)

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("badgerengine: open: %w", err)
	}

	eng := &badgerEngine{
		db:          db,
		ownsDB:      true,
		syncWrites:  cfg.syncWrites,
		inMemory:    dir == "",
		persistence: persistence,
	}

	if err := eng.seedSeqCounters(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("badgerengine: seed seq counters: %w", err)
	}

	return eng, nil
}

// NewBadgerEngineFromDB wraps an existing *badger.DB. The caller retains
// ownership of the DB — Close on the engine is a no-op.
func NewBadgerEngineFromDB(db *badger.DB) (metaengine.Engine, error) {
	eng := &badgerEngine{
		db:          db,
		ownsDB:      false,
		persistence: metaengine.PersistencePersistent,
	}

	if err := eng.seedSeqCounters(); err != nil {
		return nil, fmt.Errorf("badgerengine: seed seq counters: %w", err)
	}

	return eng, nil
}

func (e *badgerEngine) Profile() metaengine.EngineProfile {
	p := metaengine.EngineProfile{
		Name:        "badger",
		NsPerOp:     BadgerNsPerOp,
		NsPerWrite:  BadgerNsPerWrite,
		Persistence: e.persistence,
		// Per-read-pattern calibrated costs (see calibration_bench_test.go;
		// measured 2026-08-30, medians of 3 runs on 32 threads, load ~3.8).
		// Badger is a KV engine: filtered scans and aggregations have no SQL
		// pushdown, so they degrade to a full scan with Go-side work — hence
		// per-row scan costs (~630-650ns) far above the point-lookup cost.
		ReadCosts: metaengine.ReadCosts{
			// Indexed LSM point lookup (BenchmarkCalibration_BadgerGet).
			NsPerPointLookup: BadgerNsPerRead,
			// ~639 ns/row (BenchmarkCalibration_Badger_FilteredScan): full
			// MapScan + Go-side predicate over 10K rows, ~50% match.
			NsPerFilteredScan: 650,
			// ~164 ns/row (BenchmarkCalibration_Badger_CounterScan):
			// CounterGet prefix scan over 1K counters — the ReadAggregate path.
			NsPerAggregate: 165,
			// ~629 ns/row (BenchmarkCalibration_Badger_FullScan): full MapScan,
			// JSON decode of every row.
			NsPerScan: 630,
		},
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap:       metaengine.ComplexityO1,
			metaengine.ADTSet:       metaengine.ComplexityO1,
			metaengine.ADTCounter:   metaengine.ComplexityON,
			metaengine.ADTSortedMap: metaengine.ComplexityON,
			metaengine.ADTLog:       metaengine.ComplexityOLogN,
			metaengine.ADTMultimap:  metaengine.ComplexityOLogN,
			metaengine.ADTGraph:     metaengine.ComplexityODegree, // prefix-scan BFS on adjacency keys
			metaengine.ADTVector:    metaengine.ComplexityON,      // brute-force scan (degraded)
		},
		DegradedADTs: map[metaengine.ADT]bool{
			metaengine.ADTVector: true, // O(N·D) brute-force, no ANN index
		},
	}
	e.ApplyCalibration(&p)

	return p
}

// HealthCheck verifies the underlying Badger DB is responsive by opening a
// read-only transaction. A healthy DB completes the view without error; a
// closed or corrupted DB returns an error.
// Implements [metaengine.HealthChecker] for Kubernetes-style liveness probes.
func (e *badgerEngine) HealthCheck(_ context.Context) error {
	return e.db.View(func(_ *badger.Txn) error {
		return nil
	})
}

func (e *badgerEngine) Close() error {
	if !e.ownsDB {
		return nil
	}

	return e.db.Close()
}

// Key encoding helpers — package-level aliases to keycodec functions. The
// local `sep` reference is kept for inline concatenation in seedSeqCounters.

var (
	mapKey             = keycodec.MapKey //art-dupl:accept keycodec alias across engine modules
	setKey             = keycodec.SetKey
	counterKey         = keycodec.CounterKey
	counterPrefix      = keycodec.CounterPrefix
	multimapKey        = keycodec.MultimapKey
	multimapPrefix     = keycodec.MultimapPrefix
	logKey             = keycodec.LogKey
	logPrefix          = keycodec.LogPrefix
	collectionPrefix   = keycodec.CollectionPrefix
	streamKey          = keycodec.StreamKey
	streamPrefix       = keycodec.StreamPrefix
	journalKey         = keycodec.JournalKey
	journalPrefix      = keycodec.JournalPrefix
	streamSeqMapKey    = keycodec.StreamSeqKey
	encodeJSON         = keycodec.EncodeJSON
	decodeJSON         = keycodec.DecodeJSON
	encodeKeyStr       = keycodec.EncodeKeyStr
	encodeCounterValue = keycodec.EncodeCounterValue
	decodeCounterValue = keycodec.DecodeCounterValue
)

// seedSeqCounters scans existing keys to seed log/multimap/journal/stream seq
// counters on restart, preventing key collisions.
func (e *badgerEngine) seedSeqCounters() error {
	if e.persistence == metaengine.PersistenceVolatile {
		return nil
	}

	return e.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte("l" + sep)
		opts.PrefetchValues = false

		iter := txn.NewIterator(opts)
		defer iter.Close()

		for iter.Rewind(); iter.Valid(); iter.Next() {
			keyStr := string(iter.Item().Key())
			parts := strings.SplitN(keyStr, sep, 3)
			if len(parts) < 3 {
				continue
			}

			col := parts[1]

			var seq int64
			_, _ = fmt.Sscanf(parts[2], "%020d", &seq)

			actual, _ := e.logSeq.LoadOrStore(col, &atomic.Int64{})
			existing := actual.(*atomic.Int64).Load()
			if seq > existing {
				actual.(*atomic.Int64).Store(seq)
			}
		}

		return nil
	})
}

// Compile-time assertions.
var (
	_ metaengine.Engine               = (*badgerEngine)(nil)
	_ metaengine.MapBackend           = (*badgerEngine)(nil)
	_ metaengine.MapUpdater           = (*badgerEngine)(nil)
	_ metaengine.ScanBackend          = (*badgerEngine)(nil)
	_ metaengine.SetBackend           = (*badgerEngine)(nil)
	_ metaengine.CounterBackend       = (*badgerEngine)(nil)
	_ metaengine.MultimapBackend      = (*badgerEngine)(nil)
	_ metaengine.LogBackend           = (*badgerEngine)(nil)
	_ metaengine.StreamLogBackend     = (*badgerEngine)(nil)
	_ metaengine.SeqSeekableStreamLog = (*badgerEngine)(nil)
	_ metaengine.AtomicAppender       = (*badgerEngine)(nil)
	_ metaengine.Calibratable         = (*badgerEngine)(nil)
	_ metaengine.TrackerHost          = (*badgerEngine)(nil)
)

var (
	_ = context.Background // suppress unused import when ctx is unused in backends
	_ = errors.Is          // suppress unused import if no errors.Is calls remain
)
