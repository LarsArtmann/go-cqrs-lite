package metaengine

import (
	"context"
	"testing"
)

// multimapOnlyEngine wraps a memoryEngine's multimap operations but does NOT
// expose GraphAddEdge/GraphNeighbors, forcing the fallback path.
type multimapOnlyEngine struct {
	inner *memoryEngine
}

func newMultimapOnlyEngine() *multimapOnlyEngine {
	return &multimapOnlyEngine{inner: NewMemoryEngine().(*memoryEngine)}
}

func (m *multimapOnlyEngine) Profile() EngineProfile {
	p := m.inner.Profile()
	p.Name = "multimap_only"
	return p
}

func (m *multimapOnlyEngine) Close() error { return m.inner.Close() }

func (m *multimapOnlyEngine) MultiAdd(
	ctx context.Context,
	col string,
	key any,
	val any,
) error {
	return m.inner.MultiAdd(ctx, col, key, val)
}

func (m *multimapOnlyEngine) MultiGet(
	ctx context.Context,
	col string,
	key any,
) ([]any, error) {
	return m.inner.MultiGet(ctx, col, key)
}

func TestGraphFallback_AddEdgeAndNeighbors(t *testing.T) {
	t.Parallel()

	eng := newMultimapOnlyEngine()
	ctx := context.Background()

	// Build a simple graph: A → B → C, A → C
	edges := []Edge{
		{From: "A", To: "B"},
		{From: "B", To: "C"},
		{From: "A", To: "C"},
	}

	for _, e := range edges {
		if err := graphAddEdgeFallback(ctx, eng, "test_graph", e); err != nil {
			t.Fatalf("graphAddEdgeFallback %v: %v", e, err)
		}
	}

	// Depth-1 traversal from A should find B and C.
	neighbors, err := graphNeighborsFallback(ctx, eng, "test_graph", "A", 1)
	if err != nil {
		t.Fatalf("graphNeighborsFallback depth 1: %v", err)
	}

	if len(neighbors) != 2 {
		t.Fatalf("depth-1 neighbors: expected 2, got %d: %v", len(neighbors), neighbors)
	}

	seen := map[string]bool{}
	for _, n := range neighbors {
		seen[n.(string)] = true
	}
	if !seen["B"] || !seen["C"] {
		t.Errorf("expected neighbors B and C, got %v", neighbors)
	}
}

func TestGraphFallback_Depth2Traversal(t *testing.T) {
	t.Parallel()

	eng := newMultimapOnlyEngine()
	ctx := context.Background()

	// Linear chain: A → B → C → D
	chain := []Edge{
		{From: "A", To: "B"},
		{From: "B", To: "C"},
		{From: "C", To: "D"},
	}

	for _, e := range chain {
		_ = graphAddEdgeFallback(ctx, eng, "chain", e)
	}

	// Depth-2 from A should reach B (level 1) and C (level 2), but not D.
	neighbors, err := graphNeighborsFallback(ctx, eng, "chain", "A", 2)
	if err != nil {
		t.Fatalf("depth-2 traversal: %v", err)
	}

	seen := map[string]bool{}
	for _, n := range neighbors {
		seen[n.(string)] = true
	}

	if !seen["B"] {
		t.Error("expected B in depth-2 neighbors")
	}
	if !seen["C"] {
		t.Error("expected C in depth-2 neighbors")
	}
	if seen["D"] {
		t.Error("D should NOT appear in depth-2 neighbors from A")
	}
}

func TestGraphFallback_NoCycles(t *testing.T) {
	t.Parallel()

	eng := newMultimapOnlyEngine()
	ctx := context.Background()

	// Cyclic graph: A → B → A
	_ = graphAddEdgeFallback(ctx, eng, "cycle", Edge{From: "A", To: "B"})
	_ = graphAddEdgeFallback(ctx, eng, "cycle", Edge{From: "B", To: "A"})

	// Depth-5 should not loop infinitely or return duplicates.
	neighbors, err := graphNeighborsFallback(ctx, eng, "cycle", "A", 5)
	if err != nil {
		t.Fatalf("cyclic traversal: %v", err)
	}

	// Should only contain B (A is visited, cycle terminates).
	if len(neighbors) != 1 {
		t.Fatalf("expected 1 neighbor (B), got %d: %v", len(neighbors), neighbors)
	}
}

func TestGraphFallback_Depth0(t *testing.T) {
	t.Parallel()

	eng := newMultimapOnlyEngine()
	ctx := context.Background()

	_ = graphAddEdgeFallback(ctx, eng, "g", Edge{From: "X", To: "Y"})

	neighbors, err := graphNeighborsFallback(ctx, eng, "g", "X", 0)
	if err != nil {
		t.Fatalf("depth-0: %v", err)
	}

	if len(neighbors) != 0 {
		t.Errorf("depth-0 should return no neighbors, got %d", len(neighbors))
	}
}
