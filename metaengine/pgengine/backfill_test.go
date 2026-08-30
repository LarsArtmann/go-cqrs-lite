package pgengine_test

import (
	"context"
	"fmt"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestPgBackfillPlannedCollection pins the opt-in backfill helper on live PG:
// rows written to meta_map before the plan was registered become visible to
// planned scans, keyset paging covers multi-batch collections, extracted
// columns are recomputed (numeric predicates work), and re-running is
// idempotent.
func TestPgBackfillPlannedCollection(t *testing.T) {
	eng := mustNewPgEngine(t)

	ctx := context.Background()

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		t.Fatal("engine does not implement MapBackend")
	}

	// Seed meta_map while the collection is still planless.
	type item struct {
		Status string  `json:"status"`
		Amount float64 `json:"amount"`
	}

	for i := range 7 {
		status := "open"
		if i%2 == 1 {
			status = "done"
		}

		key := fmt.Sprintf("k%02d", i)

		if err := mb.MapSet(ctx, "backfill_items", key, item{Status: status, Amount: float64(i)}); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}

	applier, ok := eng.(metaengine.LayoutPlanApplier)
	if !ok {
		t.Fatal("engine does not implement LayoutPlanApplier")
	}

	plan := metaengine.BuildLayoutPlanFromType[item](
		"backfill_items", []string{"status"}, []string{"amount"},
	)
	if err := applier.ApplyLayoutPlan(plan); err != nil {
		t.Fatalf("ApplyLayoutPlan: %v", err)
	}

	// Pre-registration rows are invisible until backfilled.
	scan, ok := eng.(metaengine.ScanBackend)
	if !ok {
		t.Fatal("engine does not implement ScanBackend")
	}

	res, err := scan.MapScan(ctx, "backfill_items", nil, nil, nil, 0)
	if err != nil {
		t.Fatalf("planned scan pre-backfill: %v", err)
	}

	if len(res.Items) != 0 {
		t.Fatalf(
			"planned scan pre-backfill saw %d rows, want 0 (no-backfill contract)",
			len(res.Items),
		)
	}

	// Backfill with paging (batch 3 over 7 rows).
	n, err := metaengine.BackfillPlannedCollection(ctx, eng, "backfill_items", 3)
	if err != nil {
		t.Fatalf("BackfillPlannedCollection: %v", err)
	}

	if n != 7 {
		t.Fatalf("backfilled %d rows, want 7", n)
	}

	// Idempotency: re-running re-upserts the same rows.
	n2, err := metaengine.BackfillPlannedCollection(ctx, eng, "backfill_items", 3)
	if err != nil {
		t.Fatalf("BackfillPlannedCollection re-run: %v", err)
	}

	if n2 != 7 {
		t.Fatalf("re-run backfilled %d rows, want 7 (idempotent)", n2)
	}

	// Planned scans now see everything, with recomputed extracted columns.
	res, err = scan.MapScan(ctx, "backfill_items", nil, nil, nil, 0)
	if err != nil {
		t.Fatalf("planned scan post-backfill: %v", err)
	}

	if len(res.Items) != 7 {
		t.Fatalf("planned scan post-backfill saw %d rows, want 7", len(res.Items))
	}

	pushdown, ok := eng.(metaengine.PushdownScan)
	if !ok {
		t.Fatal("engine does not implement PushdownScan")
	}

	res, err = pushdown.PushdownMapScan(
		ctx, "backfill_items",
		[]metaengine.FilterSpec{{Column: "amount", Op: metaengine.FilterGt, Value: 5.5}},
		nil, nil, 0,
	)
	if err != nil {
		t.Fatalf("numeric predicate over backfilled column: %v", err)
	}

	if len(res.Items) != 1 {
		t.Errorf("numeric predicate over backfilled column: got %d rows, want 1", len(res.Items))
	}
}
