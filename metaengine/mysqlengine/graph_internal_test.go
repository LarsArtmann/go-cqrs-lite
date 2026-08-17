package mysqlengine

import (
	"context"
	"os"
	"slices"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// newInternalEngine builds an engine for internal (same-package) tests,
// skipping when no MySQL/MariaDB server is configured.
func newInternalEngine(tb testing.TB) *mysqlEngine {
	tb.Helper()

	dsn := os.Getenv("MYSQL_TEST_DSN")
	if dsn == "" {
		tb.Skip("MYSQL_TEST_DSN not set — skipping MySQL integration test")
	}

	eng, err := New(dsn)
	if err != nil {
		tb.Skipf("MySQL not available: %v", err)
	}

	e, ok := eng.(*mysqlEngine)
	if !ok {
		tb.Fatalf("unexpected engine type %T", eng)
	}

	tb.Cleanup(func() { _ = eng.Close() })

	return e
}

// sortedNeighbors renders a GraphNeighbors result as a sorted []string for
// order-insensitive comparison (SQL result order is unspecified).
func sortedNeighbors(t *testing.T, items []any) []string {
	t.Helper()

	out := make([]string, 0, len(items))
	for _, v := range items {
		s, ok := v.(string)
		if !ok {
			t.Fatalf("neighbor %v is %T, want string", v, v)
		}

		out = append(out, s)
	}

	slices.Sort(out)

	return out
}

// TestGraphCTEProbeEnabledOnModernServers asserts the construction-time CTE
// probe succeeds on MySQL 8+ / MariaDB 10.2+ so the fast path is taken.
func TestGraphCTEProbeEnabledOnModernServers(t *testing.T) {
	t.Parallel()

	e := newInternalEngine(t)
	if !e.graphCTE {
		t.Errorf("graphCTE = false on server %q; expected WITH RECURSIVE support", e.dialect)
	}
}

// TestGraphNeighborsUndirected_IterativeMatchesCTE is the undirected twin of
// TestGraphNeighbors_IterativeMatchesCTE: forces the both-directions BFS
// fallback and verifies it matches the undirected CTE walk. Guards the
// shared graphWalk skeleton's undirected adjacency selection.
func TestGraphNeighborsUndirected_IterativeMatchesCTE(t *testing.T) {
	t.Parallel()

	e := newInternalEngine(t)

	ctx := context.Background()
	col := "graph_und_cte_parity_internal"

	edges := []metaengine.Edge{
		{From: "b", To: "a"}, // incoming edge for a
		{From: "a", To: "c"}, // outgoing edge for a
		{From: "c", To: "b"}, // cycle back into the frontier
	}

	for _, edge := range edges {
		if err := e.GraphAddEdge(ctx, col, edge); err != nil {
			t.Fatalf("GraphAddEdge(%v): %v", edge, err)
		}
	}

	cteNeighbors, err := e.GraphNeighborsUndirected(ctx, col, "a", 2)
	if err != nil {
		t.Fatalf("GraphNeighborsUndirected (CTE): %v", err)
	}

	e.graphCTE = false

	iterativeNeighbors, err := e.GraphNeighborsUndirected(ctx, col, "a", 2)
	if err != nil {
		t.Fatalf("GraphNeighborsUndirected (iterative): %v", err)
	}

	cte := sortedNeighbors(t, cteNeighbors)
	iter := sortedNeighbors(t, iterativeNeighbors)

	if !slices.Equal(cte, iter) {
		t.Errorf("iterative undirected neighborhood %v != CTE %v", iter, cte)
	}

	want := []string{"b", "c"}
	if !slices.Equal(cte, want) {
		t.Errorf("undirected neighborhood = %v, want %v", cte, want)
	}
}

// TestGraphNeighbors_IterativeMatchesCTE forces the pre-CTE fallback path
// (as a MySQL 5.7 / MariaDB <10.2 server would take) and verifies it returns
// the same neighborhood as the CTE path: cycle-safe, deduplicated, and the
// start node excluded.
func TestGraphNeighbors_IterativeMatchesCTE(t *testing.T) {
	t.Parallel()

	e := newInternalEngine(t)

	ctx := context.Background()
	col := "graph_cte_parity_internal"

	edges := []metaengine.Edge{
		{From: "a", To: "b"}, {From: "b", To: "c"}, {From: "c", To: "a"}, // cycle
		{From: "a", To: "c"},                       // diamond entry
		{From: "c", To: "d"}, {From: "a", To: "d"}, // diamond: d reached twice
		{From: "d", To: "e"}, // one level beyond
	}

	for _, edge := range edges {
		if err := e.GraphAddEdge(ctx, col, edge); err != nil {
			t.Fatalf("GraphAddEdge(%v): %v", edge, err)
		}
	}

	cteNeighbors, err := e.GraphNeighbors(ctx, col, "a", 3)
	if err != nil {
		t.Fatalf("GraphNeighbors (CTE): %v", err)
	}

	e.graphCTE = false

	iterativeNeighbors, err := e.GraphNeighbors(ctx, col, "a", 3)
	if err != nil {
		t.Fatalf("GraphNeighbors (iterative): %v", err)
	}

	cte := sortedNeighbors(t, cteNeighbors)
	iter := sortedNeighbors(t, iterativeNeighbors)

	if !slices.Equal(cte, iter) {
		t.Errorf("iterative neighborhood %v != CTE neighborhood %v", iter, cte)
	}

	want := []string{"b", "c", "d", "e"}
	if !slices.Equal(cte, want) {
		t.Errorf("neighborhood = %v, want %v", cte, want)
	}
}

// TestGraphNeighbors_Depth1ShortCircuitMatchesCTE verifies the depth-1
// direct-adjacency short-circuit returns exactly the recursive-CTE depth-1
// neighborhood: the start node and its self-loop stay excluded and no
// depth-2 nodes leak in. The short-circuit bypasses the CTE/iterative mode
// switch entirely, so both modes must agree.
func TestGraphNeighbors_Depth1ShortCircuitMatchesCTE(t *testing.T) {
	t.Parallel()

	e := newInternalEngine(t)

	ctx := context.Background()
	col := "graph_depth1_direct_internal"

	edges := []metaengine.Edge{
		{From: "a", To: "a"}, // self-loop: must stay excluded
		{From: "a", To: "b"}, // depth-1 neighbor
		{From: "b", To: "c"}, // depth-2: must NOT leak into a depth-1 walk
	}

	for _, edge := range edges {
		if err := e.GraphAddEdge(ctx, col, edge); err != nil {
			t.Fatalf("GraphAddEdge(%v): %v", edge, err)
		}
	}

	got, err := e.GraphNeighbors(ctx, col, "a", 1)
	if err != nil {
		t.Fatalf("GraphNeighbors depth 1: %v", err)
	}

	gotSorted := sortedNeighbors(t, got)
	want := []string{"b"}
	if !slices.Equal(gotSorted, want) {
		t.Errorf("depth-1 neighborhood = %v, want %v", gotSorted, want)
	}

	if !e.graphCTE {
		t.Skip("server lacks WITH RECURSIVE — CTE comparison unavailable")
	}

	cte, err := e.graphNeighborsCTE(ctx, col, "a", 1)
	if err != nil {
		t.Fatalf("graphNeighborsCTE depth 1: %v", err)
	}

	if !slices.Equal(gotSorted, sortedNeighbors(t, cte)) {
		t.Errorf("depth-1 short-circuit %v != CTE %v", gotSorted, sortedNeighbors(t, cte))
	}

	e.graphCTE = false

	if _, err := e.GraphNeighbors(ctx, col, "a", 1); err != nil {
		t.Errorf("GraphNeighbors depth 1 without CTE mode: %v", err)
	}
}

// TestGraphNeighborsUndirected_Depth1ShortCircuitMatchesCTE is the
// undirected twin of TestGraphNeighbors_Depth1ShortCircuitMatchesCTE.
func TestGraphNeighborsUndirected_Depth1ShortCircuitMatchesCTE(t *testing.T) {
	t.Parallel()

	e := newInternalEngine(t)

	ctx := context.Background()
	col := "graph_depth1_undirected_internal"

	edges := []metaengine.Edge{
		{From: "a", To: "a"}, // self-loop: must stay excluded
		{From: "b", To: "a"}, // incoming: a depth-1 undirected neighbor
		{From: "a", To: "c"}, // outgoing: a depth-1 undirected neighbor
		{From: "c", To: "d"}, // depth-2: must NOT leak into a depth-1 walk
	}

	for _, edge := range edges {
		if err := e.GraphAddEdge(ctx, col, edge); err != nil {
			t.Fatalf("GraphAddEdge(%v): %v", edge, err)
		}
	}

	got, err := e.GraphNeighborsUndirected(ctx, col, "a", 1)
	if err != nil {
		t.Fatalf("GraphNeighborsUndirected depth 1: %v", err)
	}

	gotSorted := sortedNeighbors(t, got)
	want := []string{"b", "c"}
	if !slices.Equal(gotSorted, want) {
		t.Fatalf("undirected depth-1 neighborhood = %v, want %v", gotSorted, want)
	}

	if !e.graphCTE {
		t.Skip("server lacks WITH RECURSIVE — CTE comparison unavailable")
	}

	cte, err := e.graphNeighborsUndirectedCTE(ctx, col, "a", 1)
	if err != nil {
		t.Fatalf("graphNeighborsUndirectedCTE depth 1: %v", err)
	}

	if !slices.Equal(gotSorted, sortedNeighbors(t, cte)) {
		t.Errorf("depth-1 short-circuit %v != CTE %v", gotSorted, sortedNeighbors(t, cte))
	}
}
