package badgerengine_test

import (
	"context"
	"sort"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

type graphBackend interface {
	GraphAddEdge(ctx context.Context, collection string, edge metaengine.Edge) error
	GraphNeighbors(ctx context.Context, collection string, node any, depth int) ([]any, error)
}

type graphExtBackend interface {
	GraphRemoveEdge(ctx context.Context, collection string, edge metaengine.Edge) error
	GraphNeighborsUndirected(ctx context.Context, collection string, node any, depth int) ([]any, error)
}

func sortedNeighbors(items []any) []string {
	out := make([]string, len(items))
	for i, v := range items {
		out[i], _ = v.(string)
	}

	sort.Strings(out)

	return out
}

func addEdge(t *testing.T, eng metaengine.Engine, ctx context.Context, col, from, to string) {
	t.Helper()

	if err := eng.(graphBackend).GraphAddEdge(ctx, col, metaengine.Edge{From: from, To: to}); err != nil {
		t.Fatalf("GraphAddEdge %s→%s: %v", from, to, err)
	}
}

func TestBadgerGraph_NeighborsDirected(t *testing.T) {
	t.Parallel()

	eng := mustNewBadgerEngine(t)
	ctx := context.Background()

	addEdge(t, eng, ctx, "graph_badger", "A", "B")
	addEdge(t, eng, ctx, "graph_badger", "A", "C")
	addEdge(t, eng, ctx, "graph_badger", "B", "D")

	neighbors, err := eng.(graphBackend).GraphNeighbors(ctx, "graph_badger", "A", 1)
	if err != nil {
		t.Fatalf("GraphNeighbors(A, 1): %v", err)
	}

	if got := sortedNeighbors(neighbors); len(got) != 2 || got[0] != "B" || got[1] != "C" {
		t.Errorf("depth 1 = %v, want [B C]", got)
	}

	neighbors, err = eng.(graphBackend).GraphNeighbors(ctx, "graph_badger", "A", 2)
	if err != nil {
		t.Fatalf("GraphNeighbors(A, 2): %v", err)
	}

	if got := sortedNeighbors(neighbors); len(got) != 3 || got[2] != "D" {
		t.Errorf("depth 2 = %v, want [B C D]", got)
	}
}

func TestBadgerGraph_DirectionMatters(t *testing.T) {
	t.Parallel()

	eng := mustNewBadgerEngine(t)
	ctx := context.Background()

	addEdge(t, eng, ctx, "graph_badger_dir", "A", "B")

	neighbors, err := eng.(graphBackend).GraphNeighbors(ctx, "graph_badger_dir", "B", 1)
	if err != nil {
		t.Fatalf("GraphNeighbors(B, 1): %v", err)
	}

	if got := sortedNeighbors(neighbors); len(got) != 0 {
		t.Errorf("directed traversal from B = %v, want none (edge is A→B)", got)
	}
}

func TestBadgerGraph_NeighborsUndirected(t *testing.T) {
	t.Parallel()

	eng := mustNewBadgerEngine(t)
	ctx := context.Background()

	addEdge(t, eng, ctx, "graph_badger_und", "A", "B")
	addEdge(t, eng, ctx, "graph_badger_und", "C", "A") // incoming edge, wrong direction

	neighbors, err := eng.(graphExtBackend).GraphNeighborsUndirected(ctx, "graph_badger_und", "A", 1)
	if err != nil {
		t.Fatalf("GraphNeighborsUndirected(A, 1): %v", err)
	}

	if got := sortedNeighbors(neighbors); len(got) != 2 || got[0] != "B" || got[1] != "C" {
		t.Errorf("undirected depth 1 = %v, want [B C] (both outgoing and incoming)", got)
	}
}

func TestBadgerGraph_RemoveEdge(t *testing.T) {
	t.Parallel()

	eng := mustNewBadgerEngine(t)
	ctx := context.Background()

	addEdge(t, eng, ctx, "graph_badger_rm", "A", "B")
	addEdge(t, eng, ctx, "graph_badger_rm", "A", "C")

	if err := eng.(graphExtBackend).GraphRemoveEdge(ctx, "graph_badger_rm", metaengine.Edge{From: "A", To: "B"}); err != nil {
		t.Fatalf("GraphRemoveEdge: %v", err)
	}

	neighbors, err := eng.(graphBackend).GraphNeighbors(ctx, "graph_badger_rm", "A", 1)
	if err != nil {
		t.Fatalf("GraphNeighbors after remove: %v", err)
	}

	if got := sortedNeighbors(neighbors); len(got) != 1 || got[0] != "C" {
		t.Errorf("after removal = %v, want [C]", got)
	}

	// Idempotent: removing a missing edge is a no-op.
	if err := eng.(graphExtBackend).GraphRemoveEdge(ctx, "graph_badger_rm", metaengine.Edge{From: "A", To: "B"}); err != nil {
		t.Fatalf("GraphRemoveEdge (idempotent re-remove): %v", err)
	}
}

func TestBadgerGraph_RemoveEdgeCleansReverseIndex(t *testing.T) {
	t.Parallel()

	eng := mustNewBadgerEngine(t)
	ctx := context.Background()

	addEdge(t, eng, ctx, "graph_badger_rmdir", "A", "B")

	if err := eng.(graphExtBackend).GraphRemoveEdge(ctx, "graph_badger_rmdir", metaengine.Edge{From: "A", To: "B"}); err != nil {
		t.Fatalf("GraphRemoveEdge: %v", err)
	}

	neighbors, err := eng.(graphExtBackend).GraphNeighborsUndirected(ctx, "graph_badger_rmdir", "B", 1)
	if err != nil {
		t.Fatalf("GraphNeighborsUndirected after remove: %v", err)
	}

	if got := sortedNeighbors(neighbors); len(got) != 0 {
		t.Errorf("reverse index leaked: undirected neighbors of B = %v, want none", got)
	}
}

func TestBadgerGraph_CycleAndDiamond(t *testing.T) {
	t.Parallel()

	eng := mustNewBadgerEngine(t)
	ctx := context.Background()

	// Cycle A→B→A must terminate (visited set).
	addEdge(t, eng, ctx, "graph_badger_cycle", "A", "B")
	addEdge(t, eng, ctx, "graph_badger_cycle", "B", "A")

	neighbors, err := eng.(graphBackend).GraphNeighbors(ctx, "graph_badger_cycle", "A", 5)
	if err != nil {
		t.Fatalf("GraphNeighbors on cycle: %v", err)
	}

	if got := sortedNeighbors(neighbors); len(got) != 1 || got[0] != "B" {
		t.Errorf("cycle traversal = %v, want [B]", got)
	}

	// Diamond A→B, A→C, B→D, C→D: D must appear once.
	for _, e := range [][2]string{{"A", "B"}, {"A", "C"}, {"B", "D"}, {"C", "D"}} {
		addEdge(t, eng, ctx, "graph_badger_diamond", e[0], e[1])
	}

	neighbors, err = eng.(graphBackend).GraphNeighbors(ctx, "graph_badger_diamond", "A", 3)
	if err != nil {
		t.Fatalf("GraphNeighbors on diamond: %v", err)
	}

	if got := sortedNeighbors(neighbors); len(got) != 3 {
		t.Errorf("diamond traversal = %v, want 3 unique nodes [B C D]", got)
	}
}

func TestBadgerGraph_DepthZero(t *testing.T) {
	t.Parallel()

	eng := mustNewBadgerEngine(t)
	ctx := context.Background()

	addEdge(t, eng, ctx, "graph_badger_d0", "A", "B")

	neighbors, err := eng.(graphBackend).GraphNeighbors(ctx, "graph_badger_d0", "A", 0)
	if err != nil {
		t.Fatalf("GraphNeighbors(A, 0): %v", err)
	}

	if len(neighbors) != 0 {
		t.Errorf("depth 0 = %v, want none", neighbors)
	}
}
