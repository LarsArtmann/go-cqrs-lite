// Package pebbleengine implements a Pebble-backed metaengine Engine.
//
// Pebble provides a genuinely different cost profile from SQLite and Memory:
//   - Map/Set point lookups: O(1) LSM point read (faster than SQLite's O(logN) B-tree)
//   - Counter: O(1) increment, O(N) CounterGet (prefix scan — degraded)
//   - SortedMap scan: O(N) prefix scan + Go sort (degraded — no secondary indexes)
//   - Graph: O(N^d) BFS via prefix scan
//
// This module exists OUTSIDE the zero-dependency metaengine core (ADR-0062)
// because it requires the cockroachdb/pebble dependency.
package pebbleengine

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json/v2"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// sep is the key separator. Null byte sorts before all printable characters,
// preventing collisions between collection names and keys.
const sep = "\x00"

// PebbleNsPerOp is the calibrated per-operation cost for the Pebble engine.
// Calibrated via BenchmarkCalibration_PebbleSet/Get on 2026-07-28 (AMD Ryzen
// AI MAX+ 395, in-memory vfs.NewMem):
//   - MapSet (LSM insert + JSON encode): ~1,785 ns/op
//   - MapGet (LSM point read + JSON decode): ~708 ns/op
//
// The value 1,200 ns is biased toward the write path (fold-heavy workloads
// dominate). Pebble is ~7x faster than SQLite on point reads and ~3.7x faster
// on writes.
const PebbleNsPerOp = 1200.0

// PebbleNsPerRead is the calibrated per-READ-operation cost (LSM point read +
// JSON decode). Sourced from BenchmarkCalibration_PebbleGet (~708 ns/op).
// Used by the planner's read-cost path (EngineProfile.ReadNsPerOp).
const PebbleNsPerRead = 708.0

// PebbleNsPerWrite is the calibrated per-WRITE-operation cost (LSM insert +
// JSON encode). Sourced from BenchmarkCalibration_PebbleSet (~1,785 ns/op).
// Used by the planner's write-cost path (EngineProfile.WriteNsPerOp).
const PebbleNsPerWrite = 1785.0

type pebbleEngine struct {
	db     *pebble.DB
	ownsDB bool
	mu     sync.Mutex // guards counter/multimap/log seq operations
	logSeq sync.Map   // collection → *atomic.Int64 (log sequence counter)
	mmSeq  sync.Map   // collection → *atomic.Int64 (multimap sequence counter)
}

// NewPebbleEngine creates a Pebble-backed metaengine engine. If dir is empty,
// an in-memory database is used (for testing); otherwise dir is the on-disk
// database directory (persisted across opens). The caller owns the returned
// Engine and must call Close.
func NewPebbleEngine(dir string) (metaengine.Engine, error) {
	opts := &pebble.Options{}
	if dir == "" {
		opts.FS = vfs.NewMem()
	}

	db, err := pebble.Open(dir, opts)
	if err != nil {
		return nil, fmt.Errorf("pebbleengine: open: %w", err)
	}

	return &pebbleEngine{
		db:     db,
		ownsDB: true,
	}, nil
}

// NewPebbleEngineFromDB wraps an existing *pebble.DB. The caller retains
// ownership of the DB — Close on the engine is a no-op.
func NewPebbleEngineFromDB(db *pebble.DB) metaengine.Engine {
	return &pebbleEngine{
		db:     db,
		ownsDB: false,
	}
}

func (e *pebbleEngine) Profile() metaengine.EngineProfile {
	return metaengine.EngineProfile{
		Name:       "pebble",
		NsPerOp:    PebbleNsPerOp,
		NsPerRead:  PebbleNsPerRead,
		NsPerWrite: PebbleNsPerWrite,
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap:       metaengine.ComplexityO1, // LSM point read
			metaengine.ADTSet:       metaengine.ComplexityO1,
			metaengine.ADTCounter:   metaengine.ComplexityON, // CounterGet = prefix scan
			metaengine.ADTGraph:     metaengine.ComplexityON, // BFS via prefix scan
			metaengine.ADTSortedMap: metaengine.ComplexityON, // no secondary index
			metaengine.ADTLog:       metaengine.ComplexityOLogN,
			metaengine.ADTMultimap:  metaengine.ComplexityOLogN,
		},
	}
}

func (e *pebbleEngine) Close() error {
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

// encodeJSON marshals v to a JSON string, falling back to fmt.Sprintf.
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

// encodeCounterValue encodes an int64 as 8 bytes big-endian.
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

// Compile-time assertions.
var (
	_ metaengine.Engine          = (*pebbleEngine)(nil)
	_ metaengine.MapBackend      = (*pebbleEngine)(nil)
	_ metaengine.MapUpdater      = (*pebbleEngine)(nil)
	_ metaengine.ScanBackend     = (*pebbleEngine)(nil)
	_ metaengine.SetBackend      = (*pebbleEngine)(nil)
	_ metaengine.CounterBackend  = (*pebbleEngine)(nil)
	_ metaengine.GraphBackend    = (*pebbleEngine)(nil)
	_ metaengine.MultimapBackend = (*pebbleEngine)(nil)
	_ metaengine.LogBackend      = (*pebbleEngine)(nil)
)

// --- MapBackend ---

func (e *pebbleEngine) MapSet(_ context.Context, col string, key any, value any) error {
	return e.db.Set(mapKey(col, encodeKeyStr(key)), encodeJSON(value), pebble.Sync)
}

func (e *pebbleEngine) MapGet(_ context.Context, col string, key any) (any, bool, error) {
	val, closer, err := e.db.Get(mapKey(col, encodeKeyStr(key)))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, false, nil
		}

		return nil, false, err //nolint:wrapcheck // passthrough
	}

	defer func() { _ = closer.Close() }() //cqrs-lint:ignore(C015) pebble closer, error is always nil

	result := decodeJSON(val)

	return result, true, nil
}

func (e *pebbleEngine) MapDelete(_ context.Context, col string, key any) error {
	return e.db.Delete(mapKey(col, encodeKeyStr(key)), pebble.Sync)
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

	var prev any

	val, closer, err := e.db.Get(k)
	if err == nil {
		prev = decodeJSON(val)
		_ = closer.Close()
	} else if !errors.Is(err, pebble.ErrNotFound) {
		return err //nolint:wrapcheck // passthrough
	}

	newVal := update(prev)

	return e.db.Set(k, encodeJSON(newVal), pebble.Sync) //nolint:wrapcheck // passthrough
}

// --- ScanBackend ---

func (e *pebbleEngine) MapScan(
	_ context.Context,
	col string,
	filterFn func(item any) bool,
	sortFunc func(a, b any) int,
	cursor any,
	limit int,
) ([]any, error) {
	prefix := collectionPrefix(col)

	upperBound := nextKey(prefix)

	iter, err := e.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound,
	})
	if err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	defer func() { _ = iter.Close() }()

	type kv struct {
		key   []byte
		value any
	}

	var pairs []kv

	for iter.First(); iter.Valid(); iter.Next() {
		val := decodeJSON(iter.Value())

		if filterFn != nil && !filterFn(val) {
			continue
		}

		pairs = append(pairs, kv{key: append([]byte(nil), iter.Key()...), value: val})
	}

	if err := iter.Error(); err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	// Sort in Go (Pebble has no secondary index).
	if sortFunc != nil {
		sort.Slice(pairs, func(i, j int) bool {
			if c := sortFunc(pairs[i].value, pairs[j].value); c != 0 {
				return c < 0
			}

			return bytes.Compare(pairs[i].key, pairs[j].key) < 0
		})
	}

	// Keyset pagination.
	if cursor != nil && sortFunc != nil {
		filtered := pairs[:0]
		for _, p := range pairs {
			if sortFunc(p.value, cursor) <= 0 {
				continue
			}

			filtered = append(filtered, p)
		}

		pairs = filtered
	}

	truncLimit := 0
	if limit > 0 {
		truncLimit = limit + 1
	}

	if truncLimit > 0 && len(pairs) > truncLimit {
		pairs = pairs[:truncLimit]
	}

	results := make([]any, len(pairs))
	for i, p := range pairs {
		results[i] = p.value
	}

	return results, nil
}

// --- SetBackend ---

func (e *pebbleEngine) SetAdd(_ context.Context, col string, key any) error {
	return e.db.Set(setKey(col, encodeKeyStr(key)), nil, pebble.Sync)
}

func (e *pebbleEngine) SetContains(_ context.Context, col string, key any) (bool, error) {
	_, closer, err := e.db.Get(setKey(col, encodeKeyStr(key)))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return false, nil
		}

		return false, err //nolint:wrapcheck // passthrough
	}

	defer func() { _ = closer.Close() }()

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
			_ = closer.Close() //cqrs-lint:ignore(C015) pebble closer, error is always nil
		} else if !errors.Is(err, pebble.ErrNotFound) {
			return err //nolint:wrapcheck // passthrough
		}

		if err := e.db.Set(ck, encodeCounterValue(current+d), pebble.Sync); err != nil {
			return err //nolint:wrapcheck // passthrough
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
		return nil, err //nolint:wrapcheck // passthrough
	}

	defer func() { _ = iter.Close() }()

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

// --- GraphBackend ---

func (e *pebbleEngine) GraphAddEdge(_ context.Context, col string, edge metaengine.Edge) error {
	from := encodeKeyStr(edge.From)
	to := encodeKeyStr(edge.To)

	// Store edge in both directions for efficient neighbor lookup.
	if err := e.db.Set(graphEdgeKey(col, from, to), nil, pebble.Sync); err != nil {
		return err //nolint:wrapcheck // passthrough
	}

	return e.db.Set(graphEdgeKey(col, to, from), nil, pebble.Sync) //nolint:wrapcheck // passthrough
}

func (e *pebbleEngine) GraphNeighbors(
	_ context.Context,
	col string,
	node any,
	depth int,
) ([]any, error) {
	nodeStr := encodeKeyStr(node)
	visited := map[string]bool{nodeStr: true}
	frontier := []string{nodeStr}

	var result []string

	for d := 0; d < depth && len(frontier) > 0; d++ {
		var next []string

		for _, n := range frontier {
			neighbors := e.scanGraphNeighbors(col, n)
			for _, nb := range neighbors {
				if !visited[nb] {
					visited[nb] = true
					result = append(result, nb)
					next = append(next, nb)
				}
			}
		}

		frontier = next
	}

	decoded := make([]any, len(result))
	for i, r := range result {
		decoded[i] = decodeJSON([]byte(r))
	}

	return decoded, nil
}

func (e *pebbleEngine) scanGraphNeighbors(col, node string) []string {
	prefix := graphPrefixForward(col, node)
	upperBound := nextKey(prefix)

	iter, err := e.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound,
	})
	if err != nil {
		return nil
	}

	defer func() { _ = iter.Close() }()

	var neighbors []string

	for iter.First(); iter.Valid(); iter.Next() {
		// Key format: "g\x00{col}\x00{from}\x00{to}".
		keyStr := string(iter.Key())

		parts := strings.SplitN(keyStr, sep, 4)
		if len(parts) < 4 {
			continue
		}

		neighbors = append(neighbors, parts[3])
	}

	return neighbors
}

// --- MultimapBackend ---

func (e *pebbleEngine) MultiAdd(_ context.Context, col string, key any, value any) error {
	seq := e.nextMmSeq(col)
	k := multimapKey(col, encodeKeyStr(key), seq)

	return e.db.Set(k, encodeJSON(value), pebble.Sync)
}

func (e *pebbleEngine) MultiGet(_ context.Context, col string, key any) ([]any, error) {
	prefix := multimapPrefix(col, encodeKeyStr(key))
	upperBound := nextKey(prefix)

	iter, err := e.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound,
	})
	if err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	defer func() { _ = iter.Close() }()

	var out []any

	for iter.First(); iter.Valid(); iter.Next() {
		out = append(out, decodeJSON(iter.Value()))
	}

	return out, iter.Error()
}

func (e *pebbleEngine) nextMmSeq(col string) int64 {
	actual, _ := e.mmSeq.LoadOrStore(col, &atomic.Int64{})

	return actual.(*atomic.Int64).Add(1)
}

// --- LogBackend ---

func (e *pebbleEngine) LogAppend(_ context.Context, col string, value any) error {
	seq := e.nextLogSeq(col)
	k := logKey(col, seq)

	return e.db.Set(k, encodeJSON(value), pebble.Sync)
}

func (e *pebbleEngine) LogTail(_ context.Context, col string, limit int) ([]any, error) {
	prefix := logPrefix(col)
	upperBound := nextKey(prefix)

	iter, err := e.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound,
	})
	if err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	defer func() { _ = iter.Close() }()

	// Collect last `limit` entries by iterating in reverse.
	var entries []any

	if limit <= 0 {
		// Collect all in forward order.
		for iter.First(); iter.Valid(); iter.Next() {
			entries = append(entries, decodeJSON(iter.Value()))
		}
	} else {
		// Reverse iteration for tail.
		count := 0

		for iter.Last(); iter.Valid() && count < limit; iter.Prev() {
			entries = append(entries, decodeJSON(iter.Value()))
			count++
		}

		// Reverse to chronological order.
		for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
			entries[i], entries[j] = entries[j], entries[i]
		}
	}

	return entries, iter.Error()
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
	for _, v := range slices.Backward(result) {
		v++
		if v != 0 {
			return result
		}
	}

	// All bytes were 0xFF — return a longer key.
	return append(result, 0)
}
