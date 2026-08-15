package sqliteengine_test

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"testing"

	_ "modernc.org/sqlite"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// graphBackend is the unexported dispatch contract. We re-declare it here
// to verify structural conformance at compile time.
type graphBackend interface {
	GraphAddEdge(ctx context.Context, collection string, edge metaengine.Edge) error
	GraphNeighbors(ctx context.Context, collection string, node any, depth int) ([]any, error)
}

func newGraphTestEngine(t *testing.T) metaengine.Engine {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}
	t.Cleanup(func() { _ = eng.Close() })
	return eng
}

func TestGraph_SQLiteImplementsGraphBackend(t *testing.T) {
	eng := newGraphTestEngine(t)

	if _, ok := eng.(graphBackend); !ok {
		t.Fatal("sqliteEngine should implement graphBackend (GraphAddEdge + GraphNeighbors)")
	}
}

func TestGraph_ProfileDeclaresNativeGraph(t *testing.T) {
	eng := newGraphTestEngine(t)

	profile := eng.Profile()

	c, ok := profile.Supports[metaengine.ADTGraph]
	if !ok {
		t.Fatal("SQLite profile should declare ADTGraph in Supports")
	}

	if c != metaengine.ComplexityODegree {
		t.Errorf("ADTGraph complexity = %s, want %s (native recursive CTE)",
			c, metaengine.ComplexityODegree)
	}

	if profile.IsDegraded(metaengine.ADTGraph) {
		t.Error("ADTGraph should NOT be degraded on SQLite (native recursive CTE)")
	}
}

func TestGraph_AddEdgeAndNeighborsDepth1(t *testing.T) {
	eng := newGraphTestEngine(t)
	gb := eng.(graphBackend)

	ctx := context.Background()
	col := "test_graph_d1"

	edges := []metaengine.Edge{
		{From: "A", To: "B"},
		{From: "A", To: "C"},
		{From: "B", To: "D"},
	}

	for _, e := range edges {
		if err := gb.GraphAddEdge(ctx, col, e); err != nil {
			t.Fatalf("GraphAddEdge %v: %v", e, err)
		}
	}

	neighbors, err := gb.GraphNeighbors(ctx, col, "A", 1)
	if err != nil {
		t.Fatalf("GraphNeighbors(A, 1): %v", err)
	}

	if len(neighbors) != 2 {
		t.Fatalf("expected 2 neighbors (B, C), got %d: %v", len(neighbors), neighbors)
	}

	got := sortedStrings(neighbors)
	if got[0] != "B" || got[1] != "C" {
		t.Errorf("neighbors = %v, want [B C]", got)
	}
}

func TestGraph_NeighborsDepth2(t *testing.T) {
	eng := newGraphTestEngine(t)
	gb := eng.(graphBackend)

	ctx := context.Background()
	col := "test_graph_d2"

	edges := []metaengine.Edge{
		{From: "A", To: "B"},
		{From: "B", To: "C"},
		{From: "C", To: "D"},
	}

	for _, e := range edges {
		_ = gb.GraphAddEdge(ctx, col, e)
	}

	// Depth 2 from A reaches B (depth 1) and C (depth 2), but NOT D (depth 3).
	neighbors, err := gb.GraphNeighbors(ctx, col, "A", 2)
	if err != nil {
		t.Fatalf("GraphNeighbors(A, 2): %v", err)
	}

	got := sortedStrings(neighbors)
	if len(got) != 2 || got[0] != "B" || got[1] != "C" {
		t.Errorf("depth-2 neighbors = %v, want [B C]", got)
	}

	// Depth 3 reaches D.
	neighbors3, _ := gb.GraphNeighbors(ctx, col, "A", 3)
	got3 := sortedStrings(neighbors3)
	if len(got3) != 3 || got3[0] != "B" || got3[1] != "C" || got3[2] != "D" {
		t.Errorf("depth-3 neighbors = %v, want [B C D]", got3)
	}
}

func TestGraph_NeighborsDepth0(t *testing.T) {
	eng := newGraphTestEngine(t)
	gb := eng.(graphBackend)

	ctx := context.Background()
	col := "test_graph_d0"

	_ = gb.GraphAddEdge(ctx, col, metaengine.Edge{From: "A", To: "B"})

	neighbors, err := gb.GraphNeighbors(ctx, col, "A", 0)
	if err != nil {
		t.Fatalf("GraphNeighbors(A, 0): %v", err)
	}

	if len(neighbors) != 0 {
		t.Errorf("depth-0 should return empty, got %v", neighbors)
	}
}

func TestGraph_CycleHandling(t *testing.T) {
	eng := newGraphTestEngine(t)
	gb := eng.(graphBackend)

	ctx := context.Background()
	col := "test_graph_cycle"

	// A → B → A creates a cycle.
	_ = gb.GraphAddEdge(ctx, col, metaengine.Edge{From: "A", To: "B"})
	_ = gb.GraphAddEdge(ctx, col, metaengine.Edge{From: "B", To: "A"})

	neighbors, err := gb.GraphNeighbors(ctx, col, "A", 5)
	if err != nil {
		t.Fatalf("GraphNeighbors(A, 5) on cycle: %v", err)
	}

	// Despite the cycle, we should get {B} (A is the start, excluded; B is the
	// only distinct neighbor within any depth).
	got := sortedStrings(neighbors)
	if len(got) != 1 || got[0] != "B" {
		t.Errorf("cycle neighbors from A = %v, want [B]", got)
	}
}

func TestGraph_IdempotentEdgeAdd(t *testing.T) {
	eng := newGraphTestEngine(t)
	gb := eng.(graphBackend)

	ctx := context.Background()
	col := "test_graph_idempotent"

	// Adding the same edge twice should not error (INSERT OR IGNORE).
	if err := gb.GraphAddEdge(ctx, col, metaengine.Edge{From: "X", To: "Y"}); err != nil {
		t.Fatalf("GraphAddEdge 1: %v", err)
	}

	if err := gb.GraphAddEdge(ctx, col, metaengine.Edge{From: "X", To: "Y"}); err != nil {
		t.Fatalf("GraphAddEdge duplicate: %v", err)
	}

	neighbors, _ := gb.GraphNeighbors(ctx, col, "X", 1)
	if len(neighbors) != 1 {
		t.Errorf("expected 1 neighbor after idempotent add, got %d: %v",
			len(neighbors), neighbors)
	}
}

// TestGraph_DeepChainCTE verifies the single-query recursive-CTE traversal
// on a deep graph: a 100-node chain traversed to full depth returns every
// node, and a mid-depth traversal returns exactly the prefix. This is the
// shape where the CTE path outperforms the old per-node-per-level queries.
func TestGraph_DeepChainCTE(t *testing.T) {
	t.Parallel()

	eng := newGraphTestEngine(t)
	gb := eng.(graphBackend)

	ctx := context.Background()
	col := "test_graph_deep_chain"

	const chainLen = 100
	for i := 0; i < chainLen; i++ {
		e := metaengine.Edge{From: fmt.Sprintf("n%03d", i), To: fmt.Sprintf("n%03d", i+1)}
		if err := gb.GraphAddEdge(ctx, col, e); err != nil {
			t.Fatalf("GraphAddEdge %v: %v", e, err)
		}
	}

	full, err := gb.GraphNeighbors(ctx, col, "n000", chainLen)
	if err != nil {
		t.Fatalf("GraphNeighbors full chain: %v", err)
	}

	got := sortedStrings(full)
	if len(got) != chainLen {
		t.Fatalf("full traversal: expected %d neighbors, got %d", chainLen, len(got))
	}

	if got[0] != "n001" || got[chainLen-1] != "n100" {
		t.Errorf("full traversal endpoints = [%s, %s], want [n001, n100]", got[0], got[chainLen-1])
	}

	// Depth 5 from n000 reaches exactly n001..n005.
	partial, err := gb.GraphNeighbors(ctx, col, "n000", 5)
	if err != nil {
		t.Fatalf("GraphNeighbors depth 5: %v", err)
	}

	gotPartial := sortedStrings(partial)
	if len(gotPartial) != 5 || gotPartial[4] != "n005" {
		t.Errorf("depth-5 traversal = %v, want [n001..n005]", gotPartial)
	}
}

// TestGraph_CTEDiamondDedup verifies a node reachable via multiple paths of
// different lengths appears exactly once in the CTE result.
func TestGraph_CTEDiamondDedup(t *testing.T) {
	t.Parallel()

	eng := newGraphTestEngine(t)
	gb := eng.(graphBackend)

	ctx := context.Background()
	col := "test_graph_diamond"

	edges := []metaengine.Edge{
		{From: "A", To: "B"},
		{From: "A", To: "C"},
		{From: "B", To: "D"},
		{From: "C", To: "D"},
		{From: "A", To: "D"},
	}
	for _, e := range edges {
		if err := gb.GraphAddEdge(ctx, col, e); err != nil {
			t.Fatalf("GraphAddEdge %v: %v", e, err)
		}
	}

	neighbors, err := gb.GraphNeighbors(ctx, col, "A", 3)
	if err != nil {
		t.Fatalf("GraphNeighbors diamond: %v", err)
	}

	got := sortedStrings(neighbors)
	if len(got) != 3 || got[0] != "B" || got[1] != "C" || got[2] != "D" {
		t.Errorf("diamond neighbors = %v, want [B C D] (D exactly once)", got)
	}
}

func sortedStrings(items []any) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.(string))
	}
	sort.Strings(out)
	return out
}
