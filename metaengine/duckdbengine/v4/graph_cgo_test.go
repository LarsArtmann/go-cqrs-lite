//go:build cgo

package duckdbengine_test

import (
	"context"
	"sort"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// graph_cgo_test.go pins the native WITH RECURSIVE graph implementation on
// meta_graph_edges (mirrors the pgengine semantics): depth-limited traversal,
// cycle safety, dedup, and the honest empty results.

func addEdges(t *testing.T, eng metaengine.Engine, col string, edges ...metaengine.Edge) {
	t.Helper()

	gb, ok := eng.(interface {
		GraphAddEdge(ctx context.Context, col string, edge metaengine.Edge) error
	})
	if !ok {
		t.Fatal("engine does not implement graph dispatch")
	}

	for _, e := range edges {
		if err := gb.GraphAddEdge(context.Background(), col, e); err != nil {
			t.Fatalf("GraphAddEdge(%v): %v", e, err)
		}
	}
}

func neighbors(t *testing.T, eng metaengine.Engine, col string, node any, depth int) []string {
	t.Helper()

	gb, ok := eng.(interface {
		GraphNeighbors(
			ctx context.Context, col string, node any, depth int,
		) ([]any, error)
	})
	if !ok {
		t.Fatal("engine does not implement graph dispatch")
	}

	got, err := gb.GraphNeighbors(context.Background(), col, node, depth)
	if err != nil {
		t.Fatalf("GraphNeighbors(%v, depth=%d): %v", node, depth, err)
	}

	out := make([]string, 0, len(got))
	for _, n := range got {
		s, ok := n.(string)
		if !ok {
			t.Fatalf("neighbor %v is %T, want string", n, n)
		}
		out = append(out, s)
	}
	sort.Strings(out)

	return out
}

func TestDuckDB_Graph_DepthTraversal(t *testing.T) {
	t.Parallel()

	eng := mustNewDuckEngine(t)
	col := "graph_depth"

	addEdges(t, eng, col,
		metaengine.Edge{From: "a", To: "b"},
		metaengine.Edge{From: "a", To: "c"},
		metaengine.Edge{From: "b", To: "d"},
		metaengine.Edge{From: "d", To: "e"},
	)

	if got := neighbors(t, eng, col, "a", 1); len(got) != 2 || got[0] != "b" || got[1] != "c" {
		t.Errorf("depth 1: got %v, want [b c]", got)
	}

	got := neighbors(t, eng, col, "a", 2)
	if len(got) != 3 || got[0] != "b" || got[1] != "c" || got[2] != "d" {
		t.Errorf("depth 2: got %v, want [b c d]", got)
	}

	got = neighbors(t, eng, col, "a", 3)
	if len(got) != 4 || got[3] != "e" {
		t.Errorf("depth 3: got %v, want [b c d e]", got)
	}
}

func TestDuckDB_Graph_CycleSafe(t *testing.T) {
	t.Parallel()

	eng := mustNewDuckEngine(t)
	col := "graph_cycle"

	addEdges(t, eng, col,
		metaengine.Edge{From: "a", To: "b"},
		metaengine.Edge{From: "b", To: "a"},
		metaengine.Edge{From: "b", To: "c"},
	)

	got := neighbors(t, eng, col, "a", 5)
	if len(got) != 2 || got[0] != "b" || got[1] != "c" {
		t.Errorf("cycle walk: got %v, want [b c]", got)
	}
}

func TestDuckDB_Graph_EmptyAndDepthZero(t *testing.T) {
	t.Parallel()

	eng := mustNewDuckEngine(t)

	if got := neighbors(t, eng, "graph_empty", "a", 2); len(got) != 0 {
		t.Errorf("empty collection: got %v, want []", got)
	}

	addEdges(t, eng, "graph_d0", metaengine.Edge{From: "a", To: "b"})
	if got := neighbors(t, eng, "graph_d0", "a", 0); len(got) != 0 {
		t.Errorf("depth 0: got %v, want []", got)
	}
}

func TestDuckDB_Graph_DuplicateEdgeIdempotent(t *testing.T) {
	t.Parallel()

	eng := mustNewDuckEngine(t)
	col := "graph_dup"

	addEdges(t, eng, col, metaengine.Edge{From: "a", To: "b"})
	addEdges(t, eng, col, metaengine.Edge{From: "a", To: "b"})

	if got := neighbors(t, eng, col, "a", 1); len(got) != 1 || got[0] != "b" {
		t.Errorf("duplicate edge: got %v, want [b]", got)
	}
}

func TestDuckDB_Graph_IntegerNodeKeys(t *testing.T) {
	t.Parallel()

	eng := mustNewDuckEngine(t)
	col := "graph_int"

	addEdges(t, eng, col, metaengine.Edge{From: 1, To: 2})

	got := neighbors(t, eng, col, 1, 1)
	if len(got) != 1 || got[0] != "2" {
		t.Errorf("integer keys: got %v, want [2]", got)
	}
}
