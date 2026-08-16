package sqliteengine

import (
	"context"
	"database/sql"
	"fmt"
)

// --- undirected graph traversal ---
//
// Undirected queries expand every node along BOTH edge directions (outgoing
// via idx_graph_edges_from, incoming via idx_graph_edges_to). Like directed
// traversal they run as a single recursive CTE when the driver supports it,
// falling back to per-node-per-level indexed lookups otherwise.

// graphNeighborsUndirectedCTE walks the depth-limited neighborhood in one
// query, expanding every node along BOTH edge directions (outgoing via
// from_node, incoming via to_node).
const graphNeighborsUndirectedCTE = `WITH RECURSIVE walk(node, depth) AS (
	SELECT to_node, 1 FROM meta_graph_edges WHERE collection = ? AND from_node = ?
	UNION
	SELECT from_node, 1 FROM meta_graph_edges WHERE collection = ? AND to_node = ?
	UNION
	SELECT g.to_node, w.depth + 1
	FROM meta_graph_edges g JOIN walk w ON g.collection = ? AND g.from_node = w.node
	WHERE w.depth < ?
	UNION
	SELECT g.from_node, w.depth + 1
	FROM meta_graph_edges g JOIN walk w ON g.collection = ? AND g.to_node = w.node
	WHERE w.depth < ?
)
SELECT DISTINCT node FROM walk WHERE node <> ?`

// GraphNeighborsUndirected returns all nodes within depth hops when edges
// are followed in BOTH directions, via a recursive CTE (or the iterative
// fallback for drivers without WITH RECURSIVE).
func (e *sqliteEngine) GraphNeighborsUndirected(
	ctx context.Context,
	col string,
	node any,
	depth int,
) ([]any, error) {
	// art-dupl:accept CTE-vs-iterative dispatch mirrored in dep-isolated mysqlengine
	if depth <= 0 {
		return []any{}, nil
	}

	if e.graphCTE {
		return e.graphNeighborsUndirectedCTE(ctx, col, node, depth)
	}

	return e.graphNeighborsUndirectedIterative(ctx, col, node, depth)
}

func (e *sqliteEngine) graphNeighborsUndirectedCTE(
	ctx context.Context,
	col string,
	node any,
	depth int,
) ([]any, error) {
	start := encodeKey(node)
	rows, err := e.xd().QueryContext(ctx, graphNeighborsUndirectedCTE,
		col, start, col, start, col, depth, col, depth, start)
	if err != nil {
		return nil, fmt.Errorf("sqliteengine.GraphNeighborsUndirected: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result, err := scanNeighborRows(rows)
	if err != nil {
		return nil, fmt.Errorf("sqliteengine.GraphNeighborsUndirected: %w", err)
	}

	return result, nil
}

// graphNeighborsUndirectedIterative is the fallback for drivers without
// WITH RECURSIVE: one indexed lookup per node per level per direction.
func (e *sqliteEngine) graphNeighborsUndirectedIterative(
	ctx context.Context,
	col string,
	node any,
	depth int,
) ([]any, error) {
	startNode := encodeKey(node)
	visited := map[string]bool{startNode: true}
	frontier := []string{startNode}
	var result []any

	for level := 0; level < depth && len(frontier) > 0; level++ {
		var next []string

		for _, n := range frontier {
			outgoing, err := e.queryGraphNeighbors(ctx, col, n)
			if err != nil {
				return nil, fmt.Errorf("sqliteengine.GraphNeighborsUndirected: %w", err)
			}

			incoming, err := e.queryGraphReverseNeighbors(ctx, col, n)
			if err != nil {
				return nil, fmt.Errorf("sqliteengine.GraphNeighborsUndirected: %w", err)
			}

			for _, nb := range append(outgoing, incoming...) {
				if visited[nb] {
					continue
				}

				visited[nb] = true
				result = append(result, nb)
				next = append(next, nb)
			}
		}

		frontier = next
	}

	if result == nil {
		result = []any{}
	}

	return result, nil
}

// scanNeighborRows drains a neighbors query into []any, returning an empty
// (non-nil) slice for no rows.
func scanNeighborRows(rows *sql.Rows) ([]any, error) {
	// art-dupl:accept database/sql single-column drain; mysqlengine twin is dep-isolated
	var result []any
	for rows.Next() {
		var nb string
		if err := rows.Scan(&nb); err != nil {
			return nil, err //nolint:wrapcheck // wrapped by caller
		}

		result = append(result, nb)
	}

	if err := rows.Err(); err != nil {
		return nil, err //nolint:wrapcheck // wrapped by caller
	}

	if result == nil {
		result = []any{}
	}

	return result, nil
}
