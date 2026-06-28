package graphtest

import (
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/graph/v3"
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
		if err := sink.MergeNode(nodeRef("User", "a"), map[string]any{"name": "alice"}); err != nil {
			return err
		}

		if err := sink.MergeNode(nodeRef("User", "b"), map[string]any{"name": "bob"}); err != nil {
			return err
		}

		if err := sink.MergeNode(nodeRef("User", "c"), map[string]any{"name": "carol"}); err != nil {
			return err
		}

		if err := sink.MergeEdge(graph.EdgeRef{
			Type: "KNOWS", From: nodeRef("User", "a"), To: nodeRef("User", "b"),
		}, nil); err != nil {
			return err
		}

		if err := sink.MergeEdge(graph.EdgeRef{
			Type: "KNOWS", From: nodeRef("User", "b"), To: nodeRef("User", "c"),
		}, nil); err != nil {
			return err
		}

		return sink.MergeEdge(graph.EdgeRef{
			Type: "KNOWS", From: nodeRef("User", "a"), To: nodeRef("User", "c"),
		}, nil)
	})
	if err != nil {
		t.Fatalf("seed read graph: %v", err)
	}
}

func testReadQuery(t *testing.T, readable graph.ReadableDriver) {
	t.Helper()

	results := readable.Query(graph.Pattern{Label: "User"})
	if len(results) != 3 {
		t.Fatalf("Query(User): expected 3 nodes, got %d", len(results))
	}

	filtered := readable.Query(graph.Pattern{
		Label: "User",
		Where: func(props map[string]any) bool {
			name, _ := props["name"].(string)

			return name == "alice"
		},
	})
	if len(filtered) != 1 {
		t.Fatalf("Query(predicate): expected 1 node, got %d", len(filtered))
	}
}

func testReadTraverse(t *testing.T, readable graph.ReadableDriver) {
	t.Helper()

	visited := readable.Traverse(nodeRef("User", "a"), "KNOWS", -1)
	if len(visited) != 2 {
		t.Fatalf("Traverse(a, KNOWS): expected 2 reachable, got %d", len(visited))
	}

	direct := readable.Traverse(nodeRef("User", "a"), "KNOWS", 1)
	if len(direct) != 2 {
		t.Fatalf("Traverse(a, KNOWS, depth=1): expected 2, got %d", len(direct))
	}
}

func testReadNeighbors(t *testing.T, readable graph.ReadableDriver) {
	t.Helper()

	nodes, edges := readable.Neighbors(nodeRef("User", "b"))
	if len(nodes) < 2 {
		t.Fatalf("Neighbors(b): expected at least 2 nodes (a, c), got %d", len(nodes))
	}

	if len(edges) < 2 {
		t.Fatalf("Neighbors(b): expected at least 2 edges, got %d", len(edges))
	}
}

func testReadShortestPath(t *testing.T, readable graph.ReadableDriver) {
	t.Helper()

	path, err := readable.ShortestPath(nodeRef("User", "a"), nodeRef("User", "c"))
	if err != nil {
		t.Fatalf("ShortestPath(a,c): %v", err)
	}

	if len(path) != 2 {
		t.Fatalf("ShortestPath(a,c): expected 2 hops, got %d: %v", len(path), path)
	}

	// ShortestPath treats edges as bidirectional (property-graph default).
	reverse, err := readable.ShortestPath(nodeRef("User", "c"), nodeRef("User", "a"))
	if err != nil {
		t.Fatalf("ShortestPath(c,a): %v", err)
	}

	if len(reverse) != 2 {
		t.Fatalf("ShortestPath(c,a): expected 2 hops, got %d: %v", len(reverse), reverse)
	}

	// No path to a non-existent node.
	_, err = readable.ShortestPath(nodeRef("User", "a"), nodeRef("User", "zzz"))
	if !errors.Is(err, graph.ErrPathNotFound) {
		t.Fatalf("ShortestPath(a,zzz): expected ErrPathNotFound, got %v", err)
	}
}
