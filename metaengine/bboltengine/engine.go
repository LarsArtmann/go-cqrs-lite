// Package bboltengine implements a bbolt-backed metaengine Engine.
//
// bbolt (etcd's fork of BoltDB) is a pure-Go B+tree key-value store. It offers
// a genuinely different cost profile from Pebble (LSM) and SQLite (SQL):
//   - Map/Set point lookups: O(log N) B+tree traversal (mmap-backed, cache-friendly)
//   - Counter: O(log N) increment via atomic read-modify-write in one tx, O(N) CounterGet
//   - SortedMap scan: O(N) prefix scan + Go sort (degraded — no secondary indexes)
//   - Log: O(log N) append, O(N) tail (reverse scan not prefix-bounded)
//
// bbolt uses a single-writer model (serialized write transactions), which
// eliminates the need for external locking on atomic operations. All read-write
// transactions are serialized by bbolt internally.
//
// This module exists OUTSIDE the zero-dependency metaengine core (ADR-0062)
// because it requires the go.etcd.io/bbolt dependency.
package bboltengine

import (
	"context"
	"fmt"
	"os"
	"sync"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/keycodec"
	bolt "go.etcd.io/bbolt"
)

// bucketName is the single bbolt bucket holding all metaengine data.
// keycodec prefixes (m\x00, s\x00, c\x00, etc.) provide logical separation
// within the keyspace, matching the pebble/badger pattern.
const bucketName = "cqrs_meta"

// sep is the key separator. Null byte sorts before all printable characters,
// preventing collisions between collection names and keys.
const sep = "\x00"

// BboltNsPerOp is the estimated per-operation cost for the bbolt engine.
// bbolt uses B+tree storage with mmap — point reads are cache-friendly but
// writes require a full B+tree page update. These are initial estimates;
// use SetCalibration to override with measured values.
const BboltNsPerOp = 5000.0

// BboltNsPerRead is the measured per-READ-operation cost (B+tree point lookup
// + JSON decode, mmap page cache). It feeds ReadCosts.NsPerPointLookup in
// Profile. Re-calibrated 2026-08-30 via BenchmarkCalibration_BboltGet (~742
// ns/op median of 3; the prior 1500 estimate was ~2x conservative).
const BboltNsPerRead = 750.0

// BboltNsPerWrite is the estimated per-WRITE-operation cost.
// Write transactions involve page allocation and an optional fsync.
const BboltNsPerWrite = 5000.0

var _ metaengine.TrackerHost = (*bboltEngine)(nil)

// Option configures a bbolt engine at construction time.
type Option func(*engineConfig)

type engineConfig struct {
	noSync bool
}

// WithNoSync opens the database with bbolt's NoSync (and its companion
// NoFreelistSync): write transactions skip the commit fsync. bbolt upstream
// documents this as dangerous — data loss (and on unclean shutdown, possible
// corruption) can occur on crash. bbolt has no WAL, so unlike Pebble's async
// writes this is NOT app-crash-safe; the option is named after the bbolt knob
// it sets rather than "async writes" to avoid implying that equivalence.
func WithNoSync() Option {
	return func(cfg *engineConfig) { cfg.noSync = true }
}

// boltOptions translates the engine config to bbolt open options, copying
// bolt.DefaultOptions so the shared global is never mutated.
func boltOptions(cfg engineConfig) *bolt.Options {
	opts := *bolt.DefaultOptions
	if cfg.noSync {
		opts.NoSync = true
		opts.NoFreelistSync = true
	}

	return &opts
}

type bboltEngine struct {
	metaengine.Calibration

	db          *bolt.DB
	ownsDB      bool
	noSync      bool
	tmpPath     string // non-empty when we created a temp file (volatile mode)
	done        bool
	persistence metaengine.Persistence
	mu          sync.Mutex // guards MapUpdate, StreamAppend, StreamAppendExpected
	logSeq      sync.Map   // collection → *atomic.Int64 (log sequence counter)
	mmSeq       sync.Map   // collection → *atomic.Int64 (multimap sequence counter)
	streamSeq   sync.Map   // "col\x00sid" → *atomic.Int64 (per-stream sequence)
	journalSeq  sync.Map   // collection → *atomic.Int64 (global journal sequence)
}

// NewBboltEngine creates a bbolt-backed metaengine engine. If path is empty,
// a temporary file is used (for testing, volatile); otherwise path is the
// on-disk database file (persisted across opens). The caller owns the returned
// Engine and must call Close.
func NewBboltEngine(path string, opts ...Option) (metaengine.Engine, error) {
	cfg := engineConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	persistence := metaengine.PersistencePersistent
	tmpPath := ""

	if path == "" {
		tmpFile, err := os.CreateTemp("", "bbolt-metaengine-*.db")
		if err != nil {
			return nil, fmt.Errorf("bboltengine: create temp file: %w", err)
		}

		path = tmpFile.Name()
		_ = tmpFile.Close()
		tmpPath = path
		persistence = metaengine.PersistenceVolatile
	}

	db, err := bolt.Open(path, 0o600, boltOptions(cfg))
	if err != nil {
		if tmpPath != "" {
			_ = os.Remove(tmpPath)
		}

		return nil, fmt.Errorf("bboltengine: open: %w", err)
	}

	eng := &bboltEngine{
		db:          db,
		ownsDB:      true,
		noSync:      cfg.noSync,
		tmpPath:     tmpPath,
		persistence: persistence,
	}

	if err := eng.init(); err != nil {
		_ = db.Close()

		if tmpPath != "" {
			_ = os.Remove(tmpPath)
		}

		return nil, err
	}

	if persistence == metaengine.PersistencePersistent {
		if err := eng.seedSeqCounters(); err != nil {
			_ = db.Close()

			return nil, fmt.Errorf("bboltengine: seed seq counters: %w", err)
		}
	}

	return eng, nil
}

// NewBboltEngineFromDB wraps an existing *bolt.DB. The caller retains
// ownership of the DB — Close on the engine is a no-op.
func NewBboltEngineFromDB(db *bolt.DB) (metaengine.Engine, error) {
	eng := &bboltEngine{
		db:          db,
		ownsDB:      false,
		persistence: metaengine.PersistencePersistent,
	}

	if err := eng.init(); err != nil {
		return nil, err
	}

	if err := eng.seedSeqCounters(); err != nil {
		return nil, fmt.Errorf("bboltengine: seed seq counters: %w", err)
	}

	return eng, nil
}

func (e *bboltEngine) init() error {
	return e.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		return err //nolint:wrapcheck // bbolt error is self-describing
	})
}

func (e *bboltEngine) Profile() metaengine.EngineProfile {
	p := metaengine.EngineProfile{
		Name:        "bbolt",
		NsPerOp:     BboltNsPerOp,
		NsPerWrite:  BboltNsPerWrite,
		Persistence: e.persistence,
		// Per-read-pattern calibrated costs (see calibration_bench_test.go;
		// measured 2026-08-30, medians of 3 runs on 32 threads, load ~3.8).
		// bbolt is a KV engine: filtered scans and aggregations have no SQL
		// pushdown, so they degrade to a full scan with Go-side work — hence
		// per-row scan costs (~620-660ns) far above the point-lookup cost.
		ReadCosts: metaengine.ReadCosts{
			// B+tree point lookup (BenchmarkCalibration_BboltGet).
			NsPerPointLookup: BboltNsPerRead,
			// ~614 ns/row (BenchmarkCalibration_Bbolt_FilteredScan): full
			// MapScan + Go-side predicate over 10K rows, ~50% match.
			NsPerFilteredScan: 620,
			// ~98 ns/row (BenchmarkCalibration_Bbolt_CounterScan): CounterGet
			// prefix scan over 1K counters — the ReadAggregate path.
			NsPerAggregate: 100,
			// ~656 ns/row (BenchmarkCalibration_Bbolt_FullScan): full MapScan,
			// JSON decode of every row.
			NsPerScan: 660,
		},
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap:       metaengine.ComplexityOLogN, // B+tree lookup
			metaengine.ADTSet:       metaengine.ComplexityOLogN,
			metaengine.ADTCounter:   metaengine.ComplexityON,    // CounterGet = prefix scan
			metaengine.ADTSortedMap: metaengine.ComplexityON,    // scan + Go sort
			metaengine.ADTLog:       metaengine.ComplexityOLogN, // append O(logN), tail O(N)
			metaengine.ADTMultimap:  metaengine.ComplexityOLogN,
			metaengine.ADTVector:    metaengine.ComplexityON, // brute-force scan (degraded)
		},
		DegradedADTs: map[metaengine.ADT]bool{
			metaengine.ADTVector: true, // O(N·D) brute-force, no ANN index
		},
		Layouts: map[metaengine.ADT]metaengine.StorageLayout{
			metaengine.ADTMap:       metaengine.LayoutLSM,
			metaengine.ADTSet:       metaengine.LayoutLSM,
			metaengine.ADTCounter:   metaengine.LayoutLSM,
			metaengine.ADTSortedMap: metaengine.LayoutLSM,
			metaengine.ADTLog:       metaengine.LayoutLSM,
			metaengine.ADTMultimap:  metaengine.LayoutLSM,
		},
	}
	e.ApplyCalibration(&p)

	return p
}

func (e *bboltEngine) Close() error {
	if !e.ownsDB {
		return nil
	}

	if e.done {
		return nil
	}

	e.done = true

	err := e.db.Close()

	if e.tmpPath != "" {
		_ = os.Remove(e.tmpPath)
	}

	return err //nolint:wrapcheck // bbolt error is self-describing
}

// HealthCheck verifies the underlying bbolt DB is responsive by opening a
// read-only transaction. A healthy DB completes the view without error.
// Implements [metaengine.HealthChecker] for Kubernetes-style liveness probes.
func (e *bboltEngine) HealthCheck(_ context.Context) error {
	return e.db.View(func(_ *bolt.Tx) error {
		return nil
	})
}

// Key encoding helpers — package-level aliases to keycodec functions.
//
// var.
var (
	mapKey             = keycodec.MapKey //art-dupl:accept keycodec alias across engine modules
	setKey             = keycodec.SetKey //nolint:unused // used in backends.go
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

// Compile-time assertions.
var (
	_ metaengine.Engine               = (*bboltEngine)(nil)
	_ metaengine.MapBackend           = (*bboltEngine)(nil)
	_ metaengine.MapUpdater           = (*bboltEngine)(nil)
	_ metaengine.ScanBackend          = (*bboltEngine)(nil)
	_ metaengine.SetBackend           = (*bboltEngine)(nil)
	_ metaengine.CounterBackend       = (*bboltEngine)(nil)
	_ metaengine.MultimapBackend      = (*bboltEngine)(nil)
	_ metaengine.LogBackend           = (*bboltEngine)(nil)
	_ metaengine.StreamLogBackend     = (*bboltEngine)(nil)
	_ metaengine.SeqSeekableStreamLog = (*bboltEngine)(nil)
	_ metaengine.AtomicAppender       = (*bboltEngine)(nil)
	_ metaengine.Calibratable         = (*bboltEngine)(nil)
	_ metaengine.HealthChecker        = (*bboltEngine)(nil)
	_ metaengine.StreamingScan        = (*bboltEngine)(nil)
)
