package sqliteengine

import (
	"context"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- graph dispatch ---
//
// SQLite/Turso implements graph traversal natively via iterative BFS on
// meta_graph_edges. Each level of traversal issues simple indexed SELECTs
// (WHERE collection = ? AND from_node = ?), avoiding the need for recursive
// CTEs which are not supported by all SQL engines (notably libSQL/Turso).
//
// This is O(degree^depth) — far better than the degraded multimap BFS fallback
// (O(N * degree^depth)). The dedicated edges table with a composite index
// ensures each level lookup is O(logN).

const graphNeighborsDirectSQL = `SELECT to_node FROM meta_graph_edges WHERE collection = ? AND from_node = ?`

func (e *sqliteEngine) GraphAddEdge(
	ctx context.Context,
	col string,
	edge metaengine.Edge,
) error {
	from := encodeKey(edge.From)
	to := encodeKey(edge.To)

	if _, err := e.xc().exec(ctx, e.queries.graphAddEdge, col, from, to); err != nil {
		return fmt.Errorf("sqliteengine.GraphAddEdge: %w", err)
	}

	return nil
}

func (e *sqliteEngine) GraphNeighbors(
	ctx context.Context,
	col string,
	node any,
	depth int,
) ([]any, error) {
	if depth <= 0 {
		return []any{}, nil
	}

	startNode := encodeKey(node)
	visited := map[string]bool{startNode: true}
	frontier := []string{startNode}
	var result []any

	for level := 0; level < depth && len(frontier) > 0; level++ {
		var next []string

		for _, n := range frontier {
			neighbors, err := e.queryGraphNeighbors(ctx, col, n)
			if err != nil {
				return nil, fmt.Errorf("sqliteengine.GraphNeighbors: %w", err)
			}

			for _, nb := range neighbors {
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

func (e *sqliteEngine) queryGraphNeighbors(
	ctx context.Context,
	col, node string,
) ([]string, error) {
	rows, err := e.xc().query(ctx, graphNeighborsDirectSQL, col, node)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var neighbors []string

	for rows.Next() {
		var nb string
		if err := rows.Scan(&nb); err != nil {
			return nil, err
		}

		neighbors = append(neighbors, nb)
	}

	return neighbors, rows.Err()
}
