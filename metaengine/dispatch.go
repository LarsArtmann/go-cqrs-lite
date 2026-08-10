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

// HasGraphSupport returns true if the engine implements the graph dispatch
// contract (GraphAddEdge + GraphNeighbors). This is the exported capability
// check — consumers should not implement the interface directly; use
// graphadapter.Adapter instead (ADR-0113).
func HasGraphSupport(eng Engine) bool {
	_, ok := eng.(graphBackend)
	return ok
}
