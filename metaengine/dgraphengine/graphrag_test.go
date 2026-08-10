package dgraphengine_test

import (
	"context"
	"fmt"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestGraphRAG_SearchThenGraphTraverse validates the core GraphRAG pipeline:
//
//  1. INDEX: insert entities with text descriptions (SearchInsert) AND build a
//     knowledge graph connecting them (GraphAddEdge).
//  2. RETRIEVE: search for entities by text query (SearchQuery).
//  3. EXPAND: traverse the graph neighborhood of each hit (GraphNeighbors).
//  4. ASSEMBLE: deduplicate into a context window.
//
// This is the use case that makes Dgraph uniquely valuable — it's the only
// metaengine that implements BOTH GraphBackend AND SearchBackend at full
// parity (no degradation). A GraphRAG pipeline on Memory/SQLite/Pebble would
// need two separate engines and glue code to join them.
func TestGraphRAG_SearchThenGraphTraverse(t *testing.T) {
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
	const searchCol = "rag-docs"
	const graphCol = "rag-graph"

	// --- INDEX PHASE ---
	// Build a small knowledge graph about a software team.
	entities := []ragEntity{
		{ID: "alice", Desc: "alice senior golang engineer microservices"},
		{ID: "bob", Desc: "bob database architect postgres dgraph"},
		{ID: "carol", Desc: "carol devops kubernetes deployment"},
		{ID: "dave", Desc: "dave frontend react typescript"},
		{ID: "eve", Desc: "eve security engineer authentication oauth"},
		{ID: "frank", Desc: "frank project manager agile roadmap"},
		{ID: "grace", Desc: "grace data scientist machine learning python"},
		{ID: "henry", Desc: "henry qa automation testing golang"},
	}

	for _, e := range entities {
		if err := sb.SearchInsert(ctx, searchCol, metaengine.IndexedText{
			ID:      e.ID,
			Content: e.Desc,
		}); err != nil {
			t.Fatalf("SearchInsert %s: %v", e.ID, err)
		}
	}

	// Build relationships: who works with whom, who reports to whom.
	edges := []metaengine.Edge{
		{From: "alice", To: "bob"},   // alice works with bob (backend + db)
		{From: "alice", To: "henry"}, // alice works with henry (shared golang)
		{From: "bob", To: "carol"},   // bob works with carol (db + deployment)
		{From: "carol", To: "dave"},  // carol deploys dave's frontend
		{From: "eve", To: "alice"},   // eve secures alice's services
		{From: "frank", To: "alice"}, // frank manages alice
		{From: "frank", To: "bob"},   // frank manages bob
		{From: "grace", To: "bob"},   // grace uses bob's database
		{From: "henry", To: "dave"},  // henry tests dave's frontend
	}

	for _, edge := range edges {
		if err := gb.GraphAddEdge(ctx, graphCol, edge); err != nil {
			t.Fatalf("GraphAddEdge %v: %v", edge, err)
		}
	}

	// --- RETRIEVE PHASE ---
	// Search for "golang" — should match alice and henry.
	results, err := sb.SearchQuery(ctx, searchCol, "golang", 10)
	if err != nil {
		t.Fatalf("SearchQuery golang: %v", err)
	}

	matched := make(map[string]bool)
	for _, r := range results {
		matched[r.ID] = true
	}

	if !matched["alice"] {
		t.Error("SearchQuery 'golang': expected alice in results")
	}

	if !matched["henry"] {
		t.Error("SearchQuery 'golang': expected henry in results")
	}

	// --- EXPAND PHASE ---
	// For each search hit, traverse 2-hop graph neighborhood.
	// alice's 2-hop: bob, henry (direct) → carol (via bob), dave (via henry), eve (via alice back)
	// henry's 2-hop: alice, dave (direct) → bob, eve, frank (via alice)
	contextWindow := make(map[string]bool)
	for _, r := range results {
		neighbors, err := gb.GraphNeighbors(ctx, graphCol, r.ID, 2)
		if err != nil {
			t.Fatalf("GraphNeighbors %s depth 2: %v", r.ID, err)
		}

		contextWindow[r.ID] = true // include the search hit itself
		for _, n := range neighbors {
			contextWindow[fmt.Sprint(n)] = true
		}
	}

	// --- VERIFY ---
	// "golang" matched alice + henry.
	// alice's 2-hop neighborhood includes bob, henry, carol, dave, eve, frank.
	// henry's 2-hop neighborhood includes alice, dave, bob, eve, frank.
	// Combined context should include at least: alice, bob, henry, dave.
	// (The exact set depends on bidirectional dedup, but these are guaranteed.)
	for _, expected := range []string{"alice", "bob", "henry"} {
		if !contextWindow[expected] {
			t.Errorf("GraphRAG context: expected %s in expanded context window (got %d entities)",
				expected, len(contextWindow))
		}
	}

	// grace and frank are NOT in the golang graph neighborhood at 2 hops
	// (grace has no path from alice/henry within 2 hops; frank IS reachable
	// from alice at 2 hops via the management edge, so we don't assert him).
	if contextWindow["grace"] {
		t.Error("GraphRAG context: grace should NOT be in golang context (no graph path)")
	}

	t.Logf("GraphRAG pipeline: search 'golang' → %d hits → expanded to %d context entities",
		len(results), len(contextWindow))
}

// TestGraphRAG_DifferentQueries verifies the pipeline works across
// different search terms, each expanding to different graph neighborhoods.
func TestGraphRAG_DifferentQueries(t *testing.T) {
	t.Parallel()

	eng := mustNewDgraphEngine(t)

	gb := eng.(graphBackend)
	sb := eng.(metaengine.SearchBackend)

	ctx := context.Background()
	const searchCol = "rag-multi-docs"
	const graphCol = "rag-multi-graph"

	// Index entities with distinct vocabulary.
	entities := []ragEntity{
		{ID: "svc-auth", Desc: "authentication service oauth jwt tokens"},
		{ID: "svc-pay", Desc: "payment service stripe transactions billing"},
		{ID: "svc-notify", Desc: "notification service email sms push"},
		{ID: "db-primary", Desc: "primary database postgresql replication"},
		{ID: "db-cache", Desc: "cache database redis memory fast"},
	}

	for _, e := range entities {
		_ = sb.SearchInsert(ctx, searchCol, metaengine.IndexedText{ID: e.ID, Content: e.Desc})
	}

	// Connect services to their dependencies.
	edges := []metaengine.Edge{
		{From: "svc-auth", To: "db-primary"},
		{From: "svc-auth", To: "db-cache"},
		{From: "svc-pay", To: "db-primary"},
		{From: "svc-notify", To: "db-cache"},
	}
	for _, e := range edges {
		_ = gb.GraphAddEdge(ctx, graphCol, e)
	}

	tests := []struct {
		query         string
		mustContain   []string // entities that MUST be in context window
		mustNotExpand []string // entities that should NOT appear via graph expansion
	}{
		{
			query:       "database",
			mustContain: []string{"db-primary", "db-cache"},
		},
		{
			query:       "payment",
			mustContain: []string{"svc-pay"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			t.Parallel()
			results, err := sb.SearchQuery(ctx, searchCol, tt.query, 10)
			if err != nil {
				t.Fatalf("SearchQuery %q: %v", tt.query, err)
			}

			if len(results) == 0 {
				t.Fatalf("SearchQuery %q: no results", tt.query)
			}

			// Expand each hit to 1-hop neighbors.
			seen := make(map[string]bool)
			for _, r := range results {
				seen[r.ID] = true
				neighbors, err := gb.GraphNeighbors(ctx, graphCol, r.ID, 1)
				if err != nil {
					t.Fatalf("GraphNeighbors %s: %v", r.ID, err)
				}
				for _, n := range neighbors {
					seen[fmt.Sprint(n)] = true
				}
			}

			for _, expected := range tt.mustContain {
				if !seen[expected] {
					t.Errorf("query %q: expected %s in context window (got %v)",
						tt.query, expected, seen)
				}
			}
		})
	}
}

// ragEntity is a test helper for GraphRAG scenarios: an entity with a text
// description (for search indexing) and an ID (for graph edges).
type ragEntity struct {
	ID   string
	Desc string
}
