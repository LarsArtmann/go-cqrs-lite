package metaengine

import (
	"context"
	"fmt"
)

// GraphAddEdge adds a directed edge to an adjacency list stored in the
// memory engine's collection namespace. Implements the graphBackend
// dispatch contract so the memory engine serves as the universal fallback
// for graph queries (ADR: memory supports all ADTs).
func (m *memoryEngine) GraphAddEdge(_ context.Context, collection string, edge Edge) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.data.graphs == nil {
		m.data.graphs = make(map[string]map[any][]any)
	}

	adj := m.data.graphs[collection]
	if adj == nil {
		adj = make(map[any][]any)
		m.data.graphs[collection] = adj
	}

	fromKey := fmt.Sprint(edge.From)
	toKey := fmt.Sprint(edge.To)

	for _, existing := range adj[fromKey] {
		if existing == toKey {
			return nil
		}
	}

	adj[fromKey] = append(adj[fromKey], toKey)

	return nil
}

// GraphNeighbors returns all nodes reachable from node within the given depth.
// Depth 1 returns direct neighbors; depth 2 includes neighbors-of-neighbors.
func (m *memoryEngine) GraphNeighbors(_ context.Context, collection string, node any, depth int) ([]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	adj := m.data.graphs[collection]
	if adj == nil {
		return nil, nil
	}

	start := fmt.Sprint(node)
	visited := map[string]bool{start: true}
	frontier := []string{start}
	var result []any

	for d := 0; d < depth && len(frontier) > 0; d++ {
		var next []string

		for _, n := range frontier {
			for _, neighbor := range adj[n] {
				neighborStr, ok := neighbor.(string)
				if !ok {
					neighborStr = fmt.Sprint(neighbor)
				}

				if !visited[neighborStr] {
					visited[neighborStr] = true
					result = append(result, neighborStr)
					next = append(next, neighborStr)
				}
			}
		}

		frontier = next
	}

	return result, nil
}
