package dgraphengine_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

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

	eng := mustNewDgraphEngine(t)

	gb, ok := eng.(graphBackend)
	if !ok {
		t.Fatal("engine does not implement graph dispatch")
	}

	sb, ok := eng.(metaengine.SearchBackend)
	if !ok {
		t.Fatal("engine does not implement SearchBackend")
	}

	ctx := context.Background()
	topics := buildStressCorpus(t, ctx, gb, sb)
	latencies, elapsed := runStressQueries(t, ctx, gb, sb, topics)

	successful := int64(len(latencies))
	if successful == 0 {
		t.Fatal("every stress query failed — see the reported errors above")
	}

	qps := float64(successful) / elapsed.Seconds()

	sortLatencies(latencies)
	p50 := latencies[successful*50/100]
	p99 := latencies[successful*99/100]
	maxLat := latencies[successful-1]

	t.Logf("=== GraphRAG Concurrent Stress Results ===")
	t.Logf("Concurrent goroutines: %d", stressGoroutines)
	t.Logf("Queries per goroutine: %d", stressQueriesPerGoroutine)
	t.Logf("Total queries:         %d", successful)
	t.Logf("Elapsed:               %v", elapsed)
	t.Logf("Throughput:            %.1f queries/sec", qps)
	t.Logf("Latency p50:           %v", time.Duration(p50))
	t.Logf("Latency p99:           %v", time.Duration(p99))
	t.Logf("Latency max:           %v", time.Duration(maxLat))
	t.Logf("Corpus:                %d entities, ~%d edges", stressEntities, stressEntities*3)

	// Throughput assertion: at least 50 queries/sec (generous for CI/Dgraph warmup).
	if qps < 50 {
		t.Errorf("throughput %.1f qps < 50 qps threshold", qps)
	}

	// p99 assertion: under 15 seconds (generous for single-node Dgraph under load).
	if time.Duration(p99) > 15*time.Second {
		t.Errorf("p99 latency %v > 15s threshold", time.Duration(p99))
	}
}

// Stress-shape constants: 200 entities across 10 topic clusters with 3 edges
// each is a dense, realistic knowledge graph; 16×20 queries give stable
// percentiles without turning the test into a soak.
const (
	stressEntities            = 200
	stressGoroutines          = 16
	stressQueriesPerGoroutine = 20
)

// buildStressCorpus indexes the entity corpus and its edges, returning the
// topic list queries draw from.
func buildStressCorpus(
	t *testing.T,
	ctx context.Context,
	gb graphBackend,
	sb metaengine.SearchBackend,
) []string {
	t.Helper()

	const searchCol = "rag-stress-search"
	const graphCol = "rag-stress-graph"

	topics := []string{
		"authentication", "database", "payment", "notification",
		"cache", "queue", "gateway", "storage", "analytics", "monitoring",
	}

	t.Logf("Indexing %d entities with graph edges...", stressEntities)

	for i := range stressEntities {
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
	for i := range stressEntities {
		for j := 1; j <= 3; j++ {
			neighbor := (i + j) % stressEntities
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

	t.Logf("Corpus ready: %d entities, ~%d edges, 10 topic clusters",
		stressEntities, stressEntities*3)

	return topics
}

// runStressQueries drives the concurrent search→expand pipeline and returns
// the per-query latencies (already truncated to successful queries) plus the
// wall-clock span.
func runStressQueries(
	t *testing.T,
	ctx context.Context,
	gb graphBackend,
	sb metaengine.SearchBackend,
	topics []string,
) ([]int64, time.Duration) {
	t.Helper()

	const searchCol = "rag-stress-search"
	const graphCol = "rag-stress-graph"

	totalQueries := int64(stressGoroutines * stressQueriesPerGoroutine)
	latencies := make([]int64, 0, totalQueries)

	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(stressGoroutines)

	start := time.Now()

	for g := range stressGoroutines {
		go func(goroutineID int) {
			defer wg.Done()

			for q := range stressQueriesPerGoroutine {
				query := topics[(goroutineID+q)%len(topics)]

				latency, ok := stressPipelineQuery(t, ctx, gb, sb, searchCol, graphCol, goroutineID, q, query)
				if !ok {
					return
				}

				mu.Lock()
				latencies = append(latencies, int64(latency))
				mu.Unlock()
			}
		}(g)
	}

	wg.Wait()

	return latencies, time.Since(start)
}

// stressPipelineQuery runs ONE GraphRAG pipeline query (search → depth-2
// expansion of every hit) and reports its latency; ok=false means the query
// failed and the error was already reported.
func stressPipelineQuery(
	t *testing.T,
	ctx context.Context,
	gb graphBackend,
	sb metaengine.SearchBackend,
	searchCol, graphCol string,
	goroutineID, q int,
	query string,
) (time.Duration, bool) {
	t.Helper()

	queryStart := time.Now()

	results, err := sb.SearchQuery(ctx, searchCol, query, 5)
	if err != nil {
		t.Errorf("goroutine %d query %d: SearchQuery %q: %v",
			goroutineID, q, query, err)

		return 0, false
	}

	if len(results) == 0 {
		t.Errorf("goroutine %d query %d: SearchQuery %q returned 0 results",
			goroutineID, q, query)

		return 0, false
	}

	for _, r := range results {
		neighbors, err := gb.GraphNeighbors(ctx, graphCol, r.ID, 2)
		if err != nil {
			t.Errorf("goroutine %d: GraphNeighbors %s: %v",
				goroutineID, r.ID, err)

			return 0, false
		}

		// Verify non-empty expansion (each entity has 3+ neighbors).
		if len(neighbors) == 0 {
			t.Errorf("goroutine %d: GraphNeighbors %s returned 0 neighbors",
				goroutineID, r.ID)

			return 0, false
		}
	}

	return time.Since(queryStart), true
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
