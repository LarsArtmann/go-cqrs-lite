package dgraphengine_test

import (
	"context"
	"fmt"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// BenchmarkDgraph_MapSet measures the per-op cost of MapSet on the Dgraph engine.
// Used to validate DG_NsPerOp calibration (gRPC round-trip + RAFT consensus).
func BenchmarkDgraph_MapSet(b *testing.B) {
	eng := mustNewDgraphEngine(b)

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		b.Fatal("dgraph engine does not implement MapBackend")
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := mb.MapSet(ctx, "bench", i, i*2); err != nil {
			b.Fatalf("MapSet %d: %v", i, err)
		}
	}
}

// BenchmarkDgraph_MapGet measures the per-op cost of MapGet on the Dgraph engine.
// Used to validate DG_NsPerRead calibration (index lookup + gRPC response).
func BenchmarkDgraph_MapGet(b *testing.B) {
	eng := mustNewDgraphEngine(b)

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		b.Fatal("dgraph engine does not implement MapBackend")
	}

	ctx := context.Background()

	for i := range 1000 {
		if err := mb.MapSet(ctx, "bench", i, i*2); err != nil {
			b.Fatalf("pre-populate MapSet %d: %v", i, err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, found, err := mb.MapGet(ctx, "bench", i%1000)
		if err != nil {
			b.Fatalf("MapGet %d: %v", i, err)
		}

		if !found {
			b.Fatalf("MapGet %d: key not found", i)
		}
	}
}

// BenchmarkDgraph_CounterIncrement measures the per-op cost of CounterIncrement.
func BenchmarkDgraph_CounterIncrement(b *testing.B) {
	eng := mustNewDgraphEngine(b)

	cb, ok := eng.(metaengine.CounterBackend)
	if !ok {
		b.Fatal("dgraph engine does not implement CounterBackend")
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := cb.CounterIncrement(
			ctx,
			"bench",
			metaengine.Delta{fmt.Sprintf("c%d", i%10): 1},
		); err != nil {
			b.Fatalf("CounterIncrement %d: %v", i, err)
		}
	}
}

// BenchmarkDgraph_CounterGet measures the per-op cost of CounterGet.
func BenchmarkDgraph_CounterGet(b *testing.B) {
	eng := mustNewDgraphEngine(b)

	cb, ok := eng.(metaengine.CounterBackend)
	if !ok {
		b.Fatal("dgraph engine does not implement CounterBackend")
	}

	ctx := context.Background()

	for i := range 1000 {
		if err := cb.CounterIncrement(
			ctx,
			"bench",
			metaengine.Delta{fmt.Sprintf("c%d", i): 1},
		); err != nil {
			b.Fatalf("pre-populate CounterIncrement %d: %v", i, err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		counts, err := cb.CounterGet(ctx, "bench")
		if err != nil {
			b.Fatalf("CounterGet %d: %v", i, err)
		}

		if len(counts) == 0 {
			b.Fatalf("CounterGet %d: expected non-empty counters", i)
		}
	}
}

// BenchmarkDgraph_SetAdd measures the per-op cost of SetAdd on the Dgraph engine.
// Dgraph uses @index(exact) for O(logN) membership checks.
func BenchmarkDgraph_SetAdd(b *testing.B) {
	eng := mustNewDgraphEngine(b)

	sb, ok := eng.(metaengine.SetBackend)
	if !ok {
		b.Fatal("dgraph engine does not implement SetBackend")
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := sb.SetAdd(ctx, "bench", fmt.Sprintf("item-%d", i)); err != nil {
			b.Fatalf("SetAdd %d: %v", i, err)
		}
	}
}

// BenchmarkDgraph_GraphAddEdge measures the per-op cost of GraphAddEdge.
// Each call is a 2-step upsert: (1) create/ensure both nodes exist,
// (2) add bidirectional edges. Write-dominated by RAFT consensus.
func BenchmarkDgraph_GraphAddEdge(b *testing.B) {
	eng := mustNewDgraphEngine(b)

	gb, ok := eng.(graphBackend)
	if !ok {
		b.Fatal("dgraph engine does not implement graph dispatch")
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := gb.GraphAddEdge(ctx, "bench-graph-add",
			metaengine.Edge{From: i, To: i + 1}); err != nil {
			b.Fatalf("GraphAddEdge %d: %v", i, err)
		}
	}
}

// populateGraph builds a 100-node graph where each node connects to its next
// 3 neighbors (mod N). This creates a dense ring with ~300 bidirectional edges.
// At depth 3, most of the graph is reachable (3+9+27 = 39+ nodes).
func populateGraph(b *testing.B, gb graphBackend, collection string, numNodes int) {
	ctx := context.Background()

	for i := range numNodes {
		for j := 1; j <= 3; j++ {
			neighbor := (i + j) % numNodes
			if err := gb.GraphAddEdge(ctx, collection,
				metaengine.Edge{From: i, To: neighbor}); err != nil {
				b.Fatalf("pre-populate GraphAddEdge %d→%d: %v", i, neighbor, err)
			}
		}
	}
}

// BenchmarkDgraph_GraphNeighbors_Depth1 measures depth-1 neighbor traversal
// (direct adjacency). Read-only via NewReadOnlyTxn — bypasses RAFT.
func BenchmarkDgraph_GraphNeighbors_Depth1(b *testing.B) {
	eng := mustNewDgraphEngine(b)

	gb, ok := eng.(graphBackend)
	if !ok {
		b.Fatal("dgraph engine does not implement graph dispatch")
	}

	const numNodes = 100
	populateGraph(b, gb, "bench-graph-d1", numNodes)

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		node := i % numNodes
		neighbors, err := gb.GraphNeighbors(ctx, "bench-graph-d1", node, 1)
		if err != nil {
			b.Fatalf("GraphNeighbors depth1 %d: %v", i, err)
		}

		if len(neighbors) == 0 {
			b.Fatalf("GraphNeighbors depth1 %d: expected neighbors for node %d", i, node)
		}
	}
}

// BenchmarkDgraph_GraphNeighbors_Depth3 measures depth-3 multi-hop traversal
// via Dgraph's native @recurse — the killer feature that SQL needs recursive
// CTEs for. Read-only via NewReadOnlyTxn.
func BenchmarkDgraph_GraphNeighbors_Depth3(b *testing.B) {
	eng := mustNewDgraphEngine(b)

	gb, ok := eng.(graphBackend)
	if !ok {
		b.Fatal("dgraph engine does not implement graph dispatch")
	}

	const numNodes = 100
	populateGraph(b, gb, "bench-graph-d3", numNodes)

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		node := i % numNodes
		if _, err := gb.GraphNeighbors(ctx, "bench-graph-d3", node, 3); err != nil {
			b.Fatalf("GraphNeighbors depth3 %d: %v", i, err)
		}
	}
}

// BenchmarkDgraph_SearchInsert measures the per-op cost of SearchInsert.
// Each insertion upserts a document into the @index(term) full-text index.
// Write-dominated by RAFT consensus.
func BenchmarkDgraph_SearchInsert(b *testing.B) {
	eng := mustNewDgraphEngine(b)

	sb, ok := eng.(metaengine.SearchBackend)
	if !ok {
		b.Fatal("dgraph engine does not implement SearchBackend")
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		doc := metaengine.IndexedText{
			ID: fmt.Sprintf("doc-%d", i),
			Content: fmt.Sprintf(
				"document %d about golang performance graph database query optimization",
				i,
			),
		}
		if err := sb.SearchInsert(ctx, "bench-search-ins", doc); err != nil {
			b.Fatalf("SearchInsert %d: %v", i, err)
		}
	}
}

// BenchmarkDgraph_SearchQuery measures anyofterms() query latency over a
// 500-document corpus with varied vocabulary. Read-only via NewReadOnlyTxn.
func BenchmarkDgraph_SearchQuery(b *testing.B) {
	eng := mustNewDgraphEngine(b)

	sb, ok := eng.(metaengine.SearchBackend)
	if !ok {
		b.Fatal("dgraph engine does not implement SearchBackend")
	}

	ctx := context.Background()
	const numDocs = 500
	words := []string{
		"golang", "database", "graph", "performance", "query",
		"optimization", "cqrs", "event", "sourcing", "projection",
	}

	for i := range numDocs {
		w1 := words[i%len(words)]
		w2 := words[(i+3)%len(words)]
		w3 := words[(i+7)%len(words)]
		doc := metaengine.IndexedText{
			ID:      fmt.Sprintf("doc-%d", i),
			Content: fmt.Sprintf("%s %s %s document number %d", w1, w2, w3, i),
		}
		if err := sb.SearchInsert(ctx, "bench-search-q", doc); err != nil {
			b.Fatalf("pre-populate SearchInsert %d: %v", i, err)
		}
	}

	queries := []string{"golang", "performance", "graph", "database", "event"}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		results, err := sb.SearchQuery(ctx, "bench-search-q", queries[i%len(queries)], 10)
		if err != nil {
			b.Fatalf("SearchQuery %d: %v", i, err)
		}

		if len(results) == 0 {
			b.Fatalf("SearchQuery %d: expected results for %q", i, queries[i%len(queries)])
		}
	}
}
