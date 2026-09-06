//go:build cgo

package duckdbengine_test

import (
	"context"
	"fmt"
	"testing"

	_ "github.com/marcboeker/go-duckdb/v2"

	"database/sql"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
)

// newDuckParityEngine opens an in-memory DuckDB engine with one planned
// collection registered via ApplyLayoutPlan.
func newDuckParityEngine(t *testing.T) metaengine.Engine {
	t.Helper()

	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	eng, err := duckdbengine.New(db)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	plan := metaengine.BuildLayoutPlan("parity_items", []string{"status"}, []string{"priority"})
	if err := eng.(metaengine.LayoutPlanApplier).ApplyLayoutPlan(plan); err != nil {
		t.Fatalf("ApplyLayoutPlan: %v", err)
	}

	return eng
}

// TestDuck_MapScanKeyValues_Paged pins the KeyScanBackend contract: paged
// key+value reads over the BASE meta_map in deterministic key order, with
// hasMore signaling exactly one trailing page.
func TestDuck_MapScanKeyValues_Paged(t *testing.T) {
	t.Parallel()

	eng := newDuckParityEngine(t)
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
		gotKeys []any
		cursor  any
		hasMore = true
		rounds  int
	)

	for hasMore {
		keys, vals, more, err := kb.MapScanKeyValues(ctx, "base_items", cursor, 10)
		if err != nil {
			t.Fatalf("MapScanKeyValues: %v", err)
		}

		if len(keys) != len(vals) {
			t.Fatalf("keys/values length mismatch: %d vs %d", len(keys), len(vals))
		}

		gotKeys = append(gotKeys, keys...)
		hasMore = more
		rounds++

		if len(keys) > 0 {
			cursor = keys[len(keys)-1]
		}

		if rounds > total {
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

// TestDuck_EvolveLayoutPlan pins the LayoutPlanEvolver contract: missing
// columns are added, the operation is idempotent (a second evolve applies
// nothing — this also catches type-alias mismatches between the DDL name
// and information_schema's reported type), and the reporter sees the table.
func TestDuck_EvolveLayoutPlan(t *testing.T) {
	t.Parallel()

	eng := newDuckParityEngine(t)
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
		t.Fatalf("second evolve applied %v, want no-op (type alias mismatch?)", applied)
	}

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

// TestDuck_PlannedTables_ReportsCounts pins the PlannedTablesReporter
// contract end to end: registered plans appear with live row counts that
// reflect planned-table writes.
func TestDuck_PlannedTables_ReportsCounts(t *testing.T) {
	t.Parallel()

	eng := newDuckParityEngine(t)
	ctx := context.Background()

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
