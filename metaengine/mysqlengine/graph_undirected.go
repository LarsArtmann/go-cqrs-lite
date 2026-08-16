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
//
// Exactly TWO arms: MySQL rejects recursive references in the non-recursive
// term AND allows the CTE to be referenced at most once in the recursive
// term — the seed's two directions are a derived table and the recursive
// step expands both directions through one CASE + OR join.
const mysqlGraphNeighborsUndirectedCTE = `WITH RECURSIVE walk(node, depth) AS (
	SELECT seed.n, 1 FROM (
		SELECT to_node AS n FROM meta_graph_edges WHERE collection = ? AND from_node = ?
		UNION
		SELECT from_node AS n FROM meta_graph_edges WHERE collection = ? AND to_node = ?
	) seed
	UNION
	SELECT CASE WHEN g.from_node = w.node THEN g.to_node ELSE g.from_node END, w.depth + 1
	FROM meta_graph_edges g
	JOIN walk w ON g.collection = ? AND (g.from_node = w.node OR g.to_node = w.node)
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
	// art-dupl:accept CTE-vs-iterative dispatch mirrored in dep-isolated sqliteengine
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
		col, start, col, start, col, depth, start)
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
// RECURSIVE: one indexed lookup per node per level per direction, via the
// shared graphWalk skeleton.
func (e *mysqlEngine) graphNeighborsUndirectedIterative(
	ctx context.Context,
	col string,
	node any,
	depth int,
) ([]any, error) {
	return e.graphWalk(ctx, col, node, depth, e.graphNeighborsBothDirections) //nolint:wrapcheck
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
	// art-dupl:accept database/sql single-column drain; sqliteengine twin is dep-isolated
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
