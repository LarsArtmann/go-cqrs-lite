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

	bolt "go.etcd.io/bbolt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/keycodec"
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

// BboltNsPerRead is the estimated per-READ-operation cost.
// B+tree point reads benefit from mmap page caching.
const BboltNsPerRead = 1500.0

// BboltNsPerWrite is the estimated per-WRITE-operation cost.
// Write transactions involve page allocation and an optional fsync.
const BboltNsPerWrite = 5000.0

var _ metaengine.TrackerHost = (*bboltEngine)(nil)

type bboltEngine struct {
	db          *bolt.DB
	ownsDB      bool
	tmpPath     string // non-empty when we created a temp file (volatile mode)
	done        bool
	persistence metaengine.Persistence
	mu          sync.Mutex // guards MapUpdate, StreamAppend, StreamAppendExpected
	logSeq      sync.Map   // collection → *atomic.Int64 (log sequence counter)
	mmSeq       sync.Map   // collection → *atomic.Int64 (multimap sequence counter)
	streamSeq   sync.Map   // "col\x00sid" → *atomic.Int64 (per-stream sequence)
	journalSeq  sync.Map   // collection → *atomic.Int64 (global journal sequence)
	metaengine.Calibration
}

// NewBboltEngine creates a bbolt-backed metaengine engine. If path is empty,
// a temporary file is used (for testing, volatile); otherwise path is the
// on-disk database file (persisted across opens). The caller owns the returned
// Engine and must call Close.
func NewBboltEngine(path string) (metaengine.Engine, error) {
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

	db, err := bolt.Open(path, 0o600, bolt.DefaultOptions)
	if err != nil {
		if tmpPath != "" {
			_ = os.Remove(tmpPath)
		}

		return nil, fmt.Errorf("bboltengine: open: %w", err)
	}

	eng := &bboltEngine{
		db:          db,
		ownsDB:      true,
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
		NsPerRead:   BboltNsPerRead,
		NsPerWrite:  BboltNsPerWrite,
		Persistence: e.persistence,
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap:       metaengine.ComplexityOLogN, // B+tree lookup
			metaengine.ADTSet:       metaengine.ComplexityOLogN,
			metaengine.ADTCounter:   metaengine.ComplexityON,    // CounterGet = prefix scan
			metaengine.ADTSortedMap: metaengine.ComplexityON,    // scan + Go sort
			metaengine.ADTLog:       metaengine.ComplexityOLogN, // append O(logN), tail O(N)
			metaengine.ADTMultimap:  metaengine.ComplexityOLogN,
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
	_ metaengine.Engine           = (*bboltEngine)(nil)
	_ metaengine.MapBackend       = (*bboltEngine)(nil)
	_ metaengine.MapUpdater       = (*bboltEngine)(nil)
	_ metaengine.ScanBackend      = (*bboltEngine)(nil)
	_ metaengine.SetBackend       = (*bboltEngine)(nil)
	_ metaengine.CounterBackend   = (*bboltEngine)(nil)
	_ metaengine.MultimapBackend  = (*bboltEngine)(nil)
	_ metaengine.LogBackend       = (*bboltEngine)(nil)
	_ metaengine.StreamLogBackend = (*bboltEngine)(nil)
	_ metaengine.AtomicAppender   = (*bboltEngine)(nil)
	_ metaengine.Calibratable     = (*bboltEngine)(nil)
)
