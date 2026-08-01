package metaengine

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// memoryEngine implements all ADT backends for testing and development.
type memoryEngine struct {
	mu         sync.RWMutex
	data       *memData
	vectorIdx  *MemoryVectorIndex
	searchIdx  *MemorySearchIndex
	spatialIdx *MemorySpatialIndex
	versions   map[string]map[string]*versionChain // collection → key → chain
}

type memData struct {
	maps      map[string]map[any]any
	sets      map[string]map[any]struct{}
	counters  map[string]map[string]int64
	graphs    map[string]*memGraph
	multimaps map[string]map[any][]any
	logs      map[string][]any
}

type memGraph struct {
	adjacency map[any][]any
}

func NewMemoryEngine() Engine {
	return &memoryEngine{
		data: &memData{
			maps:      make(map[string]map[any]any),
			sets:      make(map[string]map[any]struct{}),
			counters:  make(map[string]map[string]int64),
			graphs:    make(map[string]*memGraph),
			multimaps: make(map[string]map[any][]any),
			logs:      make(map[string][]any),
		},
		vectorIdx:  NewMemoryVectorIndex(),
		searchIdx:  NewMemorySearchIndex(),
		spatialIdx: NewMemorySpatialIndex(),
		versions:   make(map[string]map[string]*versionChain),
	}
}

func (m *memoryEngine) Profile() EngineProfile {
	return EngineProfile{
		Name:    "memory",
		NsPerOp: MemoryNsPerOp,
		Supports: map[ADT]Complexity{
			ADTMap:       ComplexityO1,
			ADTSet:       ComplexityO1,
			ADTCounter:   ComplexityO1,
			ADTGraph:     ComplexityODegree,
			ADTSortedMap: ComplexityON,
			ADTLog:       ComplexityON,
			ADTMultimap:  ComplexityO1,
			ADTVector:    ComplexityON,
			ADTSearch:    ComplexityON,
			ADTSpatial:   ComplexityON,
		},
	}
}

// MemoryNsPerOp is the calibrated per-operation cost for the in-memory engine.
// Calibrated via BenchmarkCalibration_MapSet/MapGet on 2026-07-25 (AMD Ryzen
// AI MAX+ 395):
//   - MapSet: ~466 ns/op (mutex-protected map insert + JSON marshal)
//   - MapGet: ~21 ns/op (mutex-protected map lookup)
//
// The value 500 ns is a conservative round-up: fold-heavy workloads (inserts)
// dominate the cost, so we bias toward the write path.
const MemoryNsPerOp = 500.0

// getMapLocked returns or creates a map collection. Caller MUST hold m.mu.Lock().
func (m *memoryEngine) getMapLocked(col string) map[any]any {
	if m.data.maps[col] == nil {
		m.data.maps[col] = make(map[any]any)
	}

	return m.data.maps[col]
}

// getGraphLocked returns or creates a graph collection. Caller MUST hold m.mu.Lock().
func (m *memoryEngine) getGraphLocked(col string) *memGraph {
	if m.data.graphs[col] == nil {
		m.data.graphs[col] = &memGraph{adjacency: make(map[any][]any)}
	}

	return m.data.graphs[col]
}

func (m *memoryEngine) MapSet(_ context.Context, col string, key any, value any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.getMapLocked(col)[key] = value
	m.recordVersion(col, fmt.Sprint(key), value)

	return nil
}

func (m *memoryEngine) MapGet(_ context.Context, col string, key any) (any, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	store := m.data.maps[col]
	if store == nil {
		return nil, false, nil
	}

	v, ok := store[key]

	return v, ok, nil
}

func (m *memoryEngine) MapDelete(_ context.Context, col string, key any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.getMapLocked(col), key)
	m.recordVersion(col, fmt.Sprint(key), nil) // nil = tombstone for version chain

	return nil
}

// MapUpdate performs an atomic read-modify-write on a map entry.
// The update function receives the previous value (nil if absent) and returns
// the new value. The entire operation is serialized under the engine's write lock.
func (m *memoryEngine) MapUpdate(
	_ context.Context,
	col string,
	key any,
	update func(prev any) any,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	store := m.getMapLocked(col)
	prev := store[key]

	store[key] = update(prev)

	return nil
}

func (m *memoryEngine) MapScan(
	_ context.Context,
	col string,
	filterFn func(item any) bool,
	sortFunc func(a, b any) int,
	cursor any,
	limit int,
) ([]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	store := m.data.maps[col]
	if store == nil {
		return nil, nil
	}

	// Collect ALL matching (key, value) pairs before sorting.
	// Early-break before sorting produces wrong results for paginated queries
	// because the items with the lowest sort keys may be in the unscanned portion.
	type kv struct {
		key   any
		value any
	}

	var pairs []kv

	for k, v := range store {
		if filterFn != nil && !filterFn(v) {
			continue
		}

		pairs = append(pairs, kv{key: k, value: v})
	}

	// Sort with a deterministic tiebreaker: primary = sort comparator,
	// secondary = map key (as string). This ensures reproducible output
	// even though Go randomizes map iteration order.
	sort.Slice(pairs, func(i, j int) bool {
		if sortFunc != nil {
			if c := sortFunc(pairs[i].value, pairs[j].value); c != 0 {
				return c < 0
			}
		}

		return strings.Compare(fmt.Sprintf("%v", pairs[i].key), fmt.Sprintf("%v", pairs[j].key)) < 0
	})

	// Keyset pagination: skip items at or before the cursor position.
	if cursor != nil && sortFunc != nil {
		filtered := pairs[:0]
		for _, p := range pairs {
			c := sortFunc(p.value, cursor)
			if c <= 0 {
				continue
			}

			filtered = append(filtered, p)
		}

		pairs = filtered
	}

	// Truncate to limit+1 — the extra item signals HasMore to reconstructCollection.
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

// --- VectorBackend ---

func (m *memoryEngine) VectorInsert(ctx context.Context, col string, emb Embedding) error {
	return m.vectorIdx.Insert(ctx, col, emb)
}

func (m *memoryEngine) VectorSearch(ctx context.Context, col string, query []float32, k int, metric string) ([]VectorResult, error) {
	return m.vectorIdx.Search(ctx, col, query, k, metric)
}

// --- SearchBackend ---

func (m *memoryEngine) SearchInsert(ctx context.Context, col string, doc IndexedText) error {
	return m.searchIdx.Insert(ctx, col, doc)
}

func (m *memoryEngine) SearchQuery(ctx context.Context, col, query string, limit int) ([]SearchResult, error) {
	return m.searchIdx.Query(ctx, col, query, limit)
}

// --- SpatialBackend ---

func (m *memoryEngine) SpatialInsert(ctx context.Context, col string, pt Point) error {
	return m.spatialIdx.Insert(ctx, col, pt)
}

func (m *memoryEngine) SpatialRange(ctx context.Context, col string, x, y, radius float64, limit int) ([]SpatialResult, error) {
	return m.spatialIdx.Range(ctx, col, x, y, radius, limit)
}
