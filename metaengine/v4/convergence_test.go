package metaengine_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestConvergence_PlanCarriesLayout verifies that Plan() records a LayoutOption
// in each QueryAssignment — the core convergence of ADR-0124 (layout decisions
// live in the plan, not in a separate ReplanLayout call).
func TestConvergence_PlanCarriesLayout(t *testing.T) {
	t.Parallel()

	eng := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "conv-engine",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
		Layouts: map[metaengine.ADT]metaengine.StorageLayout{
			metaengine.ADTMap: metaengine.LayoutKV,
		},
	}}

	store, err := metaengine.Plan([]metaengine.Engine{eng}, findTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	plan := store.Plan()
	if plan == nil || len(plan.Queries) == 0 {
		t.Fatal("expected non-empty plan")
	}

	for _, q := range plan.Queries {
		if q.Layout == "" {
			t.Errorf(
				"query %s has empty Layout — plan should carry the layout decision",
				q.QueryName,
			)
		}
	}
}

// TestConvergence_SetPriorityUpdatesLayoutInPlan verifies that after
// SetPriority, the plan's Layout field reflects the new priority's intended
// layout. This closes the split-brain where SetPriority stored the config but
// Replan's planConfig didn't carry it.
func TestConvergence_SetPriorityUpdatesLayoutInPlan(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	eng := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "conv-kv",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
		Layouts: map[metaengine.ADT]metaengine.StorageLayout{
			metaengine.ADTMap: metaengine.LayoutKV,
		},
	}}

	store, err := metaengine.Plan([]metaengine.Engine{eng}, findTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	// KV + Balanced → Embed (verified by the 16-combo matrix test).
	beforeLayout := store.Plan().Queries[0].Layout
	if beforeLayout != metaengine.LayoutEmbed {
		t.Fatalf("KV+Balanced should be Embed, got %s", beforeLayout)
	}

	// Switch to WriteSpeed → KV should flip to Normalize.
	if err := store.SetPriority(ctx, &metaengine.PriorityConfig{
		Global: metaengine.PriorityWriteSpeed,
	}); err != nil {
		t.Fatalf("SetPriority: %v", err)
	}

	afterLayout := store.Plan().Queries[0].Layout
	if afterLayout != metaengine.LayoutNormalize {
		t.Errorf("KV+WriteSpeed should be Normalize after SetPriority, got %s", afterLayout)
	}
}

// TestConvergence_ReplanLayoutReadsActualLayout verifies that ReplanLayout
// diffs against the actual layout in the plan (not a hardcoded LayoutEmbed).
// After SetPriority updates the plan's layout, a subsequent ReplanLayout with
// the same priority should produce NO diffs (current == desired).
func TestConvergence_ReplanLayoutReadsActualLayout(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	eng := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "conv-kv2",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
		Layouts: map[metaengine.ADT]metaengine.StorageLayout{
			metaengine.ADTMap: metaengine.LayoutKV,
		},
	}}

	pc := &metaengine.PriorityConfig{Global: metaengine.PriorityWriteSpeed}

	store, err := metaengine.Plan([]metaengine.Engine{eng}, findTaskQuery(),
		metaengine.WithPriorityConfig(pc),
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	// The plan was computed with WriteSpeed → Normalize.
	currentLayout := store.Plan().Queries[0].Layout
	if currentLayout != metaengine.LayoutNormalize {
		t.Fatalf("KV+WriteSpeed should be Normalize, got %s", currentLayout)
	}

	// ReplanLayout with the SAME priority should produce no diffs because
	// the current layout already matches.
	diffs, err := store.ReplanLayout(ctx, pc)
	if err != nil {
		t.Fatalf("ReplanLayout: %v", err)
	}

	for _, d := range diffs {
		if d.From == d.To {
			t.Errorf(
				"ReplanLayout produced a no-op diff for %s: %s → %s (should have been suppressed)",
				d.QueryName,
				d.From,
				d.To,
			)
		}
	}
}
