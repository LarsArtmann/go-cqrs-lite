package adttest

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// layoutItem is the test payload used by RunLayoutMatrix.
type layoutItem struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Price  int    `json:"price"`
}

// LayoutScenario describes a single LayoutPlanner capability test.
// Each scenario seeds data, applies a layout, then queries via PushdownScan
// and verifies the results.
type LayoutScenario struct {
	Name         string
	FilterFields []string
	SortFields   []string
	Seed         func(ctx context.Context, mb metaengine.MapBackend) error
	Filter       []metaengine.FilterSpec
	Sort         *metaengine.SortSpec
	Limit        int
	ExpectCount  int
	Canonicalize func(metaengine.ScanResult) string
}

// RunLayoutMatrix tests the LayoutPlanner + PushdownScan capability across
// engines. For each factory that implements LayoutPlanner, it applies a layout,
// seeds data, and verifies that filtered/sorted scans return correct results.
//
// Engines that do not implement LayoutPlanner are automatically skipped.
//
//	func TestLayoutMatrix(t *testing.T) {
//	    adttest.RunLayoutMatrix(t, []adttest.Factory{
//	        {Name: "sqlite", Create: ...},
//	        {Name: "duckdb", Create: ...},
//	    })
//	}
func RunLayoutMatrix(t *testing.T, factories []Factory) {
	t.Helper()

	ctx := context.Background()
	scenarios := layoutScenarios()

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()

			results := make(map[string]string, len(factories))
			var mu sync.Mutex

			for _, factory := range factories {
				t.Run(factory.Name, func(t *testing.T) {
					t.Parallel()

					eng := factory.Create(t)
					defer func() { _ = eng.Close() }()

					// Skip if the engine doesn't implement LayoutPlanner.
					lp, ok := eng.(metaengine.LayoutPlanner)
					if !ok {
						t.Skipf("%s does not implement LayoutPlanner", factory.Name)
						return
					}

					// Skip if the engine doesn't implement PushdownScan.
					ps, ok := eng.(metaengine.PushdownScan)
					if !ok {
						t.Skipf("%s does not implement PushdownScan", factory.Name)
						return
					}

					collection := scenario.Name

					// Apply layout BEFORE seeding (no-backfill semantics).
					if err := lp.ApplyLayout(
						collection,
						scenario.FilterFields,
						scenario.SortFields,
					); err != nil {
						t.Fatalf("%s/%s: ApplyLayout: %v", factory.Name, scenario.Name, err)
					}

					// Idempotent: re-applying with the same fields must not error.
					if err := lp.ApplyLayout(
						collection,
						scenario.FilterFields,
						scenario.SortFields,
					); err != nil {
						t.Fatalf(
							"%s/%s: ApplyLayout (idempotent): %v",
							factory.Name,
							scenario.Name,
							err,
						)
					}

					// Seed data via MapBackend.
					mb, ok := eng.(metaengine.MapBackend)
					if !ok {
						t.Skipf("%s does not implement MapBackend", factory.Name)
						return
					}

					if err := scenario.Seed(ctx, mb); err != nil {
						t.Fatalf("%s/%s: seed: %v", factory.Name, scenario.Name, err)
					}

					// Query via PushdownScan.
					res, err := ps.PushdownMapScan(
						ctx,
						collection,
						scenario.Filter,
						scenario.Sort,
						nil,
						scenario.Limit,
					)
					if err != nil {
						t.Fatalf("%s/%s: PushdownMapScan: %v", factory.Name, scenario.Name, err)
					}

					if scenario.ExpectCount > 0 && len(res.Items) != scenario.ExpectCount {
						t.Errorf("%s/%s: expected %d items, got %d",
							factory.Name, scenario.Name, scenario.ExpectCount, len(res.Items))
					}

					mu.Lock()
					results[factory.Name] = scenario.Canonicalize(res)
					mu.Unlock()
				})
			}

			// Cross-engine parity on the layout scan result.
			t.Cleanup(func() {
				if len(results) < 2 {
					return
				}

				var engines []string
				for name := range results {
					engines = append(engines, name)
				}

				sort.Strings(engines)
				first := results[engines[0]]

				for _, name := range engines[1:] {
					if results[name] != first {
						t.Errorf("%s: cross-engine divergence\n  %s=%s\n  %s=%s",
							scenario.Name, engines[0], first, name, results[name])
					}
				}
			})
		})
	}
}

// RunLayoutConflictTest verifies that applying different column sets to the
// same collection returns ErrLayoutConflict.
func RunLayoutConflictTest(t *testing.T, factories []Factory) {
	t.Helper()

	for _, factory := range factories {
		t.Run(factory.Name, func(t *testing.T) {
			t.Parallel()

			eng := factory.Create(t)
			defer func() { _ = eng.Close() }()

			lp, ok := eng.(metaengine.LayoutPlanner)
			if !ok {
				t.Skipf("%s does not implement LayoutPlanner", factory.Name)
				return
			}

			if err := lp.ApplyLayout("conflict_test", []string{"status"}, nil); err != nil {
				t.Fatalf("first ApplyLayout: %v", err)
			}

			err := lp.ApplyLayout("conflict_test", []string{"name"}, nil)
			if !errors.Is(err, metaengine.ErrLayoutConflict) {
				t.Errorf("expected ErrLayoutConflict, got %v", err)
			}
		})
	}
}

// layoutScenarios returns the test scenarios for the LayoutPlanner matrix.
func layoutScenarios() []LayoutScenario {
	seed := func(ctx context.Context, mb metaengine.MapBackend) error {
		items := []struct {
			key  string
			item layoutItem
		}{
			{"a1", layoutItem{Name: "apple", Status: "active", Price: 100}},
			{"a2", layoutItem{Name: "avocado", Status: "active", Price: 200}},
			{"i1", layoutItem{Name: "banana", Status: "inactive", Price: 50}},
			{"a3", layoutItem{Name: "cherry", Status: "active", Price: 300}},
		}

		for _, it := range items {
			if err := mb.MapSet(ctx, "layout_filter", it.key, it.item); err != nil {
				return err //nolint:wrapcheck
			}

			if err := mb.MapSet(ctx, "layout_sort", it.key, it.item); err != nil {
				return err //nolint:wrapcheck
			}

			if err := mb.MapSet(ctx, "layout_filter_sort", it.key, it.item); err != nil {
				return err //nolint:wrapcheck
			}
		}

		return nil
	}

	canonicalize := func(res metaengine.ScanResult) string {
		var names []string

		for _, item := range res.Items {
			if m, ok := item.(map[string]any); ok {
				if name, ok := m["name"].(string); ok {
					names = append(names, name)
				}
			}
		}

		sort.Strings(names)
		return joinStrings(names)
	}

	return []LayoutScenario{
		{
			Name:         "layout_filter",
			FilterFields: []string{"status"},
			SortFields:   nil,
			Seed:         seed,
			Filter: []metaengine.FilterSpec{
				{Column: "status", Op: metaengine.FilterEq, Value: "active"},
			},
			Sort:         nil,
			Limit:        0,
			ExpectCount:  3,
			Canonicalize: canonicalize,
		},
		{
			Name:         "layout_sort",
			FilterFields: nil,
			SortFields:   []string{"price"},
			Seed:         seed,
			Filter:       nil,
			Sort:         &metaengine.SortSpec{Column: "price", Desc: true},
			Limit:        0,
			ExpectCount:  4,
			Canonicalize: canonicalize,
		},
		{
			Name:         "layout_filter_sort",
			FilterFields: []string{"status"},
			SortFields:   []string{"price"},
			Seed:         seed,
			Filter: []metaengine.FilterSpec{
				{Column: "status", Op: metaengine.FilterEq, Value: "active"},
			},
			Sort:         &metaengine.SortSpec{Column: "price", Desc: true},
			Limit:        0,
			ExpectCount:  3,
			Canonicalize: canonicalize,
		},
	}
}

// joinStrings joins strings without a separator for compact canonical comparison.
func joinStrings(s []string) string {
	var sb strings.Builder
	for _, v := range s {
		sb.WriteString(v)
		sb.WriteByte(',')
	}

	return sb.String()
}
