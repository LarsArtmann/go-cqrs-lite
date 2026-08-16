package pgengine

import (
	"context"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- graph edge removal + undirected traversal ---
//
// Both run natively on meta_graph_edges: removal is a targeted DELETE on the
// composite primary key; undirected traversal is a single WITH RECURSIVE
// statement expanding every node along BOTH edge directions (outgoing via
// idx_graph_edges_from, incoming via idx_graph_edges_to).

// pgGraphNeighborsUndirectedCTE walks the depth-limited neighborhood in one
// query, following edges in both directions.
//
// Exactly TWO arms: PostgreSQL (and MySQL) reject recursive references in
// what they parse as the non-recursive term — the seed's two directions are
// a derived table and the recursive step expands both directions through one
// CASE + OR join, keeping the single self-reference legal.
const pgGraphNeighborsUndirectedCTE = `WITH RECURSIVE walk(node, depth) AS (
	SELECT seed.n, 1 FROM (
		SELECT to_node AS n FROM meta_graph_edges WHERE collection = $1 AND from_node = $2
		UNION
		SELECT from_node AS n FROM meta_graph_edges WHERE collection = $1 AND to_node = $2
	) seed
	UNION
	SELECT CASE WHEN g.from_node = w.node THEN g.to_node ELSE g.from_node END, w.depth + 1
	FROM meta_graph_edges g
	JOIN walk w ON g.collection = $1 AND (g.from_node = w.node OR g.to_node = w.node)
	WHERE w.depth < $3
)
SELECT DISTINCT node FROM walk WHERE node <> $2`

// GraphRemoveEdge deletes the specific directed edge (ADR-0114 style
// tombstone dispatch). Idempotent: deleting a missing edge affects 0 rows.
func (e *pgEngine) GraphRemoveEdge(
	ctx context.Context,
	col string,
	edge metaengine.Edge,
) error {
	const q = `DELETE FROM meta_graph_edges WHERE collection = $1 AND from_node = $2 AND to_node = $3`

	if _, err := e.conn().
		ExecContext(ctx, q, col, encodeNodeKey(edge.From), encodeNodeKey(edge.To)); err != nil {
		return fmt.Errorf("pgengine.GraphRemoveEdge: %w", err)
	}

	return nil
}

// GraphNeighborsUndirected returns all nodes within depth hops when edges
// are followed in BOTH directions (excluding the start node), deduplicated,
// via a recursive CTE.
func (e *pgEngine) GraphNeighborsUndirected(
	ctx context.Context,
	col string,
	node any,
	depth int,
) ([]any, error) {
	if depth <= 0 {
		return []any{}, nil
	}

	start := encodeNodeKey(node)

	rows, err := e.conn().QueryContext(ctx, pgGraphNeighborsUndirectedCTE, col, start, depth)
	if err != nil {
		return nil, fmt.Errorf("pgengine.GraphNeighborsUndirected: %w", err)
	}
	defer metaengine.DeferClose(rows)

	var result []any
	for rows.Next() {
		var nb string
		if err := rows.Scan(&nb); err != nil {
			return nil, fmt.Errorf("pgengine.GraphNeighborsUndirected: row: %w", err)
		}

		result = append(result, nb)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pgengine.GraphNeighborsUndirected: %w", err)
	}

	if result == nil {
		result = []any{}
	}

	return result, nil
}
