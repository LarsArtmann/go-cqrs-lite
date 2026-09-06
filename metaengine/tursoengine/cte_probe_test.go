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

// TestTurso_RecursiveCTEProbeFails pins the remote-protocol finding: the
// turso (libSQL) driver rejects recursive CTEs ("Recursive CTEs are not yet
// supported"), so sqliteengine's construction-time probe correctly flips the
// graph path to iterative BFS instead of failing at query time. Verdict of
// the CTE-probe-over-remote-protocol TODO (2026-08-30): it does NOT hold —
// and the degraded fallback the probe feeds is what makes graph queries work
// anyway (pinned by TestTurso_GraphNeighborsDegraded).
func TestTurso_RecursiveCTEProbeFails(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("turso", ":memory:")
	if err != nil {
		t.Skipf("turso driver not available: %v", err)
	}
	defer func() { _ = db.Close() }()

	var got int

	if err := db.QueryRow(cteProbeSQL).Scan(&got); err == nil {
		t.Fatal("recursive CTE unexpectedly succeeded over the turso driver — " +
			"sqliteengine now takes the native CTE path; update this pin and re-verify graph parity")
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
