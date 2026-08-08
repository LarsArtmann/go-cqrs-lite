package dgraphengine_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	dgraphengine "github.com/larsartmann/go-cqrs-lite/metaengine/dgraphengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestGraphRAG_ConcurrentStress hammers the GraphRAG pipeline from multiple
// goroutines simultaneously to verify:
//
//  1. CORRECTNESS under concurrency — every query returns a non-empty context window.
//  2. NO PANICS or data races (run with -race).
//  3. THROUGHPUT — measures queries/sec, p50, p99 latency across 200 entities
//     with ~600 edges, 16 concurrent goroutines.
//
// This is the test that proves a single Dgraph instance can serve as a
// production GraphRAG backend under real concurrent load.
func TestGraphRAG_ConcurrentStress(t *testing.T) {
	t.Parallel()

	eng, err := dgraphengine.New(dgraphAddr())
	if err != nil {
		t.Skipf("Dgraph not available: %v", err)
	}
	defer metaengine.DeferClose(eng)

	gb, ok := eng.(metaengine.GraphBackend)
	if !ok {
		t.Fatal("engine does not implement GraphBackend")
	}

	sb, ok := eng.(metaengine.SearchBackend)
	if !ok {
		t.Fatal("engine does not implement SearchBackend")
	}

	ctx := context.Background()
	const searchCol = "rag-stress-search"
	const graphCol = "rag-stress-graph"

	// --- BUILD CORPUS ---
	// 200 entities across 10 topic clusters, each with 3 graph edges.
	// Creates a dense, realistic knowledge graph.
	const numEntities = 200
	topics := []string{
		"authentication", "database", "payment", "notification",
		"cache", "queue", "gateway", "storage", "analytics", "monitoring",
	}

	t.Logf("Indexing %d entities with graph edges...", numEntities)

	for i := range numEntities {
		topic := topics[i%len(topics)]
		doc := metaengine.IndexedText{
			ID:      fmt.Sprintf("entity-%d", i),
			Content: fmt.Sprintf("%s service component %d handles requests data", topic, i),
		}
		if err := sb.SearchInsert(ctx, searchCol, doc); err != nil {
			t.Fatalf("SearchInsert %d: %v", i, err)
		}
	}

	// Each entity connects to next 3 neighbors — 600 bidirectional edges.
	for i := range numEntities {
		for j := 1; j <= 3; j++ {
			neighbor := (i + j) % numEntities
			if err := gb.GraphAddEdge(
				ctx,
				graphCol,
				metaengine.Edge{
					From: fmt.Sprintf("entity-%d", i),
					To:   fmt.Sprintf("entity-%d", neighbor),
				},
			); err != nil {
				t.Fatalf("GraphAddEdge %d→%d: %v", i, neighbor, err)
			}
		}
	}

	t.Logf("Corpus ready: %d entities, ~%d edges, 10 topic clusters", numEntities, numEntities*3)

	// --- CONCURRENT STRESS ---
	const numGoroutines = 16
	const queriesPerGoroutine = 20
	totalQueries := int64(numGoroutines * queriesPerGoroutine)

	latencies := make([]int64, totalQueries)
	var idx int64

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	start := time.Now()

	for g := range numGoroutines {
		go func(goroutineID int) {
			defer wg.Done()

			for q := range queriesPerGoroutine {
				query := topics[(goroutineID+q)%len(topics)]
				i := atomic.AddInt64(&idx, 1) - 1

				queryStart := time.Now()

				// GraphRAG pipeline: search → expand.
				results, err := sb.SearchQuery(ctx, searchCol, query, 5)
				if err != nil {
					t.Errorf("goroutine %d query %d: SearchQuery %q: %v",
						goroutineID, q, query, err)
					return
				}

				if len(results) == 0 {
					t.Errorf("goroutine %d query %d: SearchQuery %q returned 0 results",
						goroutineID, q, query)
					return
				}

				// Expand each search hit via depth-2 graph traversal.
				for _, r := range results {
					neighbors, err := gb.GraphNeighbors(ctx, graphCol, r.ID, 2)
					if err != nil {
						t.Errorf("goroutine %d: GraphNeighbors %s: %v",
							goroutineID, r.ID, err)
						return
					}
					// Verify non-empty expansion (each entity has 3+ neighbors).
					if len(neighbors) == 0 {
						t.Errorf("goroutine %d: GraphNeighbors %s returned 0 neighbors",
							goroutineID, r.ID)
						return
					}
				}

				latencies[i] = time.Since(queryStart).Nanoseconds()
			}
		}(g)
	}

	wg.Wait()
	elapsed := time.Since(start)

	// --- REPORT ---
	successful := atomic.LoadInt64(&idx)
	qps := float64(successful) / elapsed.Seconds()

	// Sort latencies for percentile calculation.
	sortLatencies(latencies[:successful])
	p50 := latencies[successful*50/100]
	p99 := latencies[successful*99/100]
	maxLat := latencies[successful-1]

	t.Logf("=== GraphRAG Concurrent Stress Results ===")
	t.Logf("Concurrent goroutines: %d", numGoroutines)
	t.Logf("Queries per goroutine: %d", queriesPerGoroutine)
	t.Logf("Total queries:         %d", successful)
	t.Logf("Elapsed:               %v", elapsed)
	t.Logf("Throughput:            %.1f queries/sec", qps)
	t.Logf("Latency p50:           %v", time.Duration(p50))
	t.Logf("Latency p99:           %v", time.Duration(p99))
	t.Logf("Latency max:           %v", time.Duration(maxLat))
	t.Logf("Corpus:                %d entities, ~%d edges", numEntities, numEntities*3)

	// Throughput assertion: at least 50 queries/sec (generous for CI/Dgraph warmup).
	if qps < 50 {
		t.Errorf("throughput %.1f qps < 50 qps threshold", qps)
	}

	// p99 assertion: under 15 seconds (generous for single-node Dgraph under load).
	if time.Duration(p99) > 15*time.Second {
		t.Errorf("p99 latency %v > 15s threshold", time.Duration(p99))
	}
}

// sortLatencies sorts in-place using insertion sort (simple, no allocs for
// the small N we deal with).
func sortLatencies(s []int64) {
	for i := 1; i < len(s); i++ {
		key := s[i]
		j := i - 1
		for j >= 0 && s[j] > key {
			s[j+1] = s[j]
			j--
		}

		s[j+1] = key
	}
}
