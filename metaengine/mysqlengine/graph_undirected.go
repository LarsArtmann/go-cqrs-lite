package mysqlengine

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- graph edge removal + undirected traversal ---
//
// Removal is a targeted DELETE on the composite primary key. Undirected
// traversal expands both edge directions: recursive CTE when the server
// supports it (MySQL 8.0+/MariaDB 10.2+), otherwise iterative BFS with one
// indexed lookup per node per level per direction (idx_graph_edges_from /
// idx_graph_edges_to).

// mysqlGraphNeighborsUndirectedCTE walks the depth-limited neighborhood in
// one query, expanding every node along BOTH edge directions.
const mysqlGraphNeighborsUndirectedCTE = `WITH RECURSIVE walk(node, depth) AS (
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

// mysqlGraphNeighborsReverse expands against the reverse direction (incoming
// edges). Served by idx_graph_edges_to.
const mysqlGraphNeighborsReverse = `SELECT from_node FROM meta_graph_edges WHERE collection = ? AND to_node = ?`

// GraphRemoveEdge deletes the specific directed edge (ADR-0114 style
// tombstone dispatch). Idempotent: deleting a missing edge affects 0 rows.
func (e *mysqlEngine) GraphRemoveEdge(
	ctx context.Context,
	col string,
	edge metaengine.Edge,
) error {
	const q = `DELETE FROM meta_graph_edges WHERE collection = ? AND from_node = ? AND to_node = ?`

	if _, err := e.conn().
		ExecContext(ctx, q, col, encodeNodeKey(edge.From), encodeNodeKey(edge.To)); err != nil {
		return fmt.Errorf("mysqlengine.GraphRemoveEdge: %w", err)
	}

	return nil
}

// GraphNeighborsUndirected returns all nodes within depth hops when edges
// are followed in BOTH directions.
func (e *mysqlEngine) GraphNeighborsUndirected(
	ctx context.Context,
	col string,
	node any,
	depth int,
) ([]any, error) {
	if depth <= 0 {
		return []any{}, nil
	}

	if e.graphCTE {
		return e.graphNeighborsUndirectedCTE(ctx, col, node, depth)
	}

	return e.graphNeighborsUndirectedIterative(ctx, col, node, depth)
}

func (e *mysqlEngine) graphNeighborsUndirectedCTE(
	ctx context.Context,
	col string,
	node any,
	depth int,
) ([]any, error) {
	start := encodeNodeKey(node)

	rows, err := e.conn().QueryContext(ctx, mysqlGraphNeighborsUndirectedCTE,
		col, start, col, start, col, depth, col, depth, start)
	if err != nil {
		return nil, fmt.Errorf("mysqlengine.GraphNeighborsUndirected: %w", err)
	}
	defer metaengine.DeferClose(rows)

	result, err := scanGraphRows(rows)
	if err != nil {
		return nil, fmt.Errorf("mysqlengine.GraphNeighborsUndirected: %w", err)
	}

	return result, nil
}

// graphNeighborsUndirectedIterative is the fallback for servers without WITH
// RECURSIVE: one indexed lookup per node per level per direction.
func (e *mysqlEngine) graphNeighborsUndirectedIterative(
	ctx context.Context,
	col string,
	node any,
	depth int,
) ([]any, error) {
	startNode := encodeNodeKey(node)
	visited := map[string]bool{startNode: true}
	frontier := []string{startNode}
	var result []any

	for level := 0; level < depth && len(frontier) > 0; level++ {
		var next []string

		for _, n := range frontier {
			neighbors, err := e.graphNeighborsBothDirections(ctx, col, n)
			if err != nil {
				return nil, fmt.Errorf("mysqlengine.GraphNeighborsUndirected: %w", err)
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

// graphNeighborsBothDirections reads a node's outgoing and incoming adjacency.
func (e *mysqlEngine) graphNeighborsBothDirections(
	ctx context.Context,
	col, node string,
) ([]string, error) {
	outgoing, err := e.queryGraphNeighbors(ctx, col, node)
	if err != nil {
		return nil, err
	}

	rows, err := e.conn().QueryContext(ctx, mysqlGraphNeighborsReverse, col, node)
	if err != nil {
		return nil, err
	}
	defer metaengine.DeferClose(rows)

	for rows.Next() {
		var nb string
		if err := rows.Scan(&nb); err != nil {
			return nil, err //nolint:wrapcheck // wrapped by caller
		}

		outgoing = append(outgoing, nb)
	}

	if err := rows.Err(); err != nil {
		return nil, err //nolint:wrapcheck // wrapped by caller
	}

	return outgoing, nil
}

// scanGraphRows drains a neighbors query into []any, returning an empty
// (non-nil) slice for no rows.
func scanGraphRows(rows *sql.Rows) ([]any, error) {
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

// isMySQLDuplicateIndex reports whether err carries MySQL error 1061
// (duplicate index name) — init tolerates it when re-running the reverse
// index DDL on deployments that already have idx_graph_edges_to.
func isMySQLDuplicateIndex(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Duplicate key name")
}
