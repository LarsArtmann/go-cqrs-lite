package pgengine_test

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestPgPlannedTablesReporting pins the PlannedTablesReporter capability on
// live PG: a registered planned collection is listed with its physical table,
// extracted columns, and a live row count; unregistered collections are not.
func TestPgPlannedTablesReporting(t *testing.T) {
	eng := mustNewPgEngine(t)

	ctx := context.Background()

	applier, ok := eng.(metaengine.LayoutPlanApplier)
	if !ok {
		t.Fatal("engine does not implement LayoutPlanApplier")
	}

	plan := metaengine.BuildLayoutPlan("report_items", []string{"status"}, nil)
	if err := applier.ApplyLayoutPlan(plan); err != nil {
		t.Fatalf("ApplyLayoutPlan: %v", err)
	}

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		t.Fatal("engine does not implement MapBackend")
	}

	if err := mb.MapSet(ctx, "report_items", "k1", map[string]any{"status": "open"}); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	reporter, ok := eng.(metaengine.PlannedTablesReporter)
	if !ok {
		t.Fatal("engine does not implement PlannedTablesReporter")
	}

	infos, err := reporter.PlannedTables(ctx)
	if err != nil {
		t.Fatalf("PlannedTables: %v", err)
	}

	if len(infos) != 1 {
		t.Fatalf("PlannedTables returned %d entries, want 1", len(infos))
	}

	info := infos[0]
	if info.Collection != "report_items" || info.Table != "meta_planned_report_items" {
		t.Errorf("info = %+v, want collection report_items on meta_planned_report_items", info)
	}

	if info.Rows != 1 {
		t.Errorf("Rows = %d, want 1", info.Rows)
	}

	if len(info.Columns) != 1 || info.Columns[0] != "status" {
		t.Errorf("Columns = %v, want [status]", info.Columns)
	}
}
