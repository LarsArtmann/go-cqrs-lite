package metaengine_test

import (
	"context"
	"database/sql"
	"testing"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	_ "modernc.org/sqlite" // register sqlite driver
)

// TestGraphFallback_E2E_SQLiteStore runs the full Store pipeline against a real
// SQLite engine. Since SQLite now implements graphBackend natively via recursive
// CTE (ADTGraph declared as ComplexityODegree, not degraded), this test
// exercises the native CTE path through Store.Apply → Store.Execute.
func TestGraphFallback_E2E_SQLiteStore(t *testing.T) {
	t.Parallel()

	type followEvent struct {
		From string `json:"from"`
		To   string `json:"to"`
	}

	type graphQueryInput struct {
		Node  string `json:"node"`
		Depth int    `json:"depth"`
	}

	eng := newSQLiteTestEngine(t)

	q := metaengine.Query[graphQueryInput, []string](
		"follow_graph_sqlite",
		metaengine.OnRecordTyped(
			"user.followed",
			followEvent{},
			func(_ record.Record, evt followEvent) metaengine.Edge {
				return metaengine.Edge{From: evt.From, To: evt.To}
			},
		),
	)

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	// The planner must route the graph query to the only engine (SQLite),
	// emitting a DEGRADED diagnostic rather than failing.
	foundDegraded := false

	if plan := store.Plan(); plan != nil {
		for _, diag := range plan.Diagnostics {
			if diag.Level == metaengine.DiagLevelDegraded {
				foundDegraded = true
			}
		}
	}

	if !foundDegraded {
		t.Log("note: no DEGRADED diagnostic emitted for SQLite graph query")
	}

	ctx := context.Background()

	edges := []followEvent{
		{From: "dana", To: "erin"},
		{From: "erin", To: "frank"},
		{From: "dana", To: "frank"},
	}

	for _, e := range edges {
		if err := store.Apply(ctx, "user.followed", e); err != nil {
			t.Fatalf("Apply %v: %v", e, err)
		}
	}

	neighbors, err := store.ExecuteCtx(ctx, graphQueryInput{Node: "dana", Depth: 1})
	if err != nil {
		t.Fatalf("Execute depth-1: %v", err)
	}

	assertSQLiteNeighbors(t, neighbors, []string{"erin", "frank"})
}

func newSQLiteTestEngine(t *testing.T) metaengine.Engine {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	return eng
}

func assertSQLiteNeighbors(t *testing.T, got any, expected []string) {
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
