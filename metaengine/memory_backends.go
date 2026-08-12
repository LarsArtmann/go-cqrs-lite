package metaengine

import (
	"context"
	"maps"
	"slices"
)

// --- SetBackend ---

func (m *memoryEngine) SetAdd(_ context.Context, col string, key any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.data.sets[col] == nil {
		m.data.sets[col] = make(map[any]struct{})
	}

	m.data.sets[col][key] = struct{}{}

	return nil
}

func (m *memoryEngine) SetContains(_ context.Context, col string, key any) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.data.sets[col][key]

	return ok, nil
}

// --- CounterBackend ---

func (m *memoryEngine) CounterIncrement(_ context.Context, col string, deltas Delta) error {
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

func (m *memoryEngine) CounterGet(_ context.Context, col string) (map[string]int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]int64, len(m.data.counters[col]))
	maps.Copy(result, m.data.counters[col])

	return result, nil
}

// --- GraphBackend ---

func (m *memoryEngine) GraphAddEdge(_ context.Context, col string, edge Edge) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	g := m.getGraphLocked(col)
	g.adjacency[edge.From] = append(g.adjacency[edge.From], edge.To)
	g.adjacency[edge.To] = append(g.adjacency[edge.To], edge.From)

	return nil
}

func (m *memoryEngine) GraphNeighbors(
	_ context.Context,
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

// --- Lifecycle ---

func (m *memoryEngine) Close() error { return nil }

// --- MultimapBackend ---

func (m *memoryEngine) MultiAdd(_ context.Context, col string, key any, value any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.data.multimaps[col] == nil {
		m.data.multimaps[col] = make(map[any][]any)
	}

	m.data.multimaps[col][key] = append(m.data.multimaps[col][key], value)

	return nil
}

func (m *memoryEngine) MultiGet(_ context.Context, col string, key any) ([]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	values := m.data.multimaps[col][key]

	return slices.Clone(values), nil
}

// --- LogBackend ---

func (m *memoryEngine) LogAppend(_ context.Context, col string, value any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data.logs[col] = append(m.data.logs[col], value)

	return nil
}

func (m *memoryEngine) LogTail(_ context.Context, col string, limit int) ([]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	log := m.data.logs[col]
	if limit <= 0 || limit > len(log) {
		limit = len(log)
	}

	start := len(log) - limit

	return slices.Clone(log[start:]), nil
}
