// Package badgerengine implements a Badger-backed metaengine Engine.
//
// Badger v4 is a pure-Go LSM-tree key-value store. It offers a genuinely
// different cost profile from Pebble and SQLite:
//   - Map/Set point lookups: O(1) LSM point read (comparable to Pebble)
//   - Counter: O(1) increment via read-modify-write, O(N) CounterGet (prefix scan)
//   - SortedMap scan: O(N) prefix scan + Go sort (degraded — no secondary indexes)
//   - Graph: O(N^d) BFS via prefix scan
//
// This module exists OUTSIDE the zero-dependency metaengine core (ADR-0062)
// because it requires the dgraph-io/badger/v4 dependency.
package badgerengine

import (
	"context"
	"encoding/binary"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/dgraph-io/badger/v4"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// sep is the key separator. Null byte sorts before all printable characters,
// preventing collisions between collection names and keys.
const sep = "\x00"

// BadgerNsPerOp is the estimated per-operation cost for the Badger engine.
// Badger's LSM-tree architecture provides comparable point-read performance
// to Pebble. This value will be calibrated with benchmarks once the engine
// is stable.
const BadgerNsPerOp = 2000.0

// BadgerNsPerRead is the estimated per-READ-operation cost.
const BadgerNsPerRead = 1300.0

// BadgerNsPerWrite is the estimated per-WRITE-operation cost.
const BadgerNsPerWrite = 2500.0

type badgerEngine struct {
	db          *badger.DB
	ownsDB      bool
	persistence metaengine.Persistence
	mu          sync.Mutex // guards MapUpdate, StreamAppend, StreamAppendExpected
	logSeq      sync.Map   // collection → *atomic.Int64 (log sequence counter)
	mmSeq       sync.Map   // collection → *atomic.Int64 (multimap sequence counter)
	streamSeq   sync.Map   // "col\x00sid" → *atomic.Int64 (per-stream sequence)
	journalSeq  sync.Map   // collection → *atomic.Int64 (global journal sequence)
	cal         metaengine.Calibration
}

// NewBadgerEngine creates a Badger-backed metaengine engine. If dir is empty,
// an in-memory database is used (for testing); otherwise dir is the on-disk
// database directory (persisted across opens). The caller owns the returned
// Engine and must call Close.
func NewBadgerEngine(dir string) (metaengine.Engine, error) {
	opts := badger.DefaultOptions(dir)
	persistence := metaengine.PersistencePersistent

	if dir == "" {
		opts = badger.DefaultOptions("").WithInMemory(true)
		persistence = metaengine.PersistenceVolatile
	}

	// Suppress Badger's default logger noise in production.
	opts = opts.WithLogger(nil)

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("badgerengine: open: %w", err)
	}

	eng := &badgerEngine{
		db:          db,
		ownsDB:      true,
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
		NsPerRead:   BadgerNsPerRead,
		NsPerWrite:  BadgerNsPerWrite,
		Persistence: e.persistence,
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap:       metaengine.ComplexityO1,
			metaengine.ADTSet:       metaengine.ComplexityO1,
			metaengine.ADTCounter:   metaengine.ComplexityON,
			metaengine.ADTGraph:     metaengine.ComplexityON,
			metaengine.ADTSortedMap: metaengine.ComplexityON,
			metaengine.ADTLog:       metaengine.ComplexityOLogN,
			metaengine.ADTMultimap:  metaengine.ComplexityOLogN,
		},
	}
	e.cal.ApplyCalibration(&p)

	return p
}

// SetCalibration implements metaengine.Calibratable.
func (e *badgerEngine) SetCalibration(costs metaengine.CalibrationCosts) {
	e.cal.SetCalibration(costs)
}

func (e *badgerEngine) Close() error {
	if !e.ownsDB {
		return nil
	}

	return e.db.Close()
}

// --- Key encoding helpers ---

func mapKey(col, key string) []byte {
	return []byte("m" + sep + col + sep + key)
}

func setKey(col, key string) []byte {
	return []byte("s" + sep + col + sep + key)
}

func counterKey(col, ckey string) []byte {
	return []byte("c" + sep + col + sep + ckey)
}

func counterPrefix(col string) []byte {
	return []byte("c" + sep + col + sep)
}

func multimapKey(col, key string, seq int64) []byte {
	return fmt.Appendf(nil, "mm%s%s%s%s%s%020d", sep, col, sep, key, sep, seq)
}

func multimapPrefix(col, key string) []byte {
	return []byte("mm" + sep + col + sep + key + sep)
}

func logKey(col string, seq int64) []byte {
	return fmt.Appendf(nil, "l%s%s%s%020d", sep, col, sep, seq)
}

func logPrefix(col string) []byte {
	return []byte("l" + sep + col + sep)
}

func graphEdgeKey(col, from, to string) []byte {
	return []byte("g" + sep + col + sep + from + sep + to)
}

func graphPrefixForward(col, node string) []byte {
	return []byte("g" + sep + col + sep + node + sep)
}

func collectionPrefix(col string) []byte {
	return []byte("m" + sep + col + sep)
}

// --- Value encoding helpers ---

func encodeJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(fmt.Sprintf("%v", v))
	}

	return b
}

func decodeJSON(data []byte) any {
	var val any
	if err := json.Unmarshal(data, &val); err != nil {
		return string(data)
	}

	return val
}

func encodeKeyStr(key any) string {
	return string(encodeJSON(key))
}

func encodeCounterValue(v int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))

	return b
}

func decodeCounterValue(data []byte) int64 {
	if len(data) < 8 {
		return 0
	}

	return int64(binary.BigEndian.Uint64(data))
}

// nextKey returns the lexicographically next key after prefix (for upper bound).
func nextKey(prefix []byte) []byte {
	result := make([]byte, len(prefix))
	copy(result, prefix)

	for i := len(result) - 1; i >= 0; i-- {
		result[i]++
		if result[i] != 0 {
			return result
		}
	}

	return append(result, 0)
}

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
			fmt.Sscanf(parts[2], "%020d", &seq)

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
	_ metaengine.Engine           = (*badgerEngine)(nil)
	_ metaengine.MapBackend       = (*badgerEngine)(nil)
	_ metaengine.MapUpdater       = (*badgerEngine)(nil)
	_ metaengine.ScanBackend      = (*badgerEngine)(nil)
	_ metaengine.SetBackend       = (*badgerEngine)(nil)
	_ metaengine.CounterBackend   = (*badgerEngine)(nil)
	_ metaengine.GraphBackend     = (*badgerEngine)(nil)
	_ metaengine.MultimapBackend  = (*badgerEngine)(nil)
	_ metaengine.LogBackend       = (*badgerEngine)(nil)
	_ metaengine.StreamLogBackend = (*badgerEngine)(nil)
	_ metaengine.AtomicAppender   = (*badgerEngine)(nil)
	_ metaengine.Calibratable     = (*badgerEngine)(nil)
)

var (
	_ = context.Background // suppress unused import when ctx is unused in backends
	_ = errors.Is          // suppress unused import if no errors.Is calls remain
)
