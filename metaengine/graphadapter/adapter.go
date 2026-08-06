// Package graphadapter bridges the rich graph.GraphDriver API to the
// metaengine.Engine interface, making graph-shaped read models available to
// the cost-based planner (ADR-0113).
//
// The adapter wraps a graph.MemoryDriver and implements both metaengine.Engine
// (Profile + Close) and metaengine.GraphBackend (GraphAddEdge +
// GraphNeighbors). Simple Edge{From, To} folds from planner queries are
// synthesized into graph.MergeEdge calls with auto-created NodeRefs.
package graphadapter

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/graph/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Adapter wraps a graph.MemoryDriver as a metaengine.Engine.
type Adapter struct {
	driver *graph.MemoryDriver
}

var (
	_ metaengine.Engine       = (*Adapter)(nil)
	_ metaengine.GraphBackend = (*Adapter)(nil)
)

// New creates a graphadapter backed by a fresh graph.MemoryDriver.
func New() *Adapter {
	return &Adapter{driver: graph.NewMemoryDriver()}
}

// NewWithDriver creates a graphadapter wrapping an existing MemoryDriver.
// The adapter takes ownership of the driver's lifecycle (Close delegates to it).
func NewWithDriver(driver *graph.MemoryDriver) *Adapter {
	return &Adapter{driver: driver}
}

// Driver returns the underlying graph.MemoryDriver for direct graph queries
// (Traverse, Neighbors, ShortestPath via the ReadableDriver interface).
func (a *Adapter) Driver() *graph.MemoryDriver { return a.driver }

// Profile returns the engine profile declaring ADTGraph support at O(N).
func (a *Adapter) Profile() metaengine.EngineProfile {
	return metaengine.EngineProfile{
		Name: "graph-memory",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTGraph: metaengine.ComplexityON,
		},
		NsPerOp: 3000,
	}
}

// Close delegates to the underlying driver.
func (a *Adapter) Close() error { return a.driver.Close() }

// GraphAddEdge synthesizes a graph.MergeEdge call from a simple metaengine.Edge.
// NodeRefs are auto-created from the From/To values using a generic label.
func (a *Adapter) GraphAddEdge(_ context.Context, collection string, edge metaengine.Edge) error {
	fromRef := graph.NodeRef{Label: "entity", KeyProp: "id", KeyValue: fmt.Sprint(edge.From)}
	toRef := graph.NodeRef{Label: "entity", KeyProp: "id", KeyValue: fmt.Sprint(edge.To)}

	return a.driver.RunInTx(func(sink graph.GraphSink) error {
		return sink.MergeEdge(graph.EdgeRef{
			Type: collection,
			From: fromRef,
			To:   toRef,
		}, nil)
	})
}

// GraphNeighbors traverses the graph from the given node, returning neighbor IDs.
func (a *Adapter) GraphNeighbors(_ context.Context, collection string, node any, depth int) ([]any, error) {
	nodeRef := graph.NodeRef{Label: "entity", KeyProp: "id", KeyValue: fmt.Sprint(node)}
	nodes := a.driver.Traverse(nodeRef, collection, depth)

	result := make([]any, len(nodes))
	for i, n := range nodes {
		result[i] = n.Ref.KeyValue
	}

	return result, nil
}
