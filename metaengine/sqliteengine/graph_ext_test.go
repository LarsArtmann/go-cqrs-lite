package sqliteengine_test

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

type graphExtBackend interface {
	GraphRemoveEdge(ctx context.Context, collection string, edge metaengine.Edge) error
	GraphNeighborsUndirected(ctx context.Context, collection string, node any, depth int) ([]any, error)
}

func TestSQLiteGraph_RemoveEdge(t *testing.T) {
	t.Parallel()

	gb := newGraphTestEngine(t).(graphBackend)
	gx := gb.(graphExtBackend)
	ctx := context.Background()
	col := "graph_sqlite_remove"

	for _, e := range []metaengine.Edge{
		{From: "A", To: "B"},
		{From: "A", To: "C"},
	} {
		if err := gb.GraphAddEdge(ctx, col, e); err != nil {
			t.Fatalf("GraphAddEdge %v: %v", e, err)
		}
	}

	if err := gx.GraphRemoveEdge(ctx, col, metaengine.Edge{From: "A", To: "B"}); err != nil {
		t.Fatalf("GraphRemoveEdge: %v", err)
	}

	neighbors, err := gb.GraphNeighbors(ctx, col, "A", 1)
	if err != nil {
		t.Fatalf("GraphNeighbors after removal: %v", err)
	}

	if len(neighbors) != 1 || neighbors[0] != "C" {
		t.Errorf("after removal = %v, want [C]", neighbors)
	}

	// Idempotent: removing a missing edge is a no-op.
	if err := gx.GraphRemoveEdge(ctx, col, metaengine.Edge{From: "A", To: "B"}); err != nil {
		t.Fatalf("GraphRemoveEdge (re-remove): %v", err)
	}
}

func TestSQLiteGraph_NeighborsUndirected(t *testing.T) {
	t.Parallel()

	// Assertions hold for both the CTE and the iterative fallback — the
	// engine picks the mode at construction from a driver capability probe.
	gb := newGraphTestEngine(t).(graphBackend)
	gx := gb.(graphExtBackend)
	ctx := context.Background()
	col := "graph_sqlite_undirected"

	// Mixed directions: A→B, C→A, B→D. Undirected depth 1 from A sees B and C.
	for _, e := range []metaengine.Edge{
		{From: "A", To: "B"},
		{From: "C", To: "A"},
		{From: "B", To: "D"},
	} {
		if err := gb.GraphAddEdge(ctx, col, e); err != nil {
			t.Fatalf("GraphAddEdge %v: %v", e, err)
		}
	}

	neighbors, err := gx.GraphNeighborsUndirected(ctx, col, "A", 1)
	if err != nil {
		t.Fatalf("GraphNeighborsUndirected(A, 1): %v", err)
	}

	if len(neighbors) != 2 {
		t.Fatalf("undirected depth 1 = %v, want 2 neighbors [B C]", neighbors)
	}

	// Undirected depth 2 from A reaches D via B.
	neighbors, err = gx.GraphNeighborsUndirected(ctx, col, "A", 2)
	if err != nil {
		t.Fatalf("GraphNeighborsUndirected(A, 2): %v", err)
	}

	if len(neighbors) != 3 {
		t.Fatalf("undirected depth 2 = %v, want 3 unique nodes", neighbors)
	}

	// Directed traversal from A must NOT see C (edge is C→A).
	directed, err := gb.GraphNeighbors(ctx, col, "A", 2)
	if err != nil {
		t.Fatalf("GraphNeighbors(A, 2): %v", err)
	}

	if len(directed) != 2 {
		t.Errorf("directed from A = %v, want exactly [B D]", directed)
	}
}

func TestSQLiteGraph_UndirectedAfterRemoval(t *testing.T) {
	t.Parallel()

	gb := newGraphTestEngine(t).(graphBackend)
	gx := gb.(graphExtBackend)
	ctx := context.Background()
	col := "graph_sqlite_und_rm"

	if err := gb.GraphAddEdge(ctx, col, metaengine.Edge{From: "A", To: "B"}); err != nil {
		t.Fatal(err)
	}

	if err := gx.GraphRemoveEdge(ctx, col, metaengine.Edge{From: "A", To: "B"}); err != nil {
		t.Fatal(err)
	}

	neighbors, err := gx.GraphNeighborsUndirected(ctx, col, "B", 1)
	if err != nil {
		t.Fatalf("GraphNeighborsUndirected after removal: %v", err)
	}

	if len(neighbors) != 0 {
		t.Errorf("undirected neighbors after removal = %v, want none", neighbors)
	}
}
