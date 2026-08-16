package mysqlengine

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"
	"strconv"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- graph dispatch ---
//
// MySQL/MariaDB implements graph traversal natively on meta_graph_edges in
// two modes, chosen automatically at construction time:
//
//   - recursive CTE: a single WITH RECURSIVE statement (MySQL 8.0+,
//     MariaDB 10.2+) walks the depth-limited neighborhood in one query.
//   - iterative BFS: one indexed SELECT per node per level
//     (WHERE collection = ? AND from_node = ?). Servers without WITH
//     RECURSIVE — e.g. MySQL 5.7, which rejects the syntax with Error 1064 —
//     are probed at construction and degrade to this path instead of
//     failing every graph read.
//
// Both modes use the index on (collection, from_node) so each hop is an
// index lookup; both beat the degraded multimap BFS fallback
// (O(N * degree^depth)) that non-graph engines take.

// mysqlGraphNeighborsCTE walks the depth-limited neighborhood in one query.
// UNION deduplicates (node, depth) pairs; SELECT DISTINCT collapses a node
// reached at multiple depths; the outer WHERE excludes the start node so
// cycles never re-admit it.
const mysqlGraphNeighborsCTE = `WITH RECURSIVE walk(node, depth) AS (
	SELECT to_node, 1 FROM meta_graph_edges WHERE collection = ? AND from_node = ?
	UNION
	SELECT g.to_node, w.depth + 1
	FROM meta_graph_edges g JOIN walk w ON g.collection = ? AND g.from_node = w.node
	WHERE w.depth < ?
)
SELECT DISTINCT node FROM walk WHERE node <> ?`

// mysqlGraphNeighborsDirect is the per-hop adjacency lookup used by the
// iterative fallback when the server lacks WITH RECURSIVE.
const mysqlGraphNeighborsDirect = `SELECT to_node FROM meta_graph_edges WHERE collection = ? AND from_node = ?`

// mysqlCTEProbeSQL verifies the server executes WITH RECURSIVE. Any error
// (unsupported syntax, restricted proxy) disables the CTE path.
const mysqlCTEProbeSQL = `WITH RECURSIVE cqrs_cte_probe(x) AS (
	SELECT 1 UNION ALL SELECT x+1 FROM cqrs_cte_probe WHERE x < 1
) SELECT x FROM cqrs_cte_probe`

// probeRecursiveCTE reports whether the server executes recursive CTEs.
func probeRecursiveCTE(db *sql.DB) bool {
	var got int
	return db.QueryRowContext(context.Background(), mysqlCTEProbeSQL).Scan(&got) == nil
}

// GraphAddEdge inserts an edge into meta_graph_edges. INSERT IGNORE makes
// duplicate edges idempotent (replay-safe).
func (e *mysqlEngine) GraphAddEdge(
	ctx context.Context,
	col string,
	edge metaengine.Edge,
) error {
	const q = `INSERT IGNORE INTO meta_graph_edges (collection, from_node, to_node) VALUES (?, ?, ?)`

	if _, err := e.conn().
		ExecContext(ctx, q, col, encodeNodeKey(edge.From), encodeNodeKey(edge.To)); err != nil {
		return fmt.Errorf("mysqlengine.GraphAddEdge: %w", err)
	}

	return nil
}

// GraphNeighbors returns all nodes within depth hops of node (excluding
// node itself), deduplicated. Servers with WITH RECURSIVE resolve the walk
// in one query; older servers (MySQL 5.7, MariaDB <10.2) fall back to an
// iterative BFS over the indexed edges table.
func (e *mysqlEngine) GraphNeighbors(
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
func (e *mysqlEngine) graphNeighborsCTE(
	ctx context.Context,
	col string,
	node any,
	depth int,
) ([]any, error) {
	start := encodeNodeKey(node)

	rows, err := e.conn().QueryContext(ctx, mysqlGraphNeighborsCTE, col, start, col, depth, start)
	if err != nil {
		return nil, fmt.Errorf("mysqlengine.GraphNeighbors: %w", err)
	}
	defer metaengine.DeferClose(rows)

	var result []any
	for rows.Next() {
		var nb string
		if err := rows.Scan(&nb); err != nil {
			return nil, fmt.Errorf("mysqlengine.GraphNeighbors: row: %w", err)
		}

		result = append(result, nb)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysqlengine.GraphNeighbors: %w", err)
	}

	if result == nil {
		result = []any{}
	}

	return result, nil
}

// graphWalk is the shared iterative BFS skeleton for graph neighborhood
// reads: one adjacency lookup per node per level, visited-set dedup, and a
// non-nil result. The adjacency callback selects directed or both-direction
// edges; servers without WITH RECURSIVE take this path for both entry points.
func (e *mysqlEngine) graphWalk(
	ctx context.Context,
	col string,
	node any,
	depth int,
	adjacency func(ctx context.Context, col, node string) ([]string, error),
) ([]any, error) {
	startNode := encodeNodeKey(node)
	visited := map[string]bool{startNode: true}
	frontier := []string{startNode}
	var result []any

	for level := 0; level < depth && len(frontier) > 0; level++ {
		var next []string

		for _, n := range frontier {
			neighbors, err := adjacency(ctx, col, n)
			if err != nil {
				return nil, fmt.Errorf("mysqlengine.graphWalk: %w", err)
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

// graphNeighborsIterative is the fallback for servers without WITH
// RECURSIVE: one indexed lookup per node per level.
func (e *mysqlEngine) graphNeighborsIterative(
	ctx context.Context,
	col string,
	node any,
	depth int,
) ([]any, error) {
	return e.graphWalk(ctx, col, node, depth, e.queryGraphNeighbors) //nolint:wrapcheck
}

// queryGraphNeighbors reads the direct adjacency of one node.
func (e *mysqlEngine) queryGraphNeighbors(
	ctx context.Context,
	col, node string,
) ([]string, error) {
	rows, err := e.conn().QueryContext(ctx, mysqlGraphNeighborsDirect, col, node)
	if err != nil {
		return nil, err
	}
	defer metaengine.DeferClose(rows)

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

// encodeNodeKey renders a graph node key as the VARCHAR stored in
// meta_graph_edges. Mirrors sqliteengine's encoding so engines agree on the
// canonical node representation: strings stay raw, integers render decimally,
// anything else falls back to JSON.
// art-dupl:accept cross-module SQL engine pattern — separate go.mod
func encodeNodeKey(key any) string {
	switch k := key.(type) {
	case string:
		return k
	case int:
		return strconv.Itoa(k)
	case int64:
		return strconv.FormatInt(k, 10)
	case int32:
		return strconv.FormatInt(int64(k), 10)
	case uint64:
		return strconv.FormatUint(k, 10)
	case uint32:
		return strconv.FormatUint(uint64(k), 10)
	default:
		return encodeJSONString(key)
	}
}

// encodeJSONString marshals v to JSON, falling back to fmt.Sprintf("%v")
// when v is not JSON-serializable (same fallback as sqliteengine.encodeJSON).
func encodeJSONString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}

	return string(b)
}
