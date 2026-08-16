package sqliteengine

import (
	"context"
	"database/sql"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- graph dispatch ---
//
// SQLite implements graph traversal natively on meta_graph_edges in two
// modes, chosen automatically at construction time:
//
//   - recursive CTE: a single WITH RECURSIVE statement walks the whole
//     depth-limited neighborhood in one query. This is much faster for deep
//     graphs than per-level round trips (O(1) queries instead of
//     O(nodes-per-level)).
//   - iterative BFS: one indexed SELECT per node per level
//     (WHERE collection = ? AND from_node = ?). Used when the driver or
//     server does not support WITH RECURSIVE — probed at construction, so
//     libSQL/Turso deployments that lack recursive CTEs degrade gracefully
//     instead of failing.
//
// Both modes are O(degree^depth) lookups — far better than the degraded
// multimap BFS fallback (O(N * degree^depth)). The dedicated edges table
// with a composite index ensures each level lookup is O(logN).

const graphNeighborsDirectSQL = `SELECT to_node FROM meta_graph_edges WHERE collection = ? AND from_node = ?`

// graphNeighborsReverseSQL expands against the reverse direction (incoming
// edges): used by undirected traversal. Served by idx_graph_edges_to.
const graphNeighborsReverseSQL = `SELECT from_node FROM meta_graph_edges WHERE collection = ? AND to_node = ?`

// graphNeighborsCTE walks the depth-limited neighborhood in one query.
// UNION deduplicates (node, depth) pairs; SELECT DISTINCT collapses a node
// reached at multiple depths; the outer WHERE excludes the start node (the
// iterative BFS marks it visited, and cycles would otherwise re-admit it).
const graphNeighborsCTE = `WITH RECURSIVE walk(node, depth) AS (
	SELECT to_node, 1 FROM meta_graph_edges WHERE collection = ? AND from_node = ?
	UNION
	SELECT g.to_node, w.depth + 1
	FROM meta_graph_edges g JOIN walk w ON g.collection = ? AND g.from_node = w.node
	WHERE w.depth < ?
)
SELECT DISTINCT node FROM walk WHERE node <> ?`

// cteProbeSQL verifies the driver/server executes WITH RECURSIVE. Any error
// (unsupported syntax, restricted remote protocol) disables the CTE path.
const cteProbeSQL = `WITH RECURSIVE cqrs_cte_probe(x) AS (
	SELECT 1 UNION ALL SELECT x+1 FROM cqrs_cte_probe WHERE x < 1
) SELECT x FROM cqrs_cte_probe`

// probeRecursiveCTE reports whether the database executes recursive CTEs.
func probeRecursiveCTE(db *sql.DB) bool {
	var got int
	return db.QueryRowContext(context.Background(), cteProbeSQL).Scan(&got) == nil
}

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

	if e.graphCTE {
		return e.graphNeighborsCTE(ctx, col, node, depth)
	}

	return e.graphNeighborsIterative(ctx, col, node, depth)
}

// graphNeighborsCTE resolves the depth-limited neighborhood in a single
// recursive-CTE query.
func (e *sqliteEngine) graphNeighborsCTE(
	ctx context.Context,
	col string,
	node any,
	depth int,
) ([]any, error) {
	start := encodeKey(node)
	rows, err := e.xd().QueryContext(ctx, graphNeighborsCTE, col, start, col, depth, start)
	if err != nil {
		return nil, fmt.Errorf("sqliteengine.GraphNeighbors: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []any
	for rows.Next() {
		var nb string
		if err := rows.Scan(&nb); err != nil {
			return nil, fmt.Errorf("sqliteengine.GraphNeighbors: row: %w", err)
		}

		result = append(result, nb)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqliteengine.GraphNeighbors: %w", err)
	}

	if result == nil {
		result = []any{}
	}

	return result, nil
}

// graphNeighborsIterative is the fallback for drivers without WITH RECURSIVE:
// one indexed lookup per node per level.
func (e *sqliteEngine) graphNeighborsIterative(
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

// GraphRemoveEdge deletes the specific directed edge (ADR-0114 style
// tombstone dispatch). Idempotent: deleting a missing edge affects 0 rows.
func (e *sqliteEngine) GraphRemoveEdge(
	ctx context.Context,
	col string,
	edge metaengine.Edge,
) error {
	const q = `DELETE FROM meta_graph_edges WHERE collection = ? AND from_node = ? AND to_node = ?`

	if _, err := e.xc().exec(ctx, q, col, encodeKey(edge.From), encodeKey(edge.To)); err != nil {
		return fmt.Errorf("sqliteengine.GraphRemoveEdge: %w", err)
	}

	return nil
}

func (e *sqliteEngine) queryGraphNeighbors(
	ctx context.Context,
	col, node string,
) ([]string, error) {
	rows, err := e.xc().query(ctx, graphNeighborsDirectSQL, col, node)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

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

func (e *sqliteEngine) queryGraphReverseNeighbors(
	ctx context.Context,
	col, node string,
) ([]string, error) {
	rows, err := e.xc().query(ctx, graphNeighborsReverseSQL, col, node)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

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
