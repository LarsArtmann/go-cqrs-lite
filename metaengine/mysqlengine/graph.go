package mysqlengine

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"strconv"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- graph dispatch ---
//
// MySQL/MariaDB implements graph traversal natively on meta_graph_edges via
// a single WITH RECURSIVE statement (MySQL 8.0+, MariaDB 10.2+): the
// depth-limited neighborhood resolves in one query instead of the degraded
// multimap BFS fallback (O(N * degree^depth)). The index on
// (collection, from_node) makes each expansion step an index lookup.

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
// node itself), deduplicated, via a recursive CTE.
func (e *mysqlEngine) GraphNeighbors(
	ctx context.Context,
	col string,
	node any,
	depth int,
) ([]any, error) {
	if depth <= 0 {
		return []any{}, nil
	}

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
