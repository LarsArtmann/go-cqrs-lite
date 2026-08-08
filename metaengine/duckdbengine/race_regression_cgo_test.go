//go:build cgo

package duckdbengine_test

import (
	"context"
	"sync"
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestDuckDB_RaceRegression_LayoutPlanConcurrentAccess is a targeted regression
// test for the data race fixed by changing layoutMu from sync.Mutex to
// sync.RWMutex and extracting the lookupPlan helper.
//
// Before the fix, ApplyLayoutPlan (write path) and MapSet/MapGet/MapDelete/
// ExplainAggregateQuery (read paths) accessed the plans map without
// synchronization, causing a data race detected by -race.
//
// This test spawns concurrent goroutines that apply layout plans and read via
// ExplainAggregateQuery and MapSet simultaneously, proving the RWMutex
// eliminates the race. Run with: go test -race -run TestDuckDB_RaceRegression.
func TestDuckDB_RaceRegression_LayoutPlanConcurrentAccess(t *testing.T) {
	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}
	defer eng.Close()

	ctx := context.Background()
	lp, ok := eng.(metaengine.LayoutPlanApplier)
	if !ok {
		t.Fatal("engine does not implement LayoutPlanApplier")
	}

	plan := metaengine.LayoutPlan{
		Collection: "race_test",
		Table:      "meta_planned_race_test",
		Columns: []metaengine.PlannedColumn{
			{Name: "status", Type: "VARCHAR"},
			{Name: "count", Type: "INTEGER"},
		},
	}

	if err := lp.ApplyLayoutPlan(plan); err != nil {
		t.Fatalf("initial ApplyLayoutPlan: %v", err)
	}

	const goroutines = 10
	const iterations = 50

	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	// Writer: repeatedly apply the same (column-compatible) plan.
	for range goroutines {
		go func() {
			defer wg.Done()

			for range iterations {
				_ = lp.ApplyLayoutPlan(plan)
			}
		}()
	}

	// Reader 1: ExplainAggregateQuery reads plans via lookupPlan.
	ea, ok := eng.(metaengine.ExplainableAggregate)
	if !ok {
		t.Fatal("engine does not implement ExplainableAggregate")
	}

	for range goroutines {
		go func() {
			defer wg.Done()

			for range iterations {
				_, _ = ea.ExplainAggregateQuery(ctx, "race_test", metaengine.ExplainAggregateOptions{
					Fn:     metaengine.AggregateCount,
					Column: "status",
				})
			}
		}()
	}

	// Reader 2: MapSet reads plans via lookupPlan.
	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		t.Fatal("engine does not implement MapBackend")
	}

	for range goroutines {
		go func() {
			defer wg.Done()

			for i := range iterations {
				_ = mb.MapSet(ctx, "race_test", "key", map[string]any{"status": "ok", "count": i})
			}
		}()
	}

	wg.Wait()
}
