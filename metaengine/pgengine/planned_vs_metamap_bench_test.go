package pgengine_test

import (
	"context"
	"fmt"
	"testing"

	pgengine "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// plannedVsRows keeps the comparison bench fast enough to run in the
// ephemeral-PG loop while still producing planner-relevant numbers.
const plannedVsRows = 2_000

// BenchmarkPlannedVsMetaMap_Postgres_FilteredScan keeps the "counters, graph,
// and aggregates stay on meta_map for planned collections" decision (ADR-0124
// addendum) under live evidence: the same filtered scan is measured against a
// planless collection (meta_map JSONB predicates) and a planned collection
// (native extracted-column predicates + index). The planned leg should win by
// a wide margin; if it ever does not, the planned pushdown or its indexes
// regressed.
func BenchmarkPlannedVsMetaMap_Postgres_FilteredScan(b *testing.B) {
	dsn := pgDSN(b)

	eng, err := pgengine.New(dsn)
	if err != nil {
		b.Skipf("Postgres not available: %v", err)
	}
	defer eng.Close()

	if _, ok := eng.(metaengine.MapBackend); !ok {
		b.Fatal("engine does not implement MapBackend")
	}
	pushdown, ok := eng.(metaengine.PushdownScan)
	if !ok {
		b.Fatal("engine does not implement PushdownScan")
	}
	applier, ok := eng.(metaengine.LayoutPlanApplier)
	if !ok {
		b.Fatal("engine does not implement LayoutPlanApplier")
	}

	plan := metaengine.BuildLayoutPlanFromType[pgCalibrationPayload](
		"pvb_planned", []string{"status"}, []string{"amount"},
	)
	if err := applier.ApplyLayoutPlan(plan); err != nil {
		b.Fatalf("ApplyLayoutPlan: %v", err)
	}

	populatePGEngine(b, eng, "pvb_metamap", plannedVsRows)
	populatePGEngine(b, eng, "pvb_planned", plannedVsRows)

	filters := []metaengine.FilterSpec{
		{Column: "status", Op: metaengine.FilterEq, Value: "active"},
	}

	b.Run("meta_map", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			res, err := pushdown.PushdownMapScan(
				context.Background(), "pvb_metamap", filters, nil, nil, 0,
			)
			if err != nil {
				b.Fatalf("meta_map scan %d: %v", i, err)
			}

			if len(res.Items) != plannedVsRows/2 {
				b.Fatalf("meta_map scan: got %d rows, want %d", len(res.Items), plannedVsRows/2)
			}
		}
	})

	b.Run("planned", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			res, err := pushdown.PushdownMapScan(
				context.Background(), "pvb_planned", filters, nil, nil, 0,
			)
			if err != nil {
				b.Fatalf("planned scan %d: %v", i, err)
			}

			if len(res.Items) != plannedVsRows/2 {
				b.Fatalf("planned scan: got %d rows, want %d", len(res.Items), plannedVsRows/2)
			}
		}
	})
}

// BenchmarkPlannedVsMetaMap_Postgres_CounterGet evidences the decision that
// counters stay on meta_map even for planned collections: CounterGet on a
// planned collection must cost the same as on a planless one (both hit the
// meta_map counter path). A significant delta here would mean counter routing
// silently changed.
func BenchmarkPlannedVsMetaMap_Postgres_CounterGet(b *testing.B) {
	dsn := pgDSN(b)

	eng, err := pgengine.New(dsn)
	if err != nil {
		b.Skipf("Postgres not available: %v", err)
	}
	defer eng.Close()

	cb, ok := eng.(metaengine.CounterBackend)
	if !ok {
		b.Fatal("engine does not implement CounterBackend")
	}
	applier, ok := eng.(metaengine.LayoutPlanApplier)
	if !ok {
		b.Fatal("engine does not implement LayoutPlanApplier")
	}

	plan := metaengine.BuildLayoutPlanFromType[pgCalibrationPayload](
		"pvb_c_planned", []string{"status"}, nil,
	)
	if err := applier.ApplyLayoutPlan(plan); err != nil {
		b.Fatalf("ApplyLayoutPlan: %v", err)
	}

	const counters = 1_000
	ctx := context.Background()

	for _, col := range []string{"pvb_c_metamap", "pvb_c_planned"} {
		for i := range counters {
			if err := cb.CounterIncrement(ctx, col, metaengine.Delta{fmt.Sprintf("c%d", i): 1}); err != nil {
				b.Fatalf("seed counter %s/%d: %v", col, i, err)
			}
		}
	}

	b.Run("meta_map", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if _, err := cb.CounterGet(ctx, "pvb_c_metamap"); err != nil {
				b.Fatalf("meta_map counter get %d: %v", i, err)
			}
		}
	})

	b.Run("planned", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if _, err := cb.CounterGet(ctx, "pvb_c_planned"); err != nil {
				b.Fatalf("planned counter get %d: %v", i, err)
			}
		}
	})
}
