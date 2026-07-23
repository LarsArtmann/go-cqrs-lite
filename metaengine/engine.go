package metaengine

import (
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

type ScanBackend interface {
	MapScan(
		collection string,
		filters []FieldPath,
		filterValues map[string]any,
		sortField string,
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

func (m *MemoryEngine) getMapLocked(col string) map[any]any {
	if m.data.maps[col] == nil {
		m.data.maps[col] = make(map[any]any)
	}

	return m.data.maps[col]
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
	v, ok := m.getMapLocked(col)[key]

	return v, ok, nil
}

func (m *MemoryEngine) MapDelete(col string, key any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.getMapLocked(col), key)

	return nil
}

func (m *MemoryEngine) MapScan(
	col string,
	filters []FieldPath,
	filterValues map[string]any,
	sortField string,
	limit int,
) ([]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	store := m.getMapLocked(col)

	fetchLimit := limit
	if limit > 0 {
		fetchLimit = limit + 1
	}

	var results []any

	for _, v := range store {
		if !matchesFilters(v, filters, filterValues) {
			continue
		}

		results = append(results, v)
		if fetchLimit > 0 && len(results) >= fetchLimit {
			break
		}
	}

	if sortField != "" {
		sort.Slice(results, func(i, j int) bool {
			return compareByField(results[i], results[j], sortField)
		})
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

func (m *MemoryEngine) getGraphLocked(col string) *memGraph {
	if m.data.graphs[col] == nil {
		m.data.graphs[col] = &memGraph{adjacency: make(map[any][]any)}
	}

	return m.data.graphs[col]
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
	g := m.getGraphLocked(col)
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

func compareByField(a, b any, field string) bool {
	av := getFieldValue(a, field)
	bv := getFieldValue(b, field)

	return compareLess(av, bv)
}

// compareLess performs a type-aware less-than comparison for sort fields.
// Falls back to string comparison for unsupported types.
func compareLess(a, b any) bool {
	switch va := a.(type) {
	case int:
		return va < b.(int)
	case int8:
		return va < b.(int8)
	case int16:
		return va < b.(int16)
	case int32:
		return va < b.(int32)
	case int64:
		return va < b.(int64)
	case uint:
		return va < b.(uint)
	case uint8:
		return va < b.(uint8)
	case uint16:
		return va < b.(uint16)
	case uint32:
		return va < b.(uint32)
	case uint64:
		return va < b.(uint64)
	case float32:
		return va < b.(float32)
	case float64:
		return va < b.(float64)
	case string:
		return va < b.(string)
	case time.Time:
		return va.Before(b.(time.Time))
	default:
		return fmt.Sprintf("%v", a) < fmt.Sprintf("%v", b)
	}
}
