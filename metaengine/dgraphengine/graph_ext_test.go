package dgraphengine_test

import (
	"context"
	"slices"
	"sort"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// graphExtBackend is the extended graph contract: tombstone-driven edge
// removal plus undirected traversal. Dgraph stores edges symmetrically, so
// GraphNeighborsUndirected is an alias of GraphNeighbors — the tests pin
// that contract too.
type graphExtBackend interface {
	GraphRemoveEdge(ctx context.Context, collection string, edge metaengine.Edge) error
	GraphNeighborsUndirected(
		ctx context.Context,
		collection string,
		node any,
		depth int,
	) ([]any, error)
}

func sortedExtNeighbors(items []any) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

// Serial (no t.Parallel): these semantics tests write upserts that abort
// under the GraphRAG stress corpus build's write burst; the serial phase
// runs before parallel tests release, so they execute uncontended.
func TestDgraphGraph_RemoveEdgeDeletesBothDirections(t *testing.T) {
	gb := mustNewDgraphEngine(t).(graphBackend)
	gx := gb.(graphExtBackend)
	ctx := context.Background()
	col := "test_graph_dgraph_remove"

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
		t.Fatalf("GraphNeighbors(A) after removal: %v", err)
	}

	if got := sortedExtNeighbors(neighbors); len(got) != 1 || got[0] != "C" {
		t.Errorf("after removal from A = %v, want [C]", got)
	}

	// Dgraph storage is symmetric — removal must clear the reverse direction
	// too, otherwise B would still see A.
	reverse, err := gb.GraphNeighbors(ctx, col, "B", 1)
	if err != nil {
		t.Fatalf("GraphNeighbors(B) after removal: %v", err)
	}

	if got := sortedExtNeighbors(reverse); len(got) != 0 {
		t.Errorf("reverse direction after removal = %v, want none", got)
	}

	// Idempotent: removing a missing edge is a no-op.
	if err := gx.GraphRemoveEdge(ctx, col, metaengine.Edge{From: "A", To: "B"}); err != nil {
		t.Fatalf("GraphRemoveEdge (re-remove): %v", err)
	}
}

func TestDgraphGraph_UndirectedSeesIncomingEdges(t *testing.T) {
	gb := mustNewDgraphEngine(t).(graphBackend)
	gx := gb.(graphExtBackend)
	ctx := context.Background()
	col := "test_graph_dgraph_undirected"

	// C→A only. Because Dgraph stores both directions, undirected AND
	// directed traversal from A both see C — the alias contract.
	if err := gb.GraphAddEdge(ctx, col, metaengine.Edge{From: "C", To: "A"}); err != nil {
		t.Fatalf("GraphAddEdge: %v", err)
	}

	undirected, err := gx.GraphNeighborsUndirected(ctx, col, "A", 1)
	if err != nil {
		t.Fatalf("GraphNeighborsUndirected(A, 1): %v", err)
	}

	if got := sortedExtNeighbors(undirected); len(got) != 1 || got[0] != "C" {
		t.Errorf("undirected from A = %v, want [C]", got)
	}

	directed, err := gb.GraphNeighbors(ctx, col, "A", 1)
	if err != nil {
		t.Fatalf("GraphNeighbors(A, 1): %v", err)
	}

	if got := sortedExtNeighbors(directed); len(got) != 1 || got[0] != "C" {
		t.Errorf("directed from A = %v, want [C] (symmetric storage)", got)
	}
}

func TestDgraphGraph_OptionalCapabilityChecks(t *testing.T) {
	t.Parallel()

	eng := mustNewDgraphEngine(t)

	if !metaengine.HasGraphEdgeRemoval(eng) {
		t.Error("dgraphEngine should implement GraphRemoveEdge")
	}

	if !metaengine.HasUndirectedGraphSupport(eng) {
		t.Error("dgraphEngine should implement GraphNeighborsUndirected")
	}
}

// Serial (no t.Parallel): writes the depth chain then traverses it; runs in
// the serial phase with the other semantics tests to avoid write contention.
//
// Pins Dgraph @recurse depth semantics after the off-by-one fix: Dgraph's
// @recurse(depth: N) counts node LEVELS (root = level 1) and traverses only
// N-1 hops, so GraphNeighbors requests depth+1 to match every other engine's
// hop-counting depth. Without the fix, depth 2 returned [B] instead of
// [B C] — see the matrix GraphDepthBound fixture.
func TestDgraphGraph_RecurseDepthCountsHops(t *testing.T) {
	gb := mustNewDgraphEngine(t).(graphBackend)
	ctx := context.Background()
	col := "test_graph_dgraph_recurse_depth"

	for _, e := range []metaengine.Edge{
		{From: "A", To: "B"},
		{From: "B", To: "C"},
		{From: "C", To: "D"},
		{From: "D", To: "E"},
	} {
		if err := gb.GraphAddEdge(ctx, col, e); err != nil {
			t.Fatalf("GraphAddEdge %v: %v", e, err)
		}
	}

	for depth, want := range map[int][]string{
		1: {"B"},
		2: {"B", "C"},
		3: {"B", "C", "D", "E"},
	} {
		got, err := gb.GraphNeighbors(ctx, col, "A", depth)
		if err != nil {
			t.Fatalf("GraphNeighbors(A, %d): %v", depth, err)
		}

		if have := sortedExtNeighbors(got); !slices.Equal(have, want) {
			t.Errorf("depth %d = %v, want %v", depth, have, want)
		}
	}
}
