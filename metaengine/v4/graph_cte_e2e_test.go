package metaengine_test

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// TestGraphCTE_E2E_NativeSQLiteStore exercises the full Store pipeline for
// native recursive-CTE graph traversal on SQLite: Plan → Apply (edge folds
// into meta_graph_edges) → Execute (recursive CTE neighbors).
//
// This verifies that SQLite's native graphBackend implementation is correctly
// dispatched (not the multimap BFS fallback), producing correct depth-limited
// BFS results through the Store abstraction.
func TestGraphCTE_E2E_NativeSQLiteStore(t *testing.T) {
	t.Parallel()

	type followEvt struct {
		From string `json:"from"`
		To   string `json:"to"`
	}

	type graphInput struct {
		Node  string `json:"node"`
		Depth int    `json:"depth"`
	}

	eng := newSQLiteTestEngine(t)

	q := metaengine.Query[graphInput, []string](
		"native_graph_cte",
		metaengine.OnRecordTyped(
			"user.followed",
			followEvt{},
			func(_ record.Record, evt followEvt) metaengine.Edge {
				return metaengine.Edge{From: evt.From, To: evt.To}
			},
		),
	)

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	// The profile should declare native (not degraded) graph support.
	profile := eng.Profile()
	if profile.IsDegraded(metaengine.ADTGraph) {
		t.Error("SQLite should not declare ADTGraph as degraded (native CTE)")
	}

	// No DEGRADED diagnostic should be emitted for native graph.
	for _, d := range store.Plan().Diagnostics {
		if d.Level == metaengine.DiagLevelDegraded && strings.Contains(d.Message, "graph") {
			t.Errorf("unexpected DEGRADED for native graph: %s", d.Message)
		}
	}

	ctx := context.Background()

	edges := []followEvt{
		{From: "alice", To: "bob"},
		{From: "bob", To: "carol"},
		{From: "carol", To: "dave"},
		{From: "alice", To: "carol"},
	}

	for _, e := range edges {
		if err := store.Apply(ctx, "user.followed", e); err != nil {
			t.Fatalf("Apply %v: %v", e, err)
		}
	}

	// Depth 1: alice → {bob, carol}
	d1, err := store.ExecuteCtx(ctx, graphInput{Node: "alice", Depth: 1})
	if err != nil {
		t.Fatalf("Execute depth-1: %v", err)
	}
	assertGraphNeighbors(t, d1, []string{"bob", "carol"})

	// Depth 2: alice → {bob, carol, dave} (dave via carol at depth 2)
	d2, err := store.ExecuteCtx(ctx, graphInput{Node: "alice", Depth: 2})
	if err != nil {
		t.Fatalf("Execute depth-2: %v", err)
	}
	assertGraphNeighbors(t, d2, []string{"bob", "carol", "dave"})

	// Depth 3: same as depth 2 (no new nodes beyond dave)
	d3, err := store.ExecuteCtx(ctx, graphInput{Node: "alice", Depth: 3})
	if err != nil {
		t.Fatalf("Execute depth-3: %v", err)
	}
	assertGraphNeighbors(t, d3, []string{"bob", "carol", "dave"})
}

// TestGraphCTE_E2E_DoctorSection verifies the Doctor() report does NOT list
// graph queries as degraded when SQLite uses native CTE.
func TestGraphCTE_E2E_DoctorSection(t *testing.T) {
	t.Parallel()

	type followEvt struct {
		From string `json:"from"`
		To   string `json:"to"`
	}

	type graphInput struct {
		Node  string `json:"node"`
		Depth int    `json:"depth"`
	}

	eng := newSQLiteTestEngine(t)

	q := metaengine.Query[graphInput, []string](
		"doctor_graph_test",
		metaengine.OnRecordTyped(
			"user.followed",
			followEvt{},
			func(_ record.Record, evt followEvt) metaengine.Edge {
				return metaengine.Edge{From: evt.From, To: evt.To}
			},
		),
	)

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	_ = store.Apply(ctx, "user.followed", followEvt{From: "X", To: "Y"})

	report := store.Doctor(ctx)

	// The degraded section should say "none" (no degraded ADTs).
	degradedIdx := strings.Index(report, "--- Degraded ADTs ---")
	if degradedIdx < 0 {
		t.Fatal("Doctor() should include '--- Degraded ADTs ---' section")
	}

	section := report[degradedIdx:]
	if !strings.Contains(section, "none") {
		t.Errorf("Native graph should not appear in degraded section:\n%s", section)
	}
}

func assertGraphNeighbors(t *testing.T, got any, expected []string) {
	t.Helper()
	items, ok := got.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", got)
	}

	actual := make([]string, 0, len(items))
	for _, item := range items {
		s, ok := item.(string)
		if !ok {
			t.Fatalf("expected string neighbor, got %T: %v", item, item)
		}
		actual = append(actual, s)
	}

	sort.Strings(actual)
	sort.Strings(expected)

	if len(actual) != len(expected) {
		t.Errorf("expected %d neighbors %v, got %d: %v",
			len(expected), expected, len(actual), actual)
		return
	}

	for i := range actual {
		if actual[i] != expected[i] {
			t.Errorf("neighbor[%d] = %q, want %q (got: %v, want: %v)",
				i, actual[i], expected[i], actual, expected)
			return
		}
	}
}
