package dgraphengine_test

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Live: two not-yet-existing counter keys that sanitize to the same blank-node
// label must both be created. Skips without a Dgraph server.
func TestDgraph_CounterIncrement_CollidingNewKeys(t *testing.T) {
	t.Parallel()

	eng := mustNewDgraphEngine(t)
	cb := eng.(metaengine.CounterBackend)
	col := uniqueCollection(t, "countercollide")

	for want := int64(1); want <= 2; want++ {
		if err := cb.CounterIncrement(t.Context(), col, metaengine.Delta{
			"a-b": 1, "a_b": 1, "user:tenant|hits": 1,
		}); err != nil {
			t.Fatalf("CounterIncrement: %v", err)
		}

		got, err := cb.CounterGet(t.Context(), col)
		if err != nil {
			t.Fatalf("CounterGet: %v", err)
		}

		if len(got) != 3 {
			t.Fatalf("want 3 counters, got %v", got)
		}

		for _, key := range []string{"a-b", "a_b", "user:tenant|hits"} {
			if got[key] != want {
				t.Fatalf("counter %q: want %d, got %d", key, want, got[key])
			}
		}
	}
}

// Live characterization: Dgraph applies the collection @filter before
// first:, so other collections' documents sharing the query terms must not
// push a collection's matches out of the window. Pins the query shape against
// the concern raised in review (observed passing on the unmodified query with
// a live server). Skips without a Dgraph server.
func TestDgraph_SearchQuery_LimitStaysWithinCollection(t *testing.T) {
	t.Parallel()

	eng := mustNewDgraphEngine(t)
	sb := eng.(metaengine.SearchBackend)
	ctx := t.Context()
	mine := uniqueCollection(t, "searchmine")
	foreign := uniqueCollection(t, "searchforeign")

	// Foreign documents go in first so they sit in front of the term matches.
	for i := range 8 {
		if err := sb.SearchInsert(ctx, foreign, metaengine.IndexedText{
			ID: string(rune('A' + i)), Content: "zephyrstorm foreign document",
		}); err != nil {
			t.Fatalf("SearchInsert: %v", err)
		}
	}

	for i := range 3 {
		if err := sb.SearchInsert(ctx, mine, metaengine.IndexedText{
			ID: string(rune('a' + i)), Content: "zephyrstorm own document",
		}); err != nil {
			t.Fatalf("SearchInsert: %v", err)
		}
	}

	got, err := sb.SearchQuery(ctx, mine, "zephyrstorm", 2)
	if err != nil {
		t.Fatalf("SearchQuery: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("want the 2 first matches of the collection, got %d: %v", len(got), got)
	}

	for _, r := range got {
		if r.ID[0] < 'a' || r.ID[0] > 'c' {
			t.Fatalf("result %q belongs to another collection", r.ID)
		}
	}
}

// Live: after ResetEngine dropped the edge predicates, a graph write to a
// previously used collection must rebuild the predicate and be readable.
// Not parallel: ResetEngine clears the whole engine. Skips without a server.
func TestResetEngine_GraphEdgeAfterReset(t *testing.T) {
	eng := mustNewDgraphEngine(t)
	ctx := context.Background()

	col := uniqueCollection(t, "resetedge")
	adder := eng.(dgraphGraphAdder)
	reader := eng.(dgraphGraphReader)

	for round := 1; round <= 2; round++ {
		edge := metaengine.Edge{From: "a", To: "b"}
		if err := adder.GraphAddEdge(ctx, col, edge); err != nil {
			t.Fatalf("round %d: GraphAddEdge: %v", round, err)
		}

		neighbors, err := reader.GraphNeighbors(ctx, col, "a", 1)
		if err != nil {
			t.Fatalf("round %d: GraphNeighbors: %v", round, err)
		}

		if len(neighbors) != 1 {
			t.Fatalf("round %d: want 1 neighbor, got %v", round, neighbors)
		}

		if err := eng.(metaengine.EngineResetter).ResetEngine(ctx); err != nil {
			t.Fatalf("round %d: ResetEngine: %v", round, err)
		}
	}
}

// Live, NOT yet observed against a server: ResetEngine drops the edge
// predicates together with their @reverse index, so an undirected read in
// the reverse direction must only work again once the predicate was
// recreated. Not parallel (ResetEngine clears the engine). Skips without a
// Dgraph server.
func TestResetEngine_ReverseEdgeReadAfterReset(t *testing.T) {
	eng := mustNewDgraphEngine(t)
	ctx := context.Background()

	col := uniqueCollection(t, "resetedge")
	adder := eng.(dgraphGraphAdder)
	undirected := eng.(graphExtBackend)

	edge := metaengine.Edge{From: "a", To: "b"}
	if err := adder.GraphAddEdge(ctx, col, edge); err != nil {
		t.Fatalf("GraphAddEdge before reset: %v", err)
	}

	if err := eng.(metaengine.EngineResetter).ResetEngine(ctx); err != nil {
		t.Fatalf("ResetEngine: %v", err)
	}

	if err := adder.GraphAddEdge(ctx, col, edge); err != nil {
		t.Fatalf("GraphAddEdge after reset: %v", err)
	}

	neighbors, err := undirected.GraphNeighborsUndirected(ctx, col, "b", 1)
	if err != nil {
		t.Fatalf("GraphNeighborsUndirected(b) after reset: %v", err)
	}

	if got := sortedExtNeighbors(neighbors); len(got) != 1 || got[0] != "a" {
		t.Fatalf("reverse-direction read after reset = %v, want [a]", got)
	}
}
