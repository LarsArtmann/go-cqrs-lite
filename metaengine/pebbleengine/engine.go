// Package pebbleengine implements a Pebble-backed metaengine Engine.
//
// Pebble provides a genuinely different cost profile from SQLite and Memory:
//
//   - Map/Set point lookups: O(1) LSM point read (faster than SQLite's O(logN) B-tree)
//
//   - Counter: O(1) increment, O(N) CounterGet (prefix scan — degraded)
//
//   - SortedMap scan: O(N) prefix scan + Go sort (degraded — no secondary indexes)
//
//     Graph: NOT supported — this engine has no graph dispatch. Graph workloads
//     route to a graph-capable engine (dgraphengine, or graphadapter over any
//     engine); the profile deliberately omits ADTGraph.
//
// This module is a dep-isolated engine: it lives OUTSIDE the metaengine
// module because it requires the cockroachdb/pebble dependency, keeping the
// planner core's dependency budget clean (ADR-0062, as amended by the
// ADR-0046/0111 tier model).
package pebbleengine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/keycodec"
)

// sep is the key separator. Null byte sorts before all printable characters,
// preventing collisions between collection names and keys.
const sep = "\x00"

// PebbleNsPerOp is the calibrated per-operation cost for the Pebble engine.
// Re-measured 2026-08-03 on AMD Ryzen AI MAX+ 395 (in-memory vfs.NewMem)
// with correctness assertions enabled:
//   - MapSet (LSM insert + JSON encode): ~2,526 ns/op
//   - MapGet (LSM point read + JSON decode): ~1,328 ns/op
//
// The value 2,000 ns is biased toward the write path (fold-heavy workloads
// dominate). Pebble is ~2.5x faster than SQLite on point reads and ~2.5x faster
// on writes.
const PebbleNsPerOp = 2000.0

// PebbleNsPerRead is the calibrated per-READ-operation cost (LSM point read +
// JSON decode). It feeds ReadCosts.NsPerPointLookup in Profile.
// Re-calibrated 2026-08-30 via BenchmarkCalibration_PebbleGet (~684 ns/op
// median of 3; the 2026-08-03 measurement was ~1328 on a different value
// shape — the int-payload Get path is cheaper than first measured).
const PebbleNsPerRead = 700.0

// PebbleNsPerWrite is the calibrated per-WRITE-operation cost (LSM insert +
// JSON encode). Measured 2026-08-03 via BenchmarkCalibration_PebbleSet (~2,526 ns/op).
// Used by the planner's write-cost path (EngineProfile.WriteNsPerOp).
const PebbleNsPerWrite = 2500.0

var _ metaengine.TrackerHost = (*pebbleEngine)(nil)

// Option configures a Pebble engine at construction time.
type Option func(*engineConfig)

type engineConfig struct {
	syncWrites bool
	disableWAL bool
}

// WithAsyncWrites skips the per-write fsync: each write is appended to the
// write-ahead log in the page cache but returns before the fsync. Writes
// survive an application crash; a kernel or power crash may lose the most
// recent ones. The default engine fsyncs every write (pebble.Sync).
func WithAsyncWrites() Option {
	return func(cfg *engineConfig) { cfg.syncWrites = false }
}

// WithDisableWAL disables Pebble's write-ahead log: writes land in the
// memtable only and reach disk via flushes. Data may be lost on any crash.
// Only meaningful with [WithAsyncWrites] — with the WAL disabled a sync
// write degrades to a memtable flush, the slowest path Pebble has.
//
// Only [NewPebbleEngine] can apply it: [NewPebbleEngineFromDB] receives a
// database the caller already opened, so the option is ignored there.
func WithDisableWAL() Option {
	return func(cfg *engineConfig) { cfg.disableWAL = true }
}

type pebbleEngine struct {
	metaengine.Calibration

	db          *pebble.DB
	ownsDB      bool
	syncWrites  bool
	persistence metaengine.Persistence
	mu          sync.Mutex // guards counter/multimap/log seq operations
	logSeq      sync.Map   // collection → *atomic.Int64 (log sequence counter)
	mmSeq       sync.Map   // collection → *atomic.Int64 (multimap sequence counter)
	streamSeq   sync.Map   // "col\x00sid" → *atomic.Int64 (per-stream sequence)
	journalSeq  sync.Map   // collection → *atomic.Int64 (global journal sequence)
	layoutMu    sync.Mutex
	layouts     map[string]layoutPlan // collection → layout plan (secondary indexes)
}

// writeOptions returns pebble.Sync when sync writes are enabled, otherwise
// nil (pebble's default asynchronous write semantics).
func (e *pebbleEngine) writeOptions() *pebble.WriteOptions {
	if e.syncWrites {
		return pebble.Sync
	}

	return nil
}

// NewPebbleEngine creates a Pebble-backed metaengine engine. If dir is empty,
// an in-memory database is used (for testing); otherwise dir is the on-disk
// database directory (persisted across opens). The caller owns the returned
// Engine and must call Close.
func NewPebbleEngine(dir string, opts ...Option) (metaengine.Engine, error) {
	cfg := engineConfig{syncWrites: true}
	for _, opt := range opts {
		opt(&cfg)
	}

	dbOpts := &pebble.Options{DisableWAL: cfg.disableWAL}
	persistence := metaengine.PersistencePersistent
	if dir == "" {
		dbOpts.FS = vfs.NewMem()
		persistence = metaengine.PersistenceVolatile
	}

	db, err := pebble.Open(dir, dbOpts)
	if err != nil {
		return nil, fmt.Errorf("pebbleengine: open: %w", err)
	}

	eng := &pebbleEngine{
		db:          db,
		ownsDB:      true,
		syncWrites:  cfg.syncWrites,
		persistence: persistence,
	}

	// Seed seq counters from existing data on persistent DBs to prevent
	// key collisions on restart (all counters would start from 0 otherwise).
	if persistence == metaengine.PersistencePersistent {
		if err := eng.seedSeqCounters(); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("pebbleengine: seed seq counters: %w", err)
		}
	}

	return eng, nil
}

// NewPebbleEngineFromDB wraps an existing *pebble.DB. The caller retains
// ownership of the DB — Close on the engine is a no-op.
func NewPebbleEngineFromDB(db *pebble.DB, opts ...Option) (metaengine.Engine, error) {
	cfg := engineConfig{syncWrites: true}
	for _, opt := range opts {
		opt(&cfg)
	}

	eng := &pebbleEngine{
		db:          db,
		ownsDB:      false,
		syncWrites:  cfg.syncWrites,
		persistence: metaengine.PersistencePersistent,
	}

	// Seed seq counters from existing data to prevent key collisions on restart.
	if err := eng.seedSeqCounters(); err != nil {
		return nil, fmt.Errorf("pebbleengine: seed seq counters: %w", err)
	}

	return eng, nil
}

func (e *pebbleEngine) Profile() metaengine.EngineProfile {
	p := metaengine.EngineProfile{
		Name:        "pebble",
		NsPerOp:     PebbleNsPerOp,
		NsPerWrite:  PebbleNsPerWrite,
		Persistence: e.persistence,
		// Per-read-pattern calibrated costs (see calibration_bench_test.go;
		// measured 2026-08-30, medians of 3 runs on 32 threads, load ~3.8).
		// Pebble is a KV engine: filtered scans and aggregations have no SQL
		// pushdown, so they degrade to ScanRawValues with Go-side work — hence
		// per-row scan costs (~700-830ns) far above the point-lookup cost.
		ReadCosts: metaengine.ReadCosts{
			// Indexed LSM point lookup (BenchmarkCalibration_PebbleGet).
			NsPerPointLookup: PebbleNsPerRead,
			// ~833 ns/row (BenchmarkCalibration_Pebble_FilteredScan):
			// ScanRawValues + Go-side PassesFilterSpecs over 10K rows, ~50% match.
			NsPerFilteredScan: 830,
			// ~125 ns/row (BenchmarkCalibration_Pebble_CounterScan): CounterGet
			// prefix scan over 1K counters — the ReadAggregate path.
			NsPerAggregate: 125,
			// ~695 ns/row (BenchmarkCalibration_Pebble_FullScan): full
			// ScanRawValues, JSON decode of every row.
			NsPerScan: 700,
		},
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap:       metaengine.ComplexityO1, // LSM point read
			metaengine.ADTSet:       metaengine.ComplexityO1,
			metaengine.ADTCounter:   metaengine.ComplexityON, // CounterGet = prefix scan
			metaengine.ADTSortedMap: metaengine.ComplexityON, // O(limit) with sort index, O(N) fallback
			metaengine.ADTLog:       metaengine.ComplexityOLogN,
			metaengine.ADTMultimap:  metaengine.ComplexityOLogN,
			metaengine.ADTVector:    metaengine.ComplexityON,
			metaengine.ADTSearch:    metaengine.ComplexityON,
			metaengine.ADTSpatial:   metaengine.ComplexityON,
		},
		DegradedADTs: map[metaengine.ADT]bool{
			metaengine.ADTVector:  true,
			metaengine.ADTSearch:  true,
			metaengine.ADTSpatial: true,
		},
		Layouts: map[metaengine.ADT]metaengine.StorageLayout{
			metaengine.ADTMap:       metaengine.LayoutLSM,
			metaengine.ADTSet:       metaengine.LayoutLSM,
			metaengine.ADTCounter:   metaengine.LayoutLSM,
			metaengine.ADTSortedMap: metaengine.LayoutLSM,
		},
	}
	e.ApplyCalibration(&p)

	return p
}

func (e *pebbleEngine) Close() error {
	if !e.ownsDB {
		return nil
	}

	return e.db.Close()
}

// HealthCheck verifies the underlying Pebble DB is responsive by attempting a
// point read of a non-existent key. A healthy DB returns pebble.ErrNotFound;
// any other error (e.g., "database closed") indicates a problem. Pebble panics
// on use-after-close, so we recover and return as an error.
// Implements [metaengine.HealthChecker] for Kubernetes-style liveness probes.
func (e *pebbleEngine) HealthCheck(_ context.Context) (retErr error) {
	defer func() {
		if r := recover(); r != nil {
			retErr = fmt.Errorf("pebble health check: %v", r)
		}
	}()

	_, closer, err := e.db.Get([]byte("__health_check__"))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil
		}

		return err
	}

	_ = closer.Close()

	return nil
}

// Key encoding helpers — package-level aliases to keycodec functions. The
// local `sep` reference is kept for inline concatenation in seedSeqCounters.

var (
	mapKey             = keycodec.MapKey
	setKey             = keycodec.SetKey
	counterKey         = keycodec.CounterKey
	multimapKey        = keycodec.MultimapKey
	multimapPrefix     = keycodec.MultimapPrefix
	logKey             = keycodec.LogKey
	logPrefix          = keycodec.LogPrefix
	collectionPrefix   = keycodec.CollectionPrefix
	streamKey          = keycodec.StreamKey
	streamPrefix       = keycodec.StreamPrefix
	journalKey         = keycodec.JournalKey
	journalPrefix      = keycodec.JournalPrefix
	streamSeqKey       = keycodec.StreamSeqKey
	encodeJSON         = keycodec.EncodeJSON
	decodeJSON         = keycodec.DecodeJSON
	encodeKeyStr       = keycodec.EncodeKeyStr
	encodeCounterValue = keycodec.EncodeCounterValue
	decodeCounterValue = keycodec.DecodeCounterValue
)

// Compile-time assertions.
var (
	_ metaengine.Engine          = (*pebbleEngine)(nil)
	_ metaengine.MapBackend      = (*pebbleEngine)(nil)
	_ metaengine.MapUpdater      = (*pebbleEngine)(nil)
	_ metaengine.ScanBackend     = (*pebbleEngine)(nil)
	_ metaengine.SetBackend      = (*pebbleEngine)(nil)
	_ metaengine.CounterBackend  = (*pebbleEngine)(nil)
	_ metaengine.MultimapBackend = (*pebbleEngine)(nil)
	_ metaengine.LogBackend      = (*pebbleEngine)(nil)
	_ metaengine.StreamingScan   = (*pebbleEngine)(nil)
)

// --- MapBackend ---

func (e *pebbleEngine) MapSet(_ context.Context, col string, key, value any) error {
	keyStr := encodeKeyStr(key)
	valueJSON := encodeJSON(value)

	// Write secondary index entries if a layout plan exists.
	e.layoutMu.Lock()
	plan, hasLayout := e.layouts[col]
	e.layoutMu.Unlock()

	if hasLayout {
		batch := e.db.NewBatch()
		defer metaengine.DeferClose(batch)

		// Delete old index entries if the key already exists.
		if oldVal, closer, err := e.db.Get(mapKey(col, keyStr)); err == nil {
			e.deleteIndexEntries(batch, col, keyStr, oldVal, plan)

			_ = closer.Close()
		}

		if err := batch.Set(mapKey(col, keyStr), valueJSON, nil); err != nil {
			return fmt.Errorf("pebbleengine: MapSet: %w", err)
		}

		if err := e.writeIndexEntries(batch, col, keyStr, valueJSON, plan); err != nil {
			return err
		}

		return batch.Commit(e.writeOptions())
	}

	return e.db.Set(mapKey(col, keyStr), valueJSON, e.writeOptions())
}

func (e *pebbleEngine) MapGet(_ context.Context, col string, key any) (any, bool, error) {
	val, ok, err := e.getPebbleRaw(col, key)
	if err != nil || !ok {
		return nil, ok, err
	}

	return decodeJSON(val), true, nil
}

func (e *pebbleEngine) MapDelete(_ context.Context, col string, key any) error {
	keyStr := encodeKeyStr(key)

	// Clean up secondary index entries if a layout plan exists.
	e.layoutMu.Lock()
	plan, hasLayout := e.layouts[col]
	e.layoutMu.Unlock()

	if hasLayout {
		batch := e.db.NewBatch()
		defer metaengine.DeferClose(batch)

		if oldVal, closer, err := e.db.Get(mapKey(col, keyStr)); err == nil {
			e.deleteIndexEntries(batch, col, keyStr, oldVal, plan)

			_ = closer.Close()
		}

		if err := batch.Delete(mapKey(col, keyStr), nil); err != nil {
			return fmt.Errorf("pebbleengine: MapDelete: %w", err)
		}

		return batch.Commit(e.writeOptions())
	}

	return e.db.Delete(mapKey(col, keyStr), e.writeOptions())
}

// --- MapUpdater ---

func (e *pebbleEngine) MapUpdate(
	_ context.Context,
	col string,
	key any,
	update func(prev any) any,
) error {
	k := mapKey(col, encodeKeyStr(key))

	// The read-modify-write must be atomic: concurrent MapUpdate calls on the
	// same key would otherwise each read the same prev value and the last writer
	// wins, silently dropping updates. This mirrors the SQLite engine's
	// tx-atomic MapUpdate guarantee (ADR-0066). The engine-wide mutex is
	// sufficient because MapUpdate is a leaf operation.
	e.mu.Lock()
	defer e.mu.Unlock()

	var (
		prev       any
		oldValJSON []byte
	)

	val, closer, err := e.db.Get(k)
	if err == nil {
		prev = decodeJSON(val)
		oldValJSON = append([]byte(nil), val...)
		//cqrs-lint:ignore(C015,C023) library code or intentional pattern
		_ = closer.Close()
	} else if !errors.Is(err, pebble.ErrNotFound) {
		return err
	}

	newVal := update(prev)
	newValJSON := encodeJSON(newVal)

	// Update secondary index entries if a layout plan exists.
	e.layoutMu.Lock()
	plan, hasLayout := e.layouts[col]
	e.layoutMu.Unlock()

	if hasLayout {
		keyStr := encodeKeyStr(key)
		batch := e.db.NewBatch()

		defer metaengine.DeferClose(batch)

		e.deleteIndexEntries(batch, col, keyStr, oldValJSON, plan)

		if err := batch.Set(k, newValJSON, nil); err != nil {
			return fmt.Errorf("pebbleengine: MapUpdate: %w", err)
		}

		if err := e.writeIndexEntries(batch, col, keyStr, newValJSON, plan); err != nil {
			return err
		}

		return batch.Commit(e.writeOptions())
	}

	return e.db.Set(k, newValJSON, e.writeOptions())
}

// --- ScanBackend ---

func (e *pebbleEngine) MapScan(
	_ context.Context,
	col string,
	filterFn func(item any) bool,
	sortFunc func(a, b any) int,
	cursor any,
	limit int,
) (metaengine.ScanResult, error) {
	prefix := collectionPrefix(col)

	upperBound := nextKey(prefix)

	iter, err := e.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound,
	})
	if err != nil {
		return metaengine.ScanResult{}, err
	}

	defer metaengine.DeferClose(iter)

	var pairs []kvPair

	for iter.First(); iter.Valid(); iter.Next() {
		val := decodeJSON(iter.Value())

		if filterFn != nil && !filterFn(val) {
			continue
		}

		pairs = append(pairs, kvPair{
			key:   append([]byte(nil), iter.Key()...),
			value: val,
		})
	}

	if err := iter.Error(); err != nil {
		return metaengine.ScanResult{}, err
	}

	pairs = sortAndPaginate(pairs, sortFunc, cursor, limit)

	hasMore := limit > 0 && len(pairs) > limit
	if hasMore {
		pairs = pairs[:limit]
	}

	results := make([]any, len(pairs))
	for i, p := range pairs {
		results[i] = p.value
	}

	return metaengine.ScanResult{Items: results, HasMore: hasMore}, nil
}

// --- SetBackend ---

func (e *pebbleEngine) SetAdd(_ context.Context, col string, key any) error {
	return e.db.Set(setKey(col, encodeKeyStr(key)), nil, e.writeOptions())
}

func (e *pebbleEngine) SetContains(_ context.Context, col string, key any) (bool, error) {
	_, closer, err := e.db.Get(setKey(col, encodeKeyStr(key)))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return false, nil
		}

		return false, err
	}

	defer metaengine.DeferClose(closer)

	return true, nil
}

// --- CounterBackend ---

func (e *pebbleEngine) CounterIncrement(
	_ context.Context,
	col string,
	deltas metaengine.Delta,
) error {
	for k, d := range deltas {
		ck := counterKey(col, k)

		var current int64

		val, closer, err := e.db.Get(ck)
		if err == nil {
			current = decodeCounterValue(val)
			//cqrs-lint:ignore(C023) library code or intentional pattern
			_ = closer.Close() //cqrs-lint:ignore(C015) pebble closer, error is always nil
		} else if !errors.Is(err, pebble.ErrNotFound) {
			return err
		}

		if err := e.db.Set(ck, encodeCounterValue(current+d), e.writeOptions()); err != nil {
			return err
		}
	}

	return nil
}

func (e *pebbleEngine) CounterGet(_ context.Context, col string) (map[string]int64, error) {
	// Prefix scan all counters for this collection.
	prefix := []byte("c" + sep + col + sep)
	upperBound := nextKey(prefix)

	iter, err := e.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound,
	})
	if err != nil {
		return nil, err
	}

	defer metaengine.DeferClose(iter)

	result := make(map[string]int64)

	for iter.First(); iter.Valid(); iter.Next() {
		// Extract counter key from the full key: "c\x00{col}\x00{counterKey}".
		keyStr := string(iter.Key())

		parts := strings.SplitN(keyStr, sep, 3)
		if len(parts) < 3 {
			continue
		}

		result[parts[2]] = decodeCounterValue(iter.Value())
	}

	return result, iter.Error()
}

// --- MultimapBackend ---

func (e *pebbleEngine) MultiAdd(_ context.Context, col string, key, value any) error {
	seq := e.nextMmSeq(col)
	k := multimapKey(col, encodeKeyStr(key), seq)

	return e.db.Set(k, encodeJSON(value), e.writeOptions())
}

// iterJSON scans the given prefix range and decodes every value as JSON. The
// callback yields decoded values one at a time; iteration stops on the first
// decode error. Returns the iterator's terminal error (if any).
func (e *pebbleEngine) iterJSON(
	prefix, upperBound []byte,
	yield func(any),
) error {
	iter, err := e.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound,
	})
	if err != nil {
		return err
	}

	defer metaengine.DeferClose(iter)

	for iter.First(); iter.Valid(); iter.Next() {
		yield(decodeJSON(iter.Value()))
	}

	return iter.Error()
}

func (e *pebbleEngine) MultiGet(_ context.Context, col string, key any) ([]any, error) {
	prefix := multimapPrefix(col, encodeKeyStr(key))
	upperBound := nextKey(prefix)

	var out []any

	err := e.iterJSON(prefix, upperBound, func(v any) {
		out = append(out, v)
	})

	return out, err
}

func (e *pebbleEngine) nextMmSeq(col string) int64 {
	actual, _ := e.mmSeq.LoadOrStore(col, &atomic.Int64{})

	return actual.(*atomic.Int64).Add(1)
}

// --- LogBackend ---

func (e *pebbleEngine) LogAppend(_ context.Context, col string, value any) error {
	seq := e.nextLogSeq(col)
	k := logKey(col, seq)

	return e.db.Set(k, encodeJSON(value), e.writeOptions())
}

func (e *pebbleEngine) LogTail(_ context.Context, col string, limit int) ([]any, error) {
	prefix := logPrefix(col)
	upperBound := nextKey(prefix)

	if limit <= 0 {
		// No limit: stream everything in forward order via the shared JSON iterator.
		var entries []any

		err := e.iterJSON(prefix, upperBound, func(v any) {
			entries = append(entries, v)
		})

		return entries, err
	}

	// Reverse iteration for tail — needs Prev(), not First/Next, so the shared
	// helper does not apply. Take at most `limit` entries then reverse them
	// into chronological order.
	iter, err := e.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound,
	})
	if err != nil {
		return nil, err
	}

	defer metaengine.DeferClose(iter)

	var entries []any

	count := 0

	for iter.Last(); iter.Valid() && count < limit; iter.Prev() {
		entries = append(entries, decodeJSON(iter.Value()))
		count++
	}

	if err := iter.Error(); err != nil {
		return nil, err
	}

	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	return entries, nil
}

func (e *pebbleEngine) nextLogSeq(col string) int64 {
	actual, _ := e.logSeq.LoadOrStore(col, &atomic.Int64{})

	return actual.(*atomic.Int64).Add(1)
}

// --- Helpers ---

// nextKey returns the lexicographically next key after prefix (for upper bound).
func nextKey(prefix []byte) []byte {
	result := make([]byte, len(prefix))
	copy(result, prefix)

	// Iterate from the last byte backward, incrementing in place. Direct index
	// access is required: ranging over slices.Backward yields element COPIES, so
	// `v++` would modify the copy and leave `result` unchanged (the upper bound
	// would then equal the lower bound and every prefix scan would return empty).
	for i := len(result) - 1; i >= 0; i-- {
		result[i]++
		if result[i] != 0 {
			return result
		}
	}

	// All bytes were 0xFF — return a longer key.
	return append(result, 0)
}

// newPrefixIter creates an iterator over the half-open range
// [prefix, nextKey(prefix)). Callers must defer iter.Close().
func (e *pebbleEngine) newPrefixIter(prefix []byte) (*pebble.Iterator, error) {
	return e.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: nextKey(prefix),
	})
}
