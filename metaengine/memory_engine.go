package metaengine

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
	"sync"
)

// memoryEngine implements all ADT backends for testing and development.
type memoryEngine struct {
	mu   sync.RWMutex
	data *memData
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
	}
}

func (m *memoryEngine) Profile() EngineProfile {
	return EngineProfile{
		Name:   "memory",
		NsPerOp: MemoryNsPerOp,
		Supports: map[ADT]Complexity{
			ADTMap:       ComplexityO1,
			ADTSet:       ComplexityO1,
			ADTCounter:   ComplexityO1,
			ADTGraph:     ComplexityODegree,
			ADTSortedMap: ComplexityON,
			ADTLog:       ComplexityON,
			ADTMultimap:  ComplexityO1,
		},
	}
}

// MemoryNsPerOp is the calibrated per-operation cost for the in-memory engine.
// Calibrated via BenchmarkCalibration_MapSet/MapGet on 2026-07-25 (AMD Ryzen
// AI MAX+ 395):
//   - MapSet: ~466 ns/op (mutex-protected map insert + JSON marshal)
//   - MapGet: ~21 ns/op (mutex-protected map lookup)
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

func (m *memoryEngine) MapSet(ctx context.Context, col string, key any, value any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.getMapLocked(col)[key] = value

	return nil
}

func (m *memoryEngine) MapGet(ctx context.Context, col string, key any) (any, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	store := m.data.maps[col]
	if store == nil {
		return nil, false, nil
	}

	v, ok := store[key]

	return v, ok, nil
}

func (m *memoryEngine) MapDelete(ctx context.Context, col string, key any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.getMapLocked(col), key)

	return nil
}

// MapUpdate performs an atomic read-modify-write on a map entry.
// The update function receives the previous value (nil if absent) and returns
// the new value. The entire operation is serialized under the engine's write lock.
func (m *memoryEngine) MapUpdate(
	ctx context.Context,
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
	ctx context.Context,
	col string,
	filters []filterPredicate,
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
		if !passesFilters(v, filters) {
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

func (m *memoryEngine) SetAdd(ctx context.Context, col string, key any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.data.sets[col] == nil {
		m.data.sets[col] = make(map[any]struct{})
	}

	m.data.sets[col][key] = struct{}{}

	return nil
}

func (m *memoryEngine) SetContains(ctx context.Context, col string, key any) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.data.sets[col][key]

	return ok, nil
}

func (m *memoryEngine) CounterIncrement(ctx context.Context, col string, deltas Delta) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.data.counters[col] == nil {
		m.data.counters[col] = make(map[string]int64)
	}

	for k, d := range deltas {
		m.data.counters[col][k] += d
	}

	return nil
}

func (m *memoryEngine) CounterGet(ctx context.Context, col string) (map[string]int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]int64, len(m.data.counters[col]))
	maps.Copy(result, m.data.counters[col])

	return result, nil
}

func (m *memoryEngine) GraphAddEdge(ctx context.Context, col string, edge Edge) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	g := m.getGraphLocked(col)
	g.adjacency[edge.From] = append(g.adjacency[edge.From], edge.To)
	g.adjacency[edge.To] = append(g.adjacency[edge.To], edge.From)

	return nil
}

func (m *memoryEngine) GraphNeighbors(
	ctx context.Context,
	col string,
	node any,
	depth int,
) ([]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	g := m.data.graphs[col]
	if g == nil {
		return nil, nil
	}

	visited := map[any]bool{node: true}
	frontier := []any{node}
	result := []any{}

	for d := 0; d < depth && len(frontier) > 0; d++ {
		var next []any

		for _, n := range frontier {
			for _, neighbor := range g.adjacency[n] {
				if !visited[neighbor] {
					visited[neighbor] = true
					result = append(result, neighbor)
					next = append(next, neighbor)
				}
			}
		}

		frontier = next
	}

	return result, nil
}

func (m *memoryEngine) Close() error { return nil }

func (m *memoryEngine) MultiAdd(ctx context.Context, col string, key any, value any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.data.multimaps[col] == nil {
		m.data.multimaps[col] = make(map[any][]any)
	}

	m.data.multimaps[col][key] = append(m.data.multimaps[col][key], value)

	return nil
}

func (m *memoryEngine) MultiGet(ctx context.Context, col string, key any) ([]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	values := m.data.multimaps[col][key]

	return slices.Clone(values), nil
}

func (m *memoryEngine) LogAppend(ctx context.Context, col string, value any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data.logs[col] = append(m.data.logs[col], value)

	return nil
}

func (m *memoryEngine) LogTail(ctx context.Context, col string, limit int) ([]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	log := m.data.logs[col]
	if limit <= 0 || limit > len(log) {
		limit = len(log)
	}

	start := len(log) - limit

	return slices.Clone(log[start:]), nil
}
