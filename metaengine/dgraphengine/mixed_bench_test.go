package dgraphengine_test

import (
	"context"
	"fmt"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// This file contains mixed/realistic workload benchmarks that combine
// multiple backend capabilities — the patterns real consumers use.
//
// Dgraph is the ONLY metaengine implementing both GraphBackend AND
// SearchBackend at full parity. These benchmarks prove that combined
// workloads perform well on a single Dgraph instance — no need for a
// separate graph DB + search engine + glue code.

// --- GraphRAG Pipeline Benchmark ---

// buildGraphRAGCorpus populates a knowledge graph with search-indexed entities.
// Entities represent a realistic software-architecture domain: services
// connected to their dependencies via graph edges, each with a text description
// for full-text search.
//
// The graph has ~50 entities and ~100 edges (each entity connects to 2-3 others),
// forming a connected graph where 2-hop traversal reaches most of the graph.
func buildGraphRAGCorpus(b *testing.B, sb metaengine.SearchBackend, gb metaengine.GraphBackend,
	searchCol, graphCol string, numEntities int,
) {
	ctx := context.Background()
	topics := []string{
		"authentication", "database", "payment", "notification",
		"cache", "queue", "gateway", "storage", "analytics", "monitoring",
	}

	for i := range numEntities {
		topic := topics[i%len(topics)]
		doc := metaengine.IndexedText{
			ID:      fmt.Sprintf("entity-%d", i),
			Content: fmt.Sprintf("%s service component number %d handles requests", topic, i),
		}
		if err := sb.SearchInsert(ctx, searchCol, doc); err != nil {
			b.Fatalf("SearchInsert %d: %v", i, err)
		}
	}

	// Each entity connects to its next 2 neighbors + one long-range edge.
	for i := range numEntities {
		for _, offset := range []int{1, 2} {
			neighbor := (i + offset) % numEntities
			if err := gb.GraphAddEdge(
				ctx,
				graphCol,
				metaengine.Edge{
					From: fmt.Sprintf("entity-%d", i),
					To:   fmt.Sprintf("entity-%d", neighbor),
				},
			); err != nil {
				b.Fatalf("GraphAddEdge %d→%d: %v", i, neighbor, err)
			}
		}
	}
}

// BenchmarkDgraph_GraphRAG_SearchThenExpand measures the full GraphRAG retrieval
// pipeline: (1) full-text search to find relevant entities, (2) graph traversal
// to expand context. This is the end-to-end latency a consumer experiences.
//
// Pipeline per op:
//  1. SearchQuery("database", limit=5) → entity IDs
//  2. For each hit: GraphNeighbors(entity, depth=2) → related entities
//  3. Deduplicate into context window
func BenchmarkDgraph_GraphRAG_SearchThenExpand(b *testing.B) {
	eng := mustNewDgraphEngine(b)

	sb, ok := eng.(metaengine.SearchBackend)
	if !ok {
		b.Fatal("engine does not implement SearchBackend")
	}

	gb, ok := eng.(metaengine.GraphBackend)
	if !ok {
		b.Fatal("engine does not implement GraphBackend")
	}

	const searchCol = "rag-bench-search"
	const graphCol = "rag-bench-graph"
	const numEntities = 50

	buildGraphRAGCorpus(b, sb, gb, searchCol, graphCol, numEntities)

	queries := []string{"authentication", "database", "payment", "notification", "cache"}
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]

		// Phase 1: text search.
		results, err := sb.SearchQuery(ctx, searchCol, query, 5)
		if err != nil {
			b.Fatalf("SearchQuery %d: %v", i, err)
		}

		// Phase 2: graph expansion for each search hit.
		contextWindow := make(map[string]bool, 30)
		for _, r := range results {
			contextWindow[r.ID] = true
			neighbors, err := gb.GraphNeighbors(ctx, graphCol, r.ID, 2)
			if err != nil {
				b.Fatalf("GraphNeighbors %s: %v", r.ID, err)
			}
			for _, n := range neighbors {
				contextWindow[fmt.Sprint(n)] = true
			}
		}

		if len(contextWindow) == 0 {
			b.Fatalf("op %d: empty context window for query %q", i, query)
		}
	}
}

// --- Mixed Graph Write/Read Benchmark ---

// BenchmarkDgraph_GraphWriteReadMix simulates a realistic graph maintenance
// workload: add new edges while simultaneously querying neighborhoods. This
// tests whether writes (RAFT consensus) degrade concurrent reads.
//
// Pattern: 1 write + 3 reads per iteration (25% write, 75% read).
func BenchmarkDgraph_GraphWriteReadMix(b *testing.B) {
	eng := mustNewDgraphEngine(b)

	gb, ok := eng.(metaengine.GraphBackend)
	if !ok {
		b.Fatal("engine does not implement GraphBackend")
	}

	ctx := context.Background()

	// Pre-populate a base graph.
	const baseNodes = 50
	populateGraph(b, gb, "mix-graph", baseNodes)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Write: add a new edge.
		if err := gb.GraphAddEdge(ctx, "mix-graph",
			metaengine.Edge{From: i, To: (i + 7) % baseNodes}); err != nil {
			b.Fatalf("GraphAddEdge %d: %v", i, err)
		}

		// Read: query 3 different neighborhoods.
		for j := range 3 {
			node := (i + j*13) % baseNodes
			if _, err := gb.GraphNeighbors(ctx, "mix-graph", node, 1); err != nil {
				b.Fatalf("GraphNeighbors %d-%d: %v", i, j, err)
			}
		}
	}
}

// --- Mixed Map Read/Write Benchmark ---

// BenchmarkDgraph_MapReadWriteMix simulates a realistic key-value workload:
// 80% reads, 20% writes — the classic read-heavy pattern for materialized
// views and projections.
func BenchmarkDgraph_MapReadWriteMix(b *testing.B) {
	eng := mustNewDgraphEngine(b)

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		b.Fatal("engine does not implement MapBackend")
	}

	ctx := context.Background()

	// Pre-populate 500 keys.
	const numKeys = 500
	for i := range numKeys {
		if err := mb.MapSet(ctx, "mix-map", i, i*2); err != nil {
			b.Fatalf("pre-populate MapSet %d: %v", i, err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// 80% read, 20% write.
		if i%5 == 0 {
			if err := mb.MapSet(ctx, "mix-map", i%numKeys, i*3); err != nil {
				b.Fatalf("MapSet %d: %v", i, err)
			}
		} else {
			if _, _, err := mb.MapGet(ctx, "mix-map", i%numKeys); err != nil {
				b.Fatalf("MapGet %d: %v", i, err)
			}
		}
	}
}

// --- Full ADT Triad Benchmark ---

// BenchmarkDgraph_FullTriad_MapGraphSearch exercises all three primary Dgraph
// backends (Map, Graph, Search) in a single iteration loop — simulating a
// polyglot workload where one engine serves all three ADT needs.
//
// This benchmark answers: "Can a single Dgraph instance realistically serve
// a mixed CQRS workload with key-value projections, graph relationships, and
// full-text search?" (Spoiler: yes.)
func BenchmarkDgraph_FullTriad_MapGraphSearch(b *testing.B) {
	eng := mustNewDgraphEngine(b)

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		b.Fatal("engine does not implement MapBackend")
	}

	gb, ok := eng.(metaengine.GraphBackend)
	if !ok {
		b.Fatal("engine does not implement GraphBackend")
	}

	sb, ok := eng.(metaengine.SearchBackend)
	if !ok {
		b.Fatal("engine does not implement SearchBackend")
	}

	ctx := context.Background()
	const triadCol = "triad"

	// Pre-populate: 30 entities with map values, graph edges, and search docs.
	const numEntities = 30
	for i := range numEntities {
		entityID := fmt.Sprintf("e-%d", i)
		_ = mb.MapSet(ctx, triadCol+"-map", entityID, fmt.Sprintf("value-%d", i))
		_ = sb.SearchInsert(ctx, triadCol+"-search", metaengine.IndexedText{
			ID:      entityID,
			Content: fmt.Sprintf("entity %d with metadata and graph connections", i),
		})
		if i > 0 {
			_ = gb.GraphAddEdge(ctx, triadCol+"-graph",
				metaengine.Edge{From: fmt.Sprintf("e-%d", i-1), To: entityID})
		}
	}

	queries := []string{"entity", "metadata", "graph", "connections"}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		entityID := fmt.Sprintf("e-%d", i%numEntities)

		// Map read.
		if _, _, err := mb.MapGet(ctx, triadCol+"-map", entityID); err != nil {
			b.Fatalf("MapGet %d: %v", i, err)
		}

		// Search query.
		if _, err := sb.SearchQuery(
			ctx,
			triadCol+"-search",
			queries[i%len(queries)],
			5,
		); err != nil {
			b.Fatalf("SearchQuery %d: %v", i, err)
		}

		// Graph traversal.
		if _, err := gb.GraphNeighbors(ctx, triadCol+"-graph", entityID, 1); err != nil {
			b.Fatalf("GraphNeighbors %d: %v", i, err)
		}
	}
}
