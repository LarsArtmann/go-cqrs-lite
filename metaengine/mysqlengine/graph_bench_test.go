package mysqlengine

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// Graph-traversal crossover benchmarks: recursive CTE vs iterative BFS.
//
// The engine picks the CTE path at construction when the server supports
// WITH RECURSIVE; these benchmarks force both modes against the same
// synthetic graph to measure where each wins, feeding the planner's cost
// model (see docs/planning/METAENGINE-LIVE-LATENCY-MODEL.md).
//
// Run against a server:
//
//	MYSQL_TEST_DSN="..." go test -bench 'BenchmarkGraphNeighbors' -benchtime 20x -run '^$'
//
// Graph shape: node i has out-degree 2 — a chain edge (i → i+1) plus a
// scattered edge (i → (2i+1) mod N) — so depth-d walks reach O(2^d) nodes
// with realistic index-join fan-out, and deep levels stay populated.
const (
	graphBenchMinSize = 1_000
	graphBenchSizes   = 3 // 1k, 10k, 100k (x10 per step)
	graphBenchMaxDept = 6
)

// seedGraphBench bulk-inserts the synthetic edge set for a node count.
func seedGraphBench(tb testing.TB, e *mysqlEngine, col string, nodes int) {
	tb.Helper()

	var b strings.Builder
	written := 0

	flush := func() {
		if written == 0 {
			return
		}

		stmt := "INSERT IGNORE INTO meta_graph_edges (collection, from_node, to_node) VALUES " +
			strings.TrimSuffix(b.String(), ",")
		if _, err := e.conn().ExecContext(context.Background(), stmt); err != nil {
			tb.Fatalf("seed graph bench: %v", err)
		}

		b.Reset()
		written = 0
	}

	for i := range nodes {
		fmt.Fprintf(&b, "('%s','%d','%d'),", col, i, i+1)
		fmt.Fprintf(&b, "('%s','%d','%d'),", col, i, (2*i+1)%nodes)
		written += 2

		if written >= 2_000 {
			flush()
		}
	}

	flush()
}

func runGraphNeighborsBench(b *testing.B, useCTE bool) {
	e := newInternalEngine(b)

	ctx := context.Background()

	size := graphBenchMinSize

	for range graphBenchSizes {
		col := fmt.Sprintf("bench_graph_%d", size)

		seedGraphBench(b, e, col, size)

		for depth := 1; depth <= graphBenchMaxDept; depth++ {
			b.Run(fmt.Sprintf("size-%d/depth-%d", size, depth), func(b *testing.B) {
				e.graphCTE = useCTE

				b.ResetTimer()

				for range b.N {
					if _, err := e.GraphNeighbors(ctx, col, 0, depth); err != nil {
						b.Fatalf("GraphNeighbors: %v", err)
					}
				}
			})
		}

		size *= 10
	}
}

// BenchmarkGraphNeighbors_CTE measures the single-query recursive walk.
func BenchmarkGraphNeighbors_CTE(b *testing.B) { runGraphNeighborsBench(b, true) }

// BenchmarkGraphNeighbors_Iterative measures the per-hop BFS fallback.
func BenchmarkGraphNeighbors_Iterative(b *testing.B) { runGraphNeighborsBench(b, false) }
