package mysqlengine_test

import (
	"context"
	"sort"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// graphBackend is the unexported dispatch contract, re-declared to verify
// structural conformance at compile time.
type graphBackend interface {
	GraphAddEdge(ctx context.Context, collection string, edge metaengine.Edge) error
	GraphNeighbors(ctx context.Context, collection string, node any, depth int) ([]any, error)
}

func sortedGraphNeighbors(items []any) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

func TestMySQLGraph_ImplementsGraphBackend(t *testing.T) {
	t.Parallel()

	eng := mustNewMySQLEngine(t)
	if _, ok := eng.(graphBackend); !ok {
		t.Fatal("mysqlEngine should implement graphBackend (GraphAddEdge + GraphNeighbors)")
	}
}

func TestMySQLGraph_ProfileDeclaresNativeGraph(t *testing.T) {
	t.Parallel()

	eng := mustNewMySQLEngine(t)

	profile := eng.Profile()

	c, ok := profile.Supports[metaengine.ADTGraph]
	if !ok {
		t.Fatal("MySQL profile should declare ADTGraph in Supports")
	}

	if c != metaengine.ComplexityODegree {
		t.Errorf("ADTGraph complexity = %s, want %s (native WITH RECURSIVE)",
			c, metaengine.ComplexityODegree)
	}

	if profile.IsDegraded(metaengine.ADTGraph) {
		t.Error("ADTGraph should NOT be degraded on MySQL (native WITH RECURSIVE)")
	}
}

func TestMySQLGraph_NeighborsCTE(t *testing.T) {
	t.Parallel()

	eng := mustNewMySQLEngine(t)
	gb := eng.(graphBackend)

	ctx := context.Background()
	col := "test_graph_mysql_cte"

	edges := []metaengine.Edge{
		{From: "A", To: "B"},
		{From: "A", To: "C"},
		{From: "B", To: "D"},
		{From: "C", To: "D"},
		{From: "D", To: "A"}, // cycle back to start
	}
	for _, e := range edges {
		if err := gb.GraphAddEdge(ctx, col, e); err != nil {
			t.Fatalf("GraphAddEdge %v: %v", e, err)
		}
	}

	// Depth 1: B and C.
	d1, err := gb.GraphNeighbors(ctx, col, "A", 1)
	if err != nil {
		t.Fatalf("GraphNeighbors(A, 1): %v", err)
	}

	got1 := sortedGraphNeighbors(d1)
	if len(got1) != 2 || got1[0] != "B" || got1[1] != "C" {
		t.Errorf("depth-1 = %v, want [B C]", got1)
	}

	// Depth 2 adds D (via B and C — deduplicated). The A-cycle never
	// re-admits the start node.
	d2, err := gb.GraphNeighbors(ctx, col, "A", 2)
	if err != nil {
		t.Fatalf("GraphNeighbors(A, 2): %v", err)
	}

	got2 := sortedGraphNeighbors(d2)
	if len(got2) != 3 || got2[2] != "D" {
		t.Errorf("depth-2 = %v, want [B C D]", got2)
	}

	// Depth 0: empty.
	d0, err := gb.GraphNeighbors(ctx, col, "A", 0)
	if err != nil {
		t.Fatalf("GraphNeighbors(A, 0): %v", err)
	}

	if len(d0) != 0 {
		t.Errorf("depth-0 should be empty, got %v", d0)
	}
}

func TestMySQLGraph_IdempotentEdgeAdd(t *testing.T) {
	t.Parallel()

	eng := mustNewMySQLEngine(t)
	gb := eng.(graphBackend)

	ctx := context.Background()
	col := "test_graph_mysql_idempotent"

	if err := gb.GraphAddEdge(ctx, col, metaengine.Edge{From: "X", To: "Y"}); err != nil {
		t.Fatalf("GraphAddEdge 1: %v", err)
	}

	if err := gb.GraphAddEdge(ctx, col, metaengine.Edge{From: "X", To: "Y"}); err != nil {
		t.Fatalf("GraphAddEdge duplicate: %v", err)
	}

	neighbors, err := gb.GraphNeighbors(ctx, col, "X", 1)
	if err != nil {
		t.Fatalf("GraphNeighbors: %v", err)
	}

	if len(neighbors) != 1 {
		t.Errorf("expected 1 neighbor after idempotent add, got %d: %v", len(neighbors), neighbors)
	}
}
