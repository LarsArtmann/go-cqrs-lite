package metaengine

import (
	"context"
	"fmt"
	"slices"
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
func (m *memoryEngine) GraphNeighbors(
	_ context.Context,
	collection string,
	node any,
	depth int,
) ([]any, error) {
	// art-dupl:accept RLock prologue shared with GraphNeighborsUndirected below
	m.mu.RLock()
	defer m.mu.RUnlock()

	adj := m.data.graphs[collection]
	if adj == nil {
		return nil, nil
	}

	// art-dupl:accept BFS init shared with GraphNeighborsUndirected below
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

// GraphRemoveEdge removes a directed edge from the adjacency list (ADR-0114
// style tombstone dispatch). Idempotent: removing a missing edge is a no-op,
// mirroring GraphAddEdge's duplicate-edge idempotency.
func (m *memoryEngine) GraphRemoveEdge(_ context.Context, collection string, edge Edge) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	adj := m.data.graphs[collection]
	if adj == nil {
		return nil
	}

	fromKey := fmt.Sprint(edge.From)
	toKey := fmt.Sprint(edge.To)

	adj[fromKey] = slices.DeleteFunc(adj[fromKey], func(existing any) bool {
		return existing == toKey
	})

	return nil
}

// GraphNeighborsUndirected returns all nodes within depth hops when edges
// are followed in BOTH directions. The memory engine stores directed
// adjacency lists, so incoming edges are found by scanning every from-node
// (O(N) per level) — acceptable for the universal fallback engine; SQL and
// LSM engines resolve reverse edges via an index or reverse key.
func (m *memoryEngine) GraphNeighborsUndirected(
	_ context.Context,
	collection string,
	node any,
	depth int,
) ([]any, error) {
	// art-dupl:accept RLock prologue shared with GraphNeighbors above
	m.mu.RLock()
	defer m.mu.RUnlock()

	adj := m.data.graphs[collection]
	if adj == nil || depth <= 0 {
		return []any{}, nil
	}

	start := fmt.Sprint(node)
	visited := map[string]bool{start: true}
	frontier := []string{start}
	var result []any

	for d := 0; d < depth && len(frontier) > 0; d++ {
		var next []string

		for _, n := range frontier {
			for _, neighbor := range undirectedNeighbors(adj, n) {
				if !visited[neighbor] {
					visited[neighbor] = true
					result = append(result, neighbor)
					next = append(next, neighbor)
				}
			}
		}

		frontier = next
	}

	if result == nil {
		result = []any{}
	}

	return result, nil
}

// undirectedNeighbors returns the outgoing neighbors of n plus every node
// with an edge INTO n (reverse scan over all from-nodes).
func undirectedNeighbors(adj map[any][]any, n string) []string {
	seen := map[string]bool{}
	var neighbors []string

	appendNeighbor := func(candidate any) {
		key, ok := candidate.(string)
		if !ok {
			key = fmt.Sprint(candidate)
		}

		if !seen[key] {
			seen[key] = true
			neighbors = append(neighbors, key)
		}
	}

	for _, to := range adj[n] {
		appendNeighbor(to)
	}

	for from, targets := range adj {
		fromStr, ok := from.(string)
		if !ok {
			fromStr = fmt.Sprint(from)
		}

		if fromStr == n {
			continue
		}

		for _, to := range targets {
			toStr, ok := to.(string)
			if !ok {
				toStr = fmt.Sprint(to)
			}

			if toStr == n {
				appendNeighbor(fromStr)

				break
			}
		}
	}

	return neighbors
}
