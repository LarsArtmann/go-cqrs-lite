package graph

import (
	"errors"
	"testing"
)

func seedSocialGraph(t *testing.T) *MemoryDriver {
	t.Helper()

	driver := NewMemoryDriver()

	users := []struct {
		id   string
		name string
	}{
		{"u1", "alice"}, {"u2", "bob"}, {"u3", "carol"}, {"u4", "dave"},
	}

	err := driver.RunInTx(func(sink GraphSink) error {
		for _, u := range users {
			if err := sink.MergeNode(
				NodeRef{Label: "User", KeyProp: "id", KeyValue: u.id},
				map[string]any{"name": u.name},
			); err != nil {
				return err
			}
		}

		// alice KNOWS bob, bob KNOWS carol, carol KNOWS dave, alice KNOWS dave (direct).
		edges := []EdgeRef{
			{Type: "KNOWS", From: nodeRef("User", "u1"), To: nodeRef("User", "u2")},
			{Type: "KNOWS", From: nodeRef("User", "u2"), To: nodeRef("User", "u3")},
			{Type: "KNOWS", From: nodeRef("User", "u3"), To: nodeRef("User", "u4")},
			{Type: "KNOWS", From: nodeRef("User", "u1"), To: nodeRef("User", "u4")},
		}

		for _, e := range edges {
			if err := sink.MergeEdge(e, map[string]any{"weight": 1}); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		t.Fatalf("seed graph: %v", err)
	}

	return driver
}

func TestQuery_AllNodesOfLabel(t *testing.T) {
	t.Parallel()

	driver := seedSocialGraph(t)

	result := driver.Query(Pattern{Label: "User"})
	if len(result) != 4 {
		t.Fatalf("expected 4 users, got %d", len(result))
	}
}

func TestQuery_WithPredicate(t *testing.T) {
	t.Parallel()

	driver := seedSocialGraph(t)

	result := driver.Query(Pattern{
		Label: "User",
		Where: func(props map[string]any) bool {
			name, _ := props["name"].(string)

			return name == "alice"
		},
	})
	if len(result) != 1 {
		t.Fatalf("expected 1 node, got %d", len(result))
	}

	if result[0].Props["name"] != "alice" {
		t.Fatalf("expected alice, got %v", result[0].Props["name"])
	}
}

func TestQuery_AllLabels(t *testing.T) {
	t.Parallel()

	driver := seedSocialGraph(t)

	result := driver.Query(Pattern{})
	if len(result) != 4 {
		t.Fatalf("expected 4 nodes (all labels), got %d", len(result))
	}
}

func TestQuery_EmptyGraph(t *testing.T) {
	t.Parallel()

	driver := NewMemoryDriver()
	result := driver.Query(Pattern{Label: "User"})
	if len(result) != 0 {
		t.Fatalf("expected 0 nodes, got %d", len(result))
	}
}

func TestQuery_PropsAreDefensiveCopy(t *testing.T) {
	t.Parallel()

	driver := seedSocialGraph(t)

	result := driver.Query(Pattern{
		Label: "User",
		Where: func(props map[string]any) bool { return props["name"] == "alice" },
	})
	if len(result) != 1 {
		t.Fatalf("expected 1 node, got %d", len(result))
	}

	result[0].Props["name"] = "MUTATED"

	again := driver.Query(Pattern{
		Label: "User",
		Where: func(props map[string]any) bool { return props["name"] == "alice" },
	})
	if len(again) != 1 {
		t.Fatalf("mutation leaked: expected alice to still exist, got %d results", len(again))
	}
}

func TestTraverse_ImmediateNeighbors(t *testing.T) {
	t.Parallel()

	driver := seedSocialGraph(t)

	// alice KNOWS bob and dave directly.
	result := driver.Traverse(nodeRef("User", "u1"), "KNOWS", 0)
	if len(result) != 2 {
		t.Fatalf("expected 2 immediate neighbors, got %d", len(result))
	}
}

func TestTraverse_TwoHops(t *testing.T) {
	t.Parallel()

	driver := seedSocialGraph(t)

	// alice → bob → carol (and alice → dave directly at depth 0).
	// At depth 1: bob, dave. At depth 2: carol (via bob).
	result := driver.Traverse(nodeRef("User", "u1"), "KNOWS", 1)
	if len(result) != 3 {
		t.Fatalf("expected 3 nodes within 2 hops, got %d", len(result))
	}
}

func TestTraverse_Unlimited(t *testing.T) {
	t.Parallel()

	driver := seedSocialGraph(t)

	result := driver.Traverse(nodeRef("User", "u1"), "KNOWS", -1)
	if len(result) != 3 {
		t.Fatalf("expected 3 reachable nodes, got %d", len(result))
	}
}

func TestTraverse_MissingStartNode(t *testing.T) {
	t.Parallel()

	driver := seedSocialGraph(t)

	result := driver.Traverse(nodeRef("User", "nonexistent"), "KNOWS", -1)
	if len(result) != 0 {
		t.Fatalf("expected 0 nodes for missing start, got %d", len(result))
	}
}

func TestTraverse_HandlesCycles(t *testing.T) {
	t.Parallel()

	driver := NewMemoryDriver()

	_ = driver.RunInTx(func(sink GraphSink) error {
		_ = sink.MergeNode(nodeRef("User", "a"), nil)
		_ = sink.MergeNode(nodeRef("User", "b"), nil)

		_ = sink.MergeEdge(EdgeRef{Type: "KNOWS", From: nodeRef("User", "a"), To: nodeRef("User", "b")}, nil)

		return sink.MergeEdge(EdgeRef{Type: "KNOWS", From: nodeRef("User", "b"), To: nodeRef("User", "a")}, nil)
	})

	result := driver.Traverse(nodeRef("User", "a"), "KNOWS", -1)
	if len(result) != 1 {
		t.Fatalf("expected 1 reachable node (cycle-safe), got %d", len(result))
	}
}

func TestNeighbors_ReturnsNodesAndEdges(t *testing.T) {
	t.Parallel()

	driver := seedSocialGraph(t)

	nodes, edges := driver.Neighbors(nodeRef("User", "u1"))
	if len(nodes) != 2 {
		t.Fatalf("expected 2 neighbor nodes, got %d", len(nodes))
	}

	if len(edges) != 2 {
		t.Fatalf("expected 2 incident edges, got %d", len(edges))
	}
}

func TestNeighbors_MissingNode(t *testing.T) {
	t.Parallel()

	driver := seedSocialGraph(t)

	nodes, edges := driver.Neighbors(nodeRef("User", "nonexistent"))
	if nodes != nil || edges != nil {
		t.Fatalf("expected nil for missing node")
	}
}

func TestShortestPath_Direct(t *testing.T) {
	t.Parallel()

	driver := seedSocialGraph(t)

	// alice → dave (direct edge).
	path, err := driver.ShortestPath(nodeRef("User", "u1"), nodeRef("User", "u4"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(path) != 2 {
		t.Fatalf("expected path length 2 (alice → dave), got %d", len(path))
	}
}

func TestShortestPath_TwoHops(t *testing.T) {
	t.Parallel()

	driver := seedSocialGraph(t)

	// alice → bob → carol (shorter than alice → dave ← carol).
	path, err := driver.ShortestPath(nodeRef("User", "u1"), nodeRef("User", "u3"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(path) != 3 {
		t.Fatalf("expected path length 3 (alice → bob → carol), got %d", len(path))
	}
}

func TestShortestPath_SameNode(t *testing.T) {
	t.Parallel()

	driver := seedSocialGraph(t)

	path, err := driver.ShortestPath(nodeRef("User", "u1"), nodeRef("User", "u1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(path) != 1 {
		t.Fatalf("expected path length 1 (same node), got %d", len(path))
	}
}

func TestShortestPath_NotFound(t *testing.T) {
	t.Parallel()

	driver := NewMemoryDriver()

	_ = driver.RunInTx(func(sink GraphSink) error {
		_ = sink.MergeNode(nodeRef("User", "a"), nil)
		_ = sink.MergeNode(nodeRef("User", "b"), nil)

		return nil
	})

	_, err := driver.ShortestPath(nodeRef("User", "a"), nodeRef("User", "b"))
	if !errors.Is(err, ErrPathNotFound) {
		t.Fatalf("expected ErrPathNotFound, got %v", err)
	}
}
