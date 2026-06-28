package graph

// Pattern describes a declarative node query for [MemoryDriver.Query].
//
// Label filters by node label. An empty Label matches all labels.
//
// Where is a Go-native predicate function that receives a copy of the node's
// properties and returns true if the node should be included. A nil Where
// matches all nodes of the given label.
//
// This is deliberately NOT a query language. TypeDB has TypeQL because it owns
// its engine. go-cqrs-lite is a library that runs against other databases —
// we cannot impose a DSL on backends we don't control. The Go compiler
// type-checks predicate functions; a query parser cannot. For the
// MemoryDriver (which we DO own), this is the portable, composable read
// surface that makes the in-memory graph actually queryable.
type Pattern struct {
	Label string
	Where func(props map[string]any) bool
}

// NodeView is a read-only snapshot of a node returned by read operations.
// Props is a defensive copy — mutating it does not affect the driver.
type NodeView struct {
	Ref   NodeRef
	Props map[string]any
}

// EdgeView is a read-only snapshot of an edge returned by read operations.
type EdgeView struct {
	Ref   EdgeRef
	Props map[string]any
}

// ReadableDriver is implemented by drivers that own their query engine and can
// provide a typed read surface. The [MemoryDriver] implements this because it
// is the reference in-memory engine. External graph databases (Neo4j,
// Memgraph) do NOT implement this — their reads stay native (Cypher/Gremlin),
// per ADR-0038. The asymmetry is documented: writes are portable (MERGE
// semantics), reads are engine-native.
//
// Consumers using the MemoryDriver standalone (tests, single-process apps,
// local-first applications) get a full typed query API. Consumers using a real
// graph database run native queries via their driver.
type ReadableDriver interface {
	// Query returns all nodes matching the pattern.
	Query(p Pattern) []NodeView

	// Traverse performs a breadth-first traversal from the given node,
	// following edges of the specified type, up to maxDepth hops.
	// maxDepth 0 returns the start node's immediate neighbors.
	// maxDepth < 0 means unlimited depth.
	// The start node itself is excluded from the result.
	Traverse(from NodeRef, via string, maxDepth int) []NodeView

	// Neighbors returns all nodes and edges directly connected to the given
	// node, in both directions.
	Neighbors(of NodeRef) ([]NodeView, []EdgeView)

	// ShortestPath finds the shortest path between two nodes via BFS.
	// Returns the path as an ordered slice of NodeRefs (including from and to).
	// Returns ErrPathNotFound if no path exists.
	ShortestPath(from, to NodeRef) ([]NodeRef, error)
}
