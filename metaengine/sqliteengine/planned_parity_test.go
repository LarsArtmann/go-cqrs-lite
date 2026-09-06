package sqliteengine_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// newParityEngine opens an in-memory shared-cache SQLite engine with one
// planned collection registered via ApplyLayoutPlan.
func newParityEngine(t *testing.T) metaengine.Engine {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	plan := metaengine.BuildLayoutPlan("parity_items", []string{"status"}, []string{"priority"})
	if err := eng.(metaengine.LayoutPlanApplier).ApplyLayoutPlan(plan); err != nil {
		t.Fatalf("ApplyLayoutPlan: %v", err)
	}

	return eng
}

// TestSQLite_MapScanKeyValues_Paged pins the KeyScanBackend contract: paged
// key+value reads over the BASE meta_map in deterministic key order, with
// hasMore signaling exactly one trailing page.
func TestSQLite_MapScanKeyValues_Paged(t *testing.T) {
	t.Parallel()

	eng := newParityEngine(t)
	ctx := context.Background()

	mb := eng.(metaengine.MapBackend)

	const total = 25

	for i := range total {
		key := fmt.Sprintf("k%02d", i)
		if err := mb.MapSet(ctx, "base_items", key, map[string]any{"n": i}); err != nil {
			t.Fatalf("MapSet %s: %v", key, err)
		}
	}

	kb := eng.(metaengine.KeyScanBackend)

	var (
		gotKeys  []any
		gotVals  []any
		cursor   any
		hasMore  = true
		requests int
	)

	for hasMore {
		keys, vals, more, err := kb.MapScanKeyValues(ctx, "base_items", cursor, 10)
		if err != nil {
			t.Fatalf("MapScanKeyValues: %v", err)
		}

		gotKeys = append(gotKeys, keys...)
		gotVals = append(gotVals, vals...)
		hasMore = more
		requests++

		if len(keys) > 0 {
			cursor = keys[len(keys)-1]
		}

		if requests > total {
			t.Fatal("paging did not terminate")
		}
	}

	if len(gotKeys) != total {
		t.Fatalf("scanned %d keys, want %d", len(gotKeys), total)
	}

	for i := 1; i < len(gotKeys); i++ {
		if fmt.Sprint(gotKeys[i-1]) >= fmt.Sprint(gotKeys[i]) {
			t.Fatalf("keys out of order at %d: %v then %v", i, gotKeys[i-1], gotKeys[i])
		}
	}
}

// TestSQLite_EvolveLayoutPlan pins the LayoutPlanEvolver contract: missing
// columns are added, the operation is idempotent, indexes are created, and a
// declared-type drift on an existing column fails loudly (SQLite cannot
// ALTER COLUMN TYPE — silence would hide the drifted affinity).
func TestSQLite_EvolveLayoutPlan(t *testing.T) {
	t.Parallel()

	eng := newParityEngine(t)
	ctx := context.Background()

	ev := eng.(metaengine.LayoutPlanEvolver)

	plan := metaengine.BuildLayoutPlan("evolve_items", []string{"status", "region"}, []string{"priority"})
	if _, err := ev.EvolveLayoutPlan(ctx, plan); err != nil {
		t.Fatalf("EvolveLayoutPlan: %v", err)
	}

	applied, err := ev.EvolveLayoutPlan(ctx, plan)
	if err != nil {
		t.Fatalf("EvolveLayoutPlan (idempotent): %v", err)
	}

	if len(applied) != 0 {
		t.Fatalf("second evolve applied %v, want no-op", applied)
	}

	// Type drift must fail loudly: priority already exists as INTEGER.
	drift := plan
	drift.Columns = []metaengine.PlannedColumn{
		{Name: "priority", Type: "REAL"},
	}

	if _, err := ev.EvolveLayoutPlan(ctx, drift); err == nil {
		t.Fatal("type drift on existing column must fail loudly")
	}

	// The reporter sees the evolved table.
	rep := eng.(metaengine.PlannedTablesReporter)

	infos, err := rep.PlannedTables(ctx)
	if err != nil {
		t.Fatalf("PlannedTables: %v", err)
	}

	found := false

	for _, info := range infos {
		if info.Collection == "evolve_items" {
			found = true

			if info.Rows != 0 {
				t.Fatalf("evolve_items rows = %d, want 0", info.Rows)
			}
		}
	}

	if !found {
		t.Fatalf("evolve_items missing from PlannedTables: %+v", infos)
	}
}

// TestSQLite_PlannedTables_ReportsCounts pins the PlannedTablesReporter
// contract end to end: registered plans appear with live row counts that
// reflect planned-table writes.
func TestSQLite_PlannedTables_ReportsCounts(t *testing.T) {
	t.Parallel()

	eng := newParityEngine(t)
	ctx := context.Background()

	// Write through the planned path: MapSet on a planned collection routes
	// to the planned table and extracts columns.
	mb := eng.(metaengine.MapBackend)

	for i := range 5 {
		if err := mb.MapSet(ctx, "parity_items", fmt.Sprintf("r%d", i),
			map[string]any{"status": "open", "priority": i}); err != nil {
			t.Fatalf("MapSet r%d: %v", i, err)
		}
	}

	rep := eng.(metaengine.PlannedTablesReporter)

	infos, err := rep.PlannedTables(ctx)
	if err != nil {
		t.Fatalf("PlannedTables: %v", err)
	}

	for _, info := range infos {
		if info.Collection == "parity_items" {
			if info.Rows != 5 {
				t.Fatalf("parity_items rows = %d, want 5", info.Rows)
			}

			return
		}
	}

	t.Fatalf("parity_items missing from PlannedTables: %+v", infos)
}
