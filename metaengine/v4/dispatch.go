package metaengine

import "context"

// graphBackend is the internal dispatch contract for engines that support
// graph operations (AddEdge, Neighbors). It is unexported per ADR-0113:
// consumers use graphadapter.Adapter for graph support, not this interface.
// Engines that happen to have these methods will satisfy the dispatch check.
type graphBackend interface {
	GraphAddEdge(ctx context.Context, collection string, edge Edge) error
	GraphNeighbors(ctx context.Context, collection string, node any, depth int) ([]any, error)
}

// graphEdgeRemover is the optional dispatch contract for edge deletion
// (ADR-0114 style tombstones): an EdgeRemoval fold removes the specific
// directed edge. Kept separate from graphBackend so engines can adopt
// removal incrementally without losing add/read dispatch.
type graphEdgeRemover interface {
	GraphRemoveEdge(ctx context.Context, collection string, edge Edge) error
}

// undirectedGraphBackend is the optional dispatch contract for undirected
// traversal: neighbors reached by following edges in BOTH directions within
// the depth limit. Engines whose storage is already symmetric (dgraph stores
// both directions on add) implement it as an alias of GraphNeighbors.
type undirectedGraphBackend interface {
	GraphNeighborsUndirected(
		ctx context.Context,
		collection string,
		node any,
		depth int,
	) ([]any, error)
}

// HasGraphSupport returns true if the engine implements the graph dispatch
// contract (GraphAddEdge + GraphNeighbors). This is the exported capability
// check — consumers should not implement the interface directly; use
// graphadapter.Adapter instead (ADR-0113).
func HasGraphSupport(eng Engine) bool {
	_, ok := eng.(graphBackend)
	return ok
}

// HasGraphEdgeRemoval returns true if the engine supports deleting graph
// edges (GraphRemoveEdge) — required for EdgeRemoval folds (tombstone-driven
// edge deletion, ADR-0114 style).
func HasGraphEdgeRemoval(eng Engine) bool {
	_, ok := eng.(graphEdgeRemover)
	return ok
}

// HasUndirectedGraphSupport returns true if the engine supports undirected
// graph traversal (GraphNeighborsUndirected).
func HasUndirectedGraphSupport(eng Engine) bool {
	_, ok := eng.(undirectedGraphBackend)
	return ok
}
