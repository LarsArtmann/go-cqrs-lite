package metaengine

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
	_ "modernc.org/sqlite" // register sqlite driver
)

type followEvent struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type graphQueryInput struct {
	Node  string `json:"node"`
	Depth int    `json:"depth"`
}

// TestGraphFallback_E2E_StoreApplyExecute exercises the full Store pipeline:
// Plan → Apply (edge fold → multimap fallback) → Execute (traversal → BFS
// fallback) on an engine that implements MultimapBackend but NOT graphBackend.
//
// This verifies the "graceful degradation" invariant: a consumer with only a
// KV/multimap engine gets functional graph queries, albeit slower.
func TestGraphFallback_E2E_StoreApplyExecute(t *testing.T) {
	t.Parallel()

	eng := newMultimapOnlyEngine()

	q := Query[graphQueryInput, []string](
		"follow_graph",
		OnRecordTyped(
			"user.followed",
			followEvent{},
			func(_ record.Record, evt followEvent) Edge {
				return Edge{From: evt.From, To: evt.To}
			},
		),
	)

	store, err := Plan([]Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	applyEdges(t, ctx, store, "user.followed", []followEvent{
		{From: "alice", To: "bob"},
		{From: "bob", To: "carol"},
		{From: "alice", To: "carol"},
	})

	neighbors, err := store.ExecuteCtx(ctx, graphQueryInput{Node: "alice", Depth: 1})
	if err != nil {
		t.Fatalf("Execute depth-1: %v", err)
	}

	assertNeighbors(t, neighbors, []string{"bob", "carol"})

	depth2, err := store.ExecuteCtx(ctx, graphQueryInput{Node: "alice", Depth: 2})
	if err != nil {
		t.Fatalf("Execute depth-2: %v", err)
	}

	// Carol is reached at depth-1 via direct edge, so depth-2 adds nobody new.
	assertNeighbors(t, depth2, []string{"bob", "carol"})
}

// TestGraphFallback_E2E_LinearChain verifies depth-limited traversal on a
// linear chain: A → B → C → D. Depth-2 from A reaches B and C but not D.
func TestGraphFallback_E2E_LinearChain(t *testing.T) {
	t.Parallel()

	eng := newMultimapOnlyEngine()

	q := Query[graphQueryInput, []string](
		"chain_graph",
		OnRecordTyped(
			"node.linked",
			followEvent{},
			func(_ record.Record, evt followEvent) Edge {
				return Edge{From: evt.From, To: evt.To}
			},
		),
	)

	store, err := Plan([]Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	applyEdges(t, ctx, store, "node.linked", []followEvent{
		{From: "A", To: "B"},
		{From: "B", To: "C"},
		{From: "C", To: "D"},
	})

	depth2, err := store.ExecuteCtx(ctx, graphQueryInput{Node: "A", Depth: 2})
	if err != nil {
		t.Fatalf("Execute depth-2: %v", err)
	}

	assertNeighbors(t, depth2, []string{"B", "C"})

	depth3, err := store.ExecuteCtx(ctx, graphQueryInput{Node: "A", Depth: 3})
	if err != nil {
		t.Fatalf("Execute depth-3: %v", err)
	}

	assertNeighbors(t, depth3, []string{"B", "C", "D"})
}

func applyEdges(
	t *testing.T,
	ctx context.Context,
	store *Store,
	eventType string,
	edges []followEvent,
) {
	t.Helper()

	for _, e := range edges {
		if err := store.Apply(ctx, eventType, e); err != nil {
			t.Fatalf("Apply %v: %v", e, err)
		}
	}
}

func assertNeighbors(t *testing.T, got any, expected []string) {
	t.Helper()

	items, ok := got.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", got)
	}

	seen := make(map[string]bool, len(items))

	for _, item := range items {
		seen[item.(string)] = true
	}

	for _, exp := range expected {
		if !seen[exp] {
			t.Errorf("expected neighbor %q not found in %v", exp, items)
		}
	}

	if len(items) != len(expected) {
		t.Errorf("expected %d neighbors, got %d: %v", len(expected), len(items), items)
	}
}
