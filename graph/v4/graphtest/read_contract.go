package graphtest

import (
	"errors"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/graph/v4"
)

// test-graph expected counts — named to silence mnd.
const (
	seedNodesAB = 3 // total nodes in seed graph: a, b, c
	seedABTrail = 2 // reachable from a (b and c, via KNOWS)
	neighborsB  = 2 // neighbors of b (a inbound, c outbound)
	shortPathAC = 2 // a→c is 2 hops (a→b→c or a→c direct)
)

// SeedReadGraph builds a known test graph for ReadableDriver contract tests:
//
//	a --KNOWS--> b --KNOWS--> c
//	 \                        ^
//	  ----KNOWS---------------/
//
// Query, Traverse, Neighbors, and ShortestPath are tested against this shape.
func SeedReadGraph(t *testing.T, driver graph.GraphDriver) {
	t.Helper()

	err := driver.RunInTx(func(sink graph.GraphSink) error {
		if err := sink.MergeNode(
			nodeRef("a"),
			map[string]any{propName: valueAlice},
		); err != nil {
			return fmt.Errorf("seed node a: %w", err)
		}

		if err := sink.MergeNode(
			nodeRef("b"),
			map[string]any{propName: "bob"},
		); err != nil {
			return fmt.Errorf("seed node b: %w", err)
		}

		if err := sink.MergeNode(
			nodeRef("c"),
			map[string]any{propName: "carol"},
		); err != nil {
			return fmt.Errorf("seed node c: %w", err)
		}

		if err := sink.MergeEdge(graph.EdgeRef{
			Type: typeKnows, From: nodeRef("a"), To: nodeRef("b"),
		}, nil); err != nil {
			return fmt.Errorf("seed edge a→b: %w", err)
		}

		if err := sink.MergeEdge(graph.EdgeRef{
			Type: typeKnows, From: nodeRef("b"), To: nodeRef("c"),
		}, nil); err != nil {
			return fmt.Errorf("seed edge b→c: %w", err)
		}

		return sink.MergeEdge(graph.EdgeRef{
			Type: typeKnows, From: nodeRef("a"), To: nodeRef("c"),
		}, nil)
	})
	if err != nil {
		t.Fatalf("seed read graph: %v", err)
	}
}

func testReadQuery(t *testing.T, readable graph.ReadableDriver) {
	t.Helper()

	results := readable.Query(graph.Pattern{Label: labelUser, Where: nil})
	if len(results) != seedNodesAB {
		t.Fatalf("Query(User): expected 3 nodes, got %d", len(results))
	}

	filtered := readable.Query(graph.Pattern{
		Label: labelUser,
		Where: func(props map[string]any) bool {
			name, _ := props[propName].(string)

			return name == valueAlice
		},
	})
	if len(filtered) != 1 {
		t.Fatalf("Query(predicate): expected 1 node, got %d", len(filtered))
	}
}

func testReadTraverse(t *testing.T, readable graph.ReadableDriver) {
	t.Helper()

	visited := readable.Traverse(nodeRef("a"), typeKnows, -1)
	if len(visited) != seedABTrail {
		t.Fatalf("Traverse(a, KNOWS): expected 2 reachable, got %d", len(visited))
	}

	direct := readable.Traverse(nodeRef("a"), typeKnows, 1)
	if len(direct) != seedABTrail {
		t.Fatalf("Traverse(a, KNOWS, depth=1): expected 2, got %d", len(direct))
	}
}

func testReadNeighbors(t *testing.T, readable graph.ReadableDriver) {
	t.Helper()

	nodes, edges := readable.Neighbors(nodeRef("b"))
	if len(nodes) < neighborsB {
		t.Fatalf("Neighbors(b): expected at least 2 nodes (a, c), got %d", len(nodes))
	}

	if len(edges) < neighborsB {
		t.Fatalf("Neighbors(b): expected at least 2 edges, got %d", len(edges))
	}
}

func testReadShortestPath(t *testing.T, readable graph.ReadableDriver) {
	t.Helper()

	path, err := readable.ShortestPath(nodeRef("a"), nodeRef("c"))
	if err != nil {
		t.Fatalf("ShortestPath(a,c): %v", err)
	}

	if len(path) != shortPathAC {
		t.Fatalf("ShortestPath(a,c): expected 2 hops, got %d: %v", len(path), path)
	}

	// ShortestPath treats edges as bidirectional (property-graph default).
	reverse, err := readable.ShortestPath(nodeRef("c"), nodeRef("a"))
	if err != nil {
		t.Fatalf("ShortestPath(c,a): %v", err)
	}

	if len(reverse) != shortPathAC {
		t.Fatalf("ShortestPath(c,a): expected 2 hops, got %d: %v", len(reverse), reverse)
	}

	// No path to a non-existent node.
	_, err = readable.ShortestPath(nodeRef("a"), nodeRef("zzz"))
	if !errors.Is(err, graph.ErrPathNotFound) {
		t.Fatalf("ShortestPath(a,zzz): expected ErrPathNotFound, got %v", err)
	}
}
