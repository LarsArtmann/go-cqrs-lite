package pgengine_test

import (
	"context"
	"slices"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestPgEvolveLayoutPlan pins the information_schema evolution contract on
// live PG: missing columns are added, drifted types are retyped, re-running
// on a matching schema applies nothing (idempotency), and the evolved schema
// still serves planned numeric predicates.
func TestPgEvolveLayoutPlan(t *testing.T) {
	eng := mustNewPgEngine(t)

	ctx := context.Background()

	evolver, ok := eng.(metaengine.LayoutPlanEvolver)
	if !ok {
		t.Fatal("pgengine does not implement LayoutPlanEvolver")
	}

	// v1: amount declared TEXT (the legacy mis-typed shape).
	v1 := metaengine.LayoutPlan{
		Collection: "evolve_items",
		Table:      "meta_planned_evolve_items",
		Columns: []metaengine.PlannedColumn{
			{Name: "status", Type: "TEXT"},
			{Name: "amount", Type: "TEXT"},
		},
		Indexes: []metaengine.PlannedIndex{
			{Name: "idx_meta_planned_evolve_items_status", Columns: []string{"status"}},
		},
	}

	applied, err := evolver.EvolveLayoutPlan(ctx, v1)
	if err != nil {
		t.Fatalf("evolve v1: %v", err)
	}

	if len(applied) != 0 {
		t.Errorf("evolve v1 on a fresh table (created by evolve itself): applied = %v, want empty", applied)
	}

	// v2: amount retyped to DOUBLE — the evolution this capability exists for.
	v2 := metaengine.LayoutPlan{
		Collection: "evolve_items",
		Table:      "meta_planned_evolve_items",
		Columns: []metaengine.PlannedColumn{
			{Name: "status", Type: "TEXT"},
			{Name: "amount", Type: "DOUBLE"},
		},
		Indexes: []metaengine.PlannedIndex{
			{Name: "idx_meta_planned_evolve_items_status", Columns: []string{"status"}},
			{Name: "idx_meta_planned_evolve_items_amount", Columns: []string{"amount"}},
		},
	}

	applied, err = evolver.EvolveLayoutPlan(ctx, v2)
	if err != nil {
		t.Fatalf("evolve v2: %v", err)
	}

	if !slices.Contains(applied, "retype:amount") {
		t.Errorf("applied = %v, want retype:amount", applied)
	}

	// Idempotency: evolving again applies nothing.
	applied, err = evolver.EvolveLayoutPlan(ctx, v2)
	if err != nil {
		t.Fatalf("evolve v2 (re-run): %v", err)
	}

	if len(applied) != 0 {
		t.Errorf("idempotent re-run applied = %v, want empty", applied)
	}

	// Adding a column to the registered plan.
	v3 := v2
	v3.Columns = append(v3.Columns, metaengine.PlannedColumn{Name: "qty", Type: "INTEGER"})
	v3.Indexes = append(v3.Indexes, metaengine.PlannedIndex{
		Name: "idx_meta_planned_evolve_items_qty", Columns: []string{"qty"},
	})

	applied, err = evolver.EvolveLayoutPlan(ctx, v3)
	if err != nil {
		t.Fatalf("evolve v3: %v", err)
	}

	if !slices.Contains(applied, "add:qty") {
		t.Errorf("applied = %v, want add:qty", applied)
	}

	// The evolved planned table serves typed writes and numeric predicates.
	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		t.Fatal("engine does not implement MapBackend")
	}

	type item struct {
		Status string  `json:"status"`
		Amount float64 `json:"amount"`
	}

	if err := mb.MapSet(ctx, "evolve_items", "a", item{Status: "open", Amount: 2.5}); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	pushdown, ok := eng.(metaengine.PushdownScan)
	if !ok {
		t.Fatal("engine does not implement PushdownScan")
	}

	res, err := pushdown.PushdownMapScan(
		ctx, "evolve_items",
		[]metaengine.FilterSpec{{Column: "amount", Op: metaengine.FilterGt, Value: 2.0}},
		nil, nil, 0,
	)
	if err != nil {
		t.Fatalf("planned scan over evolved schema: %v", err)
	}

	if len(res.Items) != 1 {
		t.Errorf("numeric predicate over evolved column: got %d rows, want 1", len(res.Items))
	}
}
