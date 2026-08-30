package adttest

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// collectionPlannedOps is the collection name the planned-ops matrix runs on.
const collectionPlannedOps = "planned_ops_matrix"

// RunPlannedOpsMatrix exercises the full planned-table lifecycle contract
// across engines that implement LayoutPlanApplier (D3 slice 4): pre-layout
// meta_map seeds stay invisible (no-backfill), BackfillPlannedCollection
// copies them, planned writes become visible to scans and MapUpdate, native
// filter/sort pushdown agrees across engines, and EvolveLayoutPlan applies a
// column addition idempotently. Sub-capabilities (MapUpdater,
// LayoutPlanEvolver) are asserted only when the engine implements them, so
// engines can adopt capabilities incrementally without failing the matrix.
//
// Engines that do not implement LayoutPlanApplier are skipped entirely.
//
//	func TestPlannedOpsMatrix(t *testing.T) {
//	    adttest.RunPlannedOpsMatrix(t, []adttest.Factory{
//	        {Name: "sqlite", Create: ...},
//	        {Name: "postgres", Create: ...},
//	    })
//	}
func RunPlannedOpsMatrix(t *testing.T, factories []Factory) {
	t.Helper()

	ctx := context.Background()
	const collection = collectionPlannedOps

	for _, factory := range factories {
		t.Run(factory.Name, func(t *testing.T) {
			t.Parallel()

			eng := factory.Create(t)
			defer metaengine.DeferClose(eng)

			applier, ok := eng.(metaengine.LayoutPlanApplier)
			if !ok {
				t.Skipf("%s does not implement LayoutPlanApplier", factory.Name)

				return
			}

			mb, ok := eng.(metaengine.MapBackend)
			if !ok {
				t.Skipf("%s does not implement MapBackend", factory.Name)

				return
			}

			scan, scanOK := eng.(metaengine.ScanBackend)

			// Pre-layout seeds land in meta_map and must stay invisible to
			// planned scans (the no-backfill contract).
			seedLayoutItem(t, ctx, mb, "pre-a", "open", 1)
			seedLayoutItem(t, ctx, mb, "pre-b", "done", 2)

			plan := metaengine.BuildLayoutPlanFromType[layoutItem](
				collection, []string{"Status"}, []string{"Price"},
			)
			if err := applier.ApplyLayoutPlan(plan); err != nil {
				t.Fatalf("ApplyLayoutPlan: %v", err)
			}

			// Idempotent re-apply with the same columns must not error.
			if err := applier.ApplyLayoutPlan(plan); err != nil {
				t.Fatalf("ApplyLayoutPlan (idempotent): %v", err)
			}

			assertScanCount(t, ctx, scanOK, scan, collection, 0)

			// Opt-in backfill makes the pre-layout rows visible — only on
			// engines with the KeyScanBackend capability (SQL meta_map).
			visible := 0

			backfilled := false

			if _, hasKeys := eng.(metaengine.KeyScanBackend); hasKeys {
				n, err := metaengine.BackfillPlannedCollection(ctx, eng, collection, 1)
				if err != nil {
					t.Fatalf("BackfillPlannedCollection: %v", err)
				}

				if n != 2 {
					t.Fatalf("backfilled %d rows, want 2", n)
				}

				visible += 2
				backfilled = true

				assertScanCount(t, ctx, scanOK, scan, collection, visible)
			}

			// Post-layout writes take the planned path and are visible.
			seedLayoutItem(t, ctx, mb, "post-a", "open", 3)
			seedLayoutItem(t, ctx, mb, "post-b", "open", 5)

			visible += 2

			assertScanCount(t, ctx, scanOK, scan, collection, visible)

			// Native filter+sort pushdown must agree across engines: the
			// open items ascending. pre-a only joins when the backfill leg
			// ran (KeyScanBackend engines); otherwise it stays invisible.
			wantParts := []string{"post-a:3", "post-b:5"}
			if backfilled {
				wantParts = []string{"pre-a:1", "post-a:3", "post-b:5"}
			}

			res, err := pushdownScan(eng, ctx, collection,
				[]metaengine.FilterSpec{{Column: "Status", Op: metaengine.FilterEq, Value: "open"}},
				&metaengine.SortSpec{Column: "Price", Desc: false})
			if err != nil {
				t.Fatalf("PushdownMapScan: %v", err)
			}

			if len(res.Items) != len(wantParts) {
				t.Fatalf("open items: got %d, want %d", len(res.Items), len(wantParts))
			}

			got := canonicalizeItems(res)
			want := strings.Join(wantParts, ",")
			if got != want {
				t.Errorf("open items ascending = %q, want %q", got, want)
			}

			// MapUpdate read-modify-write through the planned table, on a
			// key that exists on every engine (post-layout seed).
			if updater, updOK := eng.(metaengine.MapUpdater); updOK {
				err := updater.MapUpdate(ctx, collection, "post-b", func(prev any) any {
					doc, ok := prev.(map[string]any)
					if !ok {
						return prev
					}

					doc["Price"] = float64(9)

					return doc
				})
				if err != nil {
					t.Fatalf("MapUpdate: %v", err)
				}

				got, found, err := mb.MapGet(ctx, collection, "post-b")
				if err != nil || !found {
					t.Fatalf("MapGet after update: found=%v err=%v", found, err)
				}

				doc, mapOK := got.(map[string]any)
				if !mapOK {
					t.Fatalf("MapGet after update returned %T (%v), want map[string]any", got, got)
				}

				price := firstFloat(doc, "Price", "price")
				if price != 9 {
					t.Errorf("updated Price = %v, want 9", price)
				}
			}

			// Column evolution: adding a column applies once, then nothing.
			if evolver, evoOK := eng.(metaengine.LayoutPlanEvolver); evoOK {
				grown := plan
				grown.Columns = append(grown.Columns, metaengine.PlannedColumn{Name: "Qty", Type: "INTEGER"})

				applied, err := evolver.EvolveLayoutPlan(ctx, grown)
				if err != nil {
					t.Fatalf("EvolveLayoutPlan: %v", err)
				}

				found := false

				for _, action := range applied {
					if strings.HasPrefix(action, "add:") {
						found = true
					}
				}

				if !found {
					t.Errorf("evolution applied = %v, want an add: action", applied)
				}

				applied, err = evolver.EvolveLayoutPlan(ctx, grown)
				if err != nil {
					t.Fatalf("EvolveLayoutPlan (re-run): %v", err)
				}

				if len(applied) != 0 {
					t.Errorf("idempotent re-run applied = %v, want empty", applied)
				}
			}
		})
	}
}

func seedLayoutItem(
	t *testing.T,
	ctx context.Context,
	mb metaengine.MapBackend,
	name, status string, price int,
) {
	t.Helper()

	item := layoutItem{Name: name, Status: status, Price: price}
	if err := mb.MapSet(ctx, collectionPlannedOps, name, item); err != nil {
		t.Fatalf("seed %s: %v", name, err)
	}
}

func assertScanCount(
	t *testing.T,
	ctx context.Context,
	ok bool,
	scan metaengine.ScanBackend,
	collection string,
	want int,
) {
	t.Helper()

	if !ok {
		return
	}

	res, err := scan.MapScan(ctx, collection, nil, nil, nil, 0)
	if err != nil {
		t.Fatalf("MapScan: %v", err)
	}

	if len(res.Items) != want {
		t.Fatalf("MapScan saw %d rows, want %d", len(res.Items), want)
	}
}

func pushdownScan(
	eng metaengine.Engine,
	ctx context.Context,
	collection string,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
) (metaengine.ScanResult, error) {
	ps, ok := eng.(metaengine.PushdownScan)
	if !ok {
		return metaengine.ScanResult{}, errors.New("engine does not implement PushdownScan")
	}

	return ps.PushdownMapScan(ctx, collection, filters, sort, nil, 0)
}

func canonicalizeItems(res metaengine.ScanResult) string {
	parts := make([]string, 0, len(res.Items))

	for _, item := range res.Items {
		doc, ok := item.(map[string]any)
		if !ok {
			continue
		}

		// Engines disagree on map key casing (Go field names vs json tags):
		// accept both so the matrix pins ordering, not key case.
		name, _ := firstString(doc, "Name", "name")
		price := firstFloat(doc, "Price", "price")

		parts = append(parts, fmt.Sprintf("%s:%d", name, int64(price)))
	}

	// Canonical order is by name; the matrix compares filter+sort results,
	// so keep the caller's order (results are already sorted by the query).
	return strings.Join(parts, ",")
}

func firstString(doc map[string]any, keys ...string) (string, bool) {
	for _, k := range keys {
		if v, ok := doc[k].(string); ok {
			return v, true
		}
	}

	return "", false
}

func firstFloat(doc map[string]any, keys ...string) float64 {
	for _, k := range keys {
		if v, ok := doc[k].(float64); ok {
			return v
		}
	}

	return 0
}
