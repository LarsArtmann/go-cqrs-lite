package metaengine

import (
	"context"
	"fmt"
)

// graphAddEdgeFallback stores a graph edge via MultimapBackend when the engine
// does not implement graphBackend natively. This is the degraded path: edges
// are stored as multimap entries (From → To), and traversal requires iterative
// MultiGet calls (O(N * degree^depth) instead of O(degree^depth)).
//
// The planner emits a DiagLevelDegraded diagnostic when this path is taken.
func graphAddEdgeFallback(
	ctx context.Context,
	eng Engine,
	col string,
	edge Edge,
) error {
	mb, ok := eng.(MultimapBackend)
	if !ok {
		return unsupportedEngine(errUnsupportedGraphOps, eng.Profile().Name)
	}

	if err := mb.MultiAdd(ctx, col, edge.From, edge.To); err != nil {
		return fmt.Errorf("graph fallback add edge %s: %w", col, err)
	}

	return nil
}

// graphNeighborsFallback performs BFS traversal via MultimapBackend when the
// engine does not implement graphBackend natively. Each level of the traversal
// issues one MultiGet per node, making this O(N * degree^depth) — functional
// but slow compared to native graph backends. Delegates to GraphBFSNodes
// with typedNodeKey as the dedup key, so returned neighbors keep their
// original dynamic types (the string-keyed GraphBFS is the SQL-engine form).
// Result is never nil (the shared nil-vs-empty contract, 2026-10-01).
func graphNeighborsFallback(
	ctx context.Context,
	eng Engine,
	col string,
	node any,
	depth int,
) ([]any, error) {
	mb, ok := eng.(MultimapBackend)
	if !ok {
		return nil, unsupportedEngine(errUnsupportedGraphReads, eng.Profile().Name)
	}

	//nolint:wrapcheck // GraphBFSNodes wraps expand errors with the label prefix
	return GraphBFSNodes(
		ctx,
		node,
		depth,
		"metaengine.graphNeighborsFallback",
		typedNodeKey,
		func(ctx context.Context, n any) ([]any, error) {
			neighbors, err := mb.MultiGet(ctx, col, n)
			if err != nil {
				return nil, fmt.Errorf("graph fallback neighbors %s: %w", col, err)
			}

			return neighbors, nil
		},
	)
}

// typedNodeKey builds a dedup key that includes the dynamic type, so
// mixed-typed nodes never collide (int(1) and "1" are distinct nodes).
func typedNodeKey(node any) string {
	return fmt.Sprintf("%[1]T:%[1]v", node)
}
