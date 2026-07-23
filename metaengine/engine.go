package metaengine

import (
	"cmp"
	"fmt"
	"maps"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

// EngineProfile describes what an engine can do and at what cost.
type EngineProfile struct {
	Name     string
	Supports map[ADT]Complexity
}

func (p EngineProfile) SupportsADT(adt ADT) (Complexity, bool) {
	c, ok := p.Supports[adt]

	return c, ok
}

func (p EngineProfile) String() string {
	var parts []string
	for adt, c := range p.Supports {
		parts = append(parts, fmt.Sprintf("%s@%s", adt, c))
	}

	sort.Strings(parts)

	return fmt.Sprintf("%s: %s", p.Name, strings.Join(parts, " "))
}

// Per-ADT backend interfaces (ISP — engines implement only what they support).
// This mirrors the existing kv.ViewStore + ViewQuerier + ViewCounter pattern.

type MapBackend interface {
	MapSet(collection string, key any, value any) error
	MapGet(collection string, key any) (any, bool, error)
	MapDelete(collection string, key any) error
}

// MapUpdater is an optional capability for atomic read-modify-write on map entries.
// Engines that implement this interface handle FoldUpdate without race conditions.
// Mirrors the kv.ViewUpdater pattern from the existing codebase.
type MapUpdater interface {
	MapUpdate(collection string, key any, update func(prev any) any) error
}

// ScanBackend handles filtered+sorted scans for collection queries.
// Filters are runtime predicates (typed closures from FilterOn), not field
// name strings. Sort is a runtime comparator (from SortOn), or nil for default.
type ScanBackend interface {
	MapScan(
		collection string,
		filters []filterPredicate,
		sortFunc func(a, b any) int,
		cursor any,
		limit int,
	) ([]any, error)
}

type SetBackend interface {
	SetAdd(collection string, key any) error
	SetContains(collection string, key any) (bool, error)
}

type CounterBackend interface {
	CounterIncrement(collection string, deltas Delta) error
	CounterGet(collection string) (map[string]int64, error)
}

type GraphBackend interface {
	GraphAddEdge(collection string, edge Edge) error
	GraphNeighbors(collection string, node any, depth int) ([]any, error)
}

// Closer is the lifecycle interface.
type Closer interface {
	Close() error
}

// Engine is a storage backend with a cost profile.
// An engine implements whichever ADT backends it supports.
// The planner checks capabilities at runtime via type assertion.
type Engine interface {
	Profile() EngineProfile
	Closer
}

// MemoryEngine implements all ADT backends for testing and development.
type MemoryEngine struct {
	mu   sync.RWMutex
	data *memData
}

type memData struct {
	maps     map[string]map[any]any
	sets     map[string]map[any]struct{}
	counters map[string]map[string]int64
	graphs   map[string]*memGraph
}

type memGraph struct {
	adjacency map[any][]any
}

func NewMemoryEngine() *MemoryEngine {
	return &MemoryEngine{
		data: &memData{
			maps:     make(map[string]map[any]any),
			sets:     make(map[string]map[any]struct{}),
			counters: make(map[string]map[string]int64),
			graphs:   make(map[string]*memGraph),
		},
	}
}

func (m *MemoryEngine) Profile() EngineProfile {
	return EngineProfile{
		Name: "memory",
		Supports: map[ADT]Complexity{
			ADTMap:       ComplexityO1,
			ADTSet:       ComplexityO1,
			ADTCounter:   ComplexityO1,
			ADTGraph:     ComplexityODegree,
			ADTSortedMap: ComplexityON,
			ADTLog:       ComplexityON,
		},
	}
}

// getMapLocked returns or creates a map collection. Caller MUST hold m.mu.Lock().
func (m *MemoryEngine) getMapLocked(col string) map[any]any {
	if m.data.maps[col] == nil {
		m.data.maps[col] = make(map[any]any)
	}

	return m.data.maps[col]
}

// getGraphLocked returns or creates a graph collection. Caller MUST hold m.mu.Lock().
func (m *MemoryEngine) getGraphLocked(col string) *memGraph {
	if m.data.graphs[col] == nil {
		m.data.graphs[col] = &memGraph{adjacency: make(map[any][]any)}
	}

	return m.data.graphs[col]
}

func (m *MemoryEngine) MapSet(col string, key any, value any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.getMapLocked(col)[key] = value

	return nil
}

func (m *MemoryEngine) MapGet(col string, key any) (any, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	store := m.data.maps[col]
	if store == nil {
		return nil, false, nil
	}

	v, ok := store[key]

	return v, ok, nil
}

func (m *MemoryEngine) MapDelete(col string, key any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.getMapLocked(col), key)

	return nil
}

// MapUpdate performs an atomic read-modify-write on a map entry.
// The update function receives the previous value (nil if absent) and returns
// the new value. The entire operation is serialized under the engine's write lock.
func (m *MemoryEngine) MapUpdate(col string, key any, update func(prev any) any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	store := m.getMapLocked(col)
	prev := store[key]

	store[key] = update(prev)

	return nil
}

func (m *MemoryEngine) MapScan(
	col string,
	filters []FieldPath,
	filterValues map[string]any,
	sortField string,
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
		if !matchesFilters(v, filters, filterValues) {
			continue
		}

		pairs = append(pairs, kv{key: k, value: v})
	}

	// Sort with a deterministic tiebreaker: primary = sort field value,
	// secondary = map key (as string). This ensures reproducible output
	// even though Go randomizes map iteration order.
	sort.Slice(pairs, func(i, j int) bool {
		return compareItems(
			pairs[i].value, pairs[j].value,
			pairs[i].key, pairs[j].key,
			sortField,
		) < 0
	})

	// Keyset pagination: skip items at or before the cursor position.
	if cursor != nil && sortField != "" {
		cursorKey := extractKeyValue(cursor)

		filtered := pairs[:0]
		for _, p := range pairs {
			if compareItems(p.value, cursor, p.key, cursorKey, sortField) <= 0 {
				continue
			}

			filtered = append(filtered, p)
		}

		pairs = filtered
	}

	// Truncate to limit+1 — the extra item signals HasMore to reconstructPage.
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

func (m *MemoryEngine) SetAdd(col string, key any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.data.sets[col] == nil {
		m.data.sets[col] = make(map[any]struct{})
	}

	m.data.sets[col][key] = struct{}{}

	return nil
}

func (m *MemoryEngine) SetContains(col string, key any) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.data.sets[col][key]

	return ok, nil
}

func (m *MemoryEngine) CounterIncrement(col string, deltas Delta) error {
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

func (m *MemoryEngine) CounterGet(col string) (map[string]int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]int64, len(m.data.counters[col]))
	maps.Copy(result, m.data.counters[col])

	return result, nil
}

func (m *MemoryEngine) GraphAddEdge(col string, edge Edge) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	g := m.getGraphLocked(col)
	g.adjacency[edge.From] = append(g.adjacency[edge.From], edge.To)
	g.adjacency[edge.To] = append(g.adjacency[edge.To], edge.From)

	return nil
}

func (m *MemoryEngine) GraphNeighbors(col string, node any, depth int) ([]any, error) {
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

func (m *MemoryEngine) Close() error { return nil }

// SQLiteEngineProfile returns the cost profile for a SQLite engine.
// Used for multi-engine planning without a real SQLite implementation.
func SQLiteEngineProfile() EngineProfile {
	return EngineProfile{
		Name: "sqlite",
		Supports: map[ADT]Complexity{
			ADTMap:       ComplexityOLogN,
			ADTSet:       ComplexityOLogN,
			ADTCounter:   ComplexityO1,
			ADTGraph:     ComplexityON,
			ADTSortedMap: ComplexityOLogN,
			ADTLog:       ComplexityOLogN,
		},
	}
}

// matchesFilters checks if a value matches all filter criteria by field name.
func matchesFilters(value any, filters []FieldPath, filterValues map[string]any) bool {
	if len(filters) == 0 {
		return true
	}

	for _, filter := range filters {
		expected, ok := filterValues[filter.Field]
		if !ok {
			continue
		}

		actual := getFieldValue(value, filter.Field)
		if !reflect.DeepEqual(actual, expected) {
			return false
		}
	}

	return true
}

// compareValue performs a type-aware tri-state comparison: -1 (a < b), 0 (equal), +1 (a > b).
// Falls back to string comparison for unsupported or mismatched types.
func compareValue(a, b any) int {
	if a == nil || b == nil {
		if a == b {
			return 0
		}

		if a == nil {
			return -1
		}

		return 1
	}

	switch va := a.(type) {
	case int:
		if vb, ok := b.(int); ok {
			return cmp.Compare(va, vb)
		}
	case int8:
		if vb, ok := b.(int8); ok {
			return cmp.Compare(va, vb)
		}
	case int16:
		if vb, ok := b.(int16); ok {
			return cmp.Compare(va, vb)
		}
	case int32:
		if vb, ok := b.(int32); ok {
			return cmp.Compare(va, vb)
		}
	case int64:
		if vb, ok := b.(int64); ok {
			return cmp.Compare(va, vb)
		}
	case uint:
		if vb, ok := b.(uint); ok {
			return cmp.Compare(va, vb)
		}
	case uint8:
		if vb, ok := b.(uint8); ok {
			return cmp.Compare(va, vb)
		}
	case uint16:
		if vb, ok := b.(uint16); ok {
			return cmp.Compare(va, vb)
		}
	case uint32:
		if vb, ok := b.(uint32); ok {
			return cmp.Compare(va, vb)
		}
	case uint64:
		if vb, ok := b.(uint64); ok {
			return cmp.Compare(va, vb)
		}
	case float32:
		if vb, ok := b.(float32); ok {
			return cmp.Compare(va, vb)
		}
	case float64:
		if vb, ok := b.(float64); ok {
			return cmp.Compare(va, vb)
		}
	case string:
		if vb, ok := b.(string); ok {
			return cmp.Compare(va, vb)
		}
	case time.Time:
		if vb, ok := b.(time.Time); ok {
			switch {
			case va.Before(vb):
				return -1
			case va.After(vb):
				return 1
			default:
				return 0
			}
		}
	}

	return strings.Compare(fmt.Sprintf("%v", a), fmt.Sprintf("%v", b))
}

// compareItems compares two projection records by sort field (primary) and map
// key (secondary tiebreaker). The tiebreaker ensures deterministic ordering
// even when sort field values are equal and Go's map iteration is randomized.
// Returns -1 if a sorts before b, 0 if equal, +1 if a sorts after b.
func compareItems(a, b any, keyA, keyB any, sortField string) int {
	if sortField != "" {
		aVal := getFieldValue(a, sortField)
		bVal := getFieldValue(b, sortField)

		if c := compareValue(aVal, bVal); c != 0 {
			return c
		}
	}

	return strings.Compare(fmt.Sprintf("%v", keyA), fmt.Sprintf("%v", keyB))
}
