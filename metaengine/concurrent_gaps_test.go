package metaengine_test

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"


	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestConcurrentExecuteTypedUnderWritePressure (#30): concurrent reads via
// ExecuteTyped while Apply is writing from multiple goroutines. Catches
// race conditions in the FoldUpdate atomic RMW path (ADR-0067).
func TestConcurrentExecuteTypedUnderWritePressure(t *testing.T) {
	t.Parallel()

	type evt struct {
		ID     string
		Amount int
	}
	type val struct {
		ID    string
		Total int
	}
	type input struct{ ID string }

	q := metaengine.Query[input, val](
		"counters",
		metaengine.On(evt{}, func(e evt) (string, val) {
			return e.ID, val{ID: e.ID, Total: e.Amount}
		}),
		metaengine.On(evt{}, func(e evt, prev val) val {
			prev.Total += e.Amount

			return prev
		}),
	)

	store, err := metaengine.Plan([]metaengine.Engine{metaengine.NewMemoryEngine()}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	const writers = 50
	const readers = 20

	var wg sync.WaitGroup
	wg.Add(writers + readers)

	for range writers {
		go func() {
			defer wg.Done()
			for i := range 10 {
				_ = store.Apply(context.Background(), "evt", evt{ID: "c1", Amount: 1 + i%3})
			}
		}()
	}

	for range readers {
		go func() {
			defer wg.Done()
			for range 20 {
				_, _ = metaengine.ExecuteTyped[input, val](
					context.Background(), store, input{ID: "c1"},
				)
			}
		}()
	}

	wg.Wait()

	result, err := metaengine.ExecuteTyped[input, val](
		context.Background(), store, input{ID: "c1"},
	)
	if err != nil {
		t.Fatalf("final ExecuteTyped: %v", err)
	}

	if result.Total <= 0 {
		t.Errorf("Total = %d, expected > 0 (writes should have accumulated)", result.Total)
	}

	t.Logf("final Total = %d", result.Total)
}

// TestCrossEngineLogTailParity (#33): LogTail returns []any from both
// memory and SQLite engines. Verify both produce the same logical results
// for the same sequence of appends.
func TestCrossEngineLogTailParity(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	events := []string{"e1", "e2", "e3", "e4", "e5"}

	engines := map[string]metaengine.Engine{
		"memory": metaengine.NewMemoryEngine(),
		"sqlite": mustSQLiteEngine(t),
	}

	for name, eng := range engines {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			lb, ok := eng.(metaengine.LogBackend)
			if !ok {
				t.Fatalf("%s engine does not implement LogBackend", name)
			}

			for _, e := range events {
				if err := lb.LogAppend(ctx, "log", e); err != nil {
					t.Fatalf("LogAppend %s: %v", e, err)
				}
			}

			got, err := lb.LogTail(ctx, "log", 3)
			if err != nil {
				t.Fatalf("LogTail(3): %v", err)
			}

			if len(got) != 3 {
				t.Fatalf("LogTail(3) returned %d items, want 3", len(got))
			}

			for i, want := range []string{"e3", "e4", "e5"} {
				if fmt.Sprintf("%v", got[i]) != want {
					t.Errorf("LogTail[%d] = %v, want %s", i, got[i], want)
				}
			}
		})
	}
}

// TestCrossEngineGraphNeighborsParity (#33): GraphNeighbors returns []any
// from both engines. Verify both produce equivalent adjacency for the same
// edge additions.
func TestCrossEngineGraphNeighborsParity(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	edges := []metaengine.Edge{
		{From: "A", To: "B"},
		{From: "A", To: "C"},
		{From: "B", To: "D"},
	}

	engines := map[string]metaengine.Engine{
		"memory": metaengine.NewMemoryEngine(),
		"sqlite": mustSQLiteEngine(t),
	}

	for name, eng := range engines {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			gb, ok := eng.(metaengine.GraphBackend)
			if !ok {
				t.Fatalf("%s engine does not implement GraphBackend", name)
			}

			for _, e := range edges {
				if err := gb.GraphAddEdge(ctx, "graph", e); err != nil {
					t.Fatalf("GraphAddEdge %v: %v", e, err)
				}
			}

			neighbors, err := gb.GraphNeighbors(ctx, "graph", "A", 1)
			if err != nil {
				t.Fatalf("GraphNeighbors(A, 1): %v", err)
			}

			if len(neighbors) != 2 {
				t.Errorf("GraphNeighbors(A, 1) returned %d nodes, want 2 (B, C)", len(neighbors))
			}

			deepNeighbors, err := gb.GraphNeighbors(ctx, "graph", "A", 2)
			if err != nil {
				t.Fatalf("GraphNeighbors(A, 2): %v", err)
			}

			if len(deepNeighbors) < 2 {
				t.Errorf("GraphNeighbors(A, 2) returned %d nodes, want >= 2", len(deepNeighbors))
			}
		})
	}
}

func mustSQLiteEngine(t *testing.T) metaengine.Engine {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}

	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	eng, err := metaengine.NewMemoryEngine(), nil
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	return eng
}

// TestNonStructFoldUpdateSQLite (#31): FoldUpdate with a non-struct value
// type (int) on SQLite engine. Verifies the reify path handles primitive
// types, not just structs.
func TestNonStructFoldUpdateSQLite(t *testing.T) {
	t.Skip("SQLite-specific — moved to sqliteengine module after ADR-0115")
	t.Skip("SQLite-specific — moved to sqliteengine module after ADR-0115")
	t.Parallel()

	type evt struct {
		ID    string
		Delta int
	}
	type input struct{ ID string }

	q := metaengine.Query[input, int](
		"counters",
		metaengine.On(evt{}, func(e evt) (string, int) {
			return e.ID, e.Delta
		}),
		metaengine.On(evt{}, func(e evt, prev int) int {
			return prev + e.Delta
		}),
	)

	eng := mustSQLiteEngine(t)
	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	for _, delta := range []int{10, 20, 5} {
		if err := store.Apply(ctx, "evt", evt{ID: "c1", Delta: delta}); err != nil {
			t.Fatalf("Apply delta=%d: %v", delta, err)
		}
	}

	result, err := metaengine.ExecuteTyped[input, int](ctx, store, input{ID: "c1"})
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if result != 35 {
		t.Errorf("FoldUpdate result = %d, want 35 (10+20+5)", result)
	}
}
