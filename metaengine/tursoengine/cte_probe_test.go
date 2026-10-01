package tursoengine_test

import (
	"context"
	"database/sql"
	"slices"
	"testing"

	_ "turso.tech/database/tursogo" // registers "turso" driver with database/sql

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// cteProbeSQL mirrors sqliteengine's construction-time probe
// (probeRecursiveCTE): any error here — unsupported syntax, restricted remote
// protocol — disables the single-query recursive-CTE traversal in favor of
// iterative BFS.
const cteProbeSQL = `WITH RECURSIVE cqrs_cte_probe(x) AS (
	SELECT 1 UNION ALL SELECT x+1 FROM cqrs_cte_probe WHERE x < 1
) SELECT x FROM cqrs_cte_probe`

// TestTurso_RecursiveCTEProbeSucceeds pins the remote-protocol state of the
// turso (libSQL) driver, re-verified 2026-10-01: the driver NOW executes
// recursive CTEs, so sqliteengine's construction-time probe
// (probeRecursiveCTE) enables the single-query recursive-CTE traversal over
// turso DSNs. History: the 2026-08-30 verdict was the opposite ("Recursive
// CTEs are not yet supported") and the probe flipped graph queries to
// iterative BFS — the probe mechanism is unchanged and still protects
// against servers without CTE support; only this pin flipped with the
// upstream driver. Graph parity across both paths is pinned by
// TestTurso_GraphNeighborsDegraded (iterative BFS) and the sqliteengine
// graph suite (native CTE).
func TestTurso_RecursiveCTEProbeSucceeds(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("turso", ":memory:")
	if err != nil {
		t.Skipf("turso driver not available: %v", err)
	}
	defer func() { _ = db.Close() }()

	var got int

	if err := db.QueryRow(cteProbeSQL).Scan(&got); err != nil {
		t.Fatalf("recursive CTE unexpectedly failed over the turso driver "+
			"(upstream regression? probe falls back to iterative BFS): %v", err)
	}

	if got != 1 {
		t.Fatalf("cte probe returned %d, want 1 (seed row; the recursion terminates immediately)", got)
	}
}

// TestTurso_GraphNeighborsDegraded proves the iterative-BFS fallback answers
// correctly over the turso driver: depth-limited neighborhood of A in the
// chain A→B→C→D→E at depth 3 is exactly {B, C, D} (3 hops).
func TestTurso_GraphNeighborsDegraded(t *testing.T) {
	eng := mustNewTursoEngine(t)

	ctx := context.Background()

	graph, ok := eng.(interface {
		GraphAddEdge(ctx context.Context, collection string, edge metaengine.Edge) error
		GraphNeighbors(ctx context.Context, collection string, node any, depth int) ([]any, error)
	})
	if !ok {
		t.Skipf("turso engine does not implement the graph dispatch contract")
	}

	const col = "cte_graph"

	for _, edge := range []metaengine.Edge{
		{From: "A", To: "B"},
		{From: "B", To: "C"},
		{From: "C", To: "D"},
		{From: "D", To: "E"},
	} {
		if err := graph.GraphAddEdge(ctx, col, edge); err != nil {
			t.Fatalf("GraphAddEdge %s->%s: %v", edge.From, edge.To, err)
		}
	}

	nodes, err := graph.GraphNeighbors(ctx, col, "A", 3)
	if err != nil {
		t.Fatalf("GraphNeighbors: %v", err)
	}

	got := make([]string, 0, len(nodes))
	for _, n := range nodes {
		got = append(got, n.(string))
	}

	slices.Sort(got)

	want := []string{"B", "C", "D"}

	if !slices.Equal(got, want) {
		t.Errorf("depth-3 neighborhood of A = %v, want %v", got, want)
	}
}
