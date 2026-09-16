package metaengine_test

import (
	"context"
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestExplainPlanAndDoctor_VectorPathLabel pins the k-NN path observability
// (ADR-0140): engines implementing VectorPathReporter get a per-engine path
// line in both ExplainPlan and Doctor; engines that cannot report a path
// render none.
func TestExplainPlanAndDoctor_VectorPathLabel(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	memory := metaengine.NewMemoryEngine()

	if err := memory.(metaengine.VectorBackend).VectorInsert(ctx, "docs", metaengine.Embedding{
		ID: "a", Values: []float32{1},
	}); err != nil {
		t.Fatalf("VectorInsert: %v", err)
	}

	store, err := metaengine.Plan(
		[]metaengine.Engine{
			memory,
			&fakeVectorOnlyEngine{
				fakeEngine: &fakeEngine{profile: metaengine.EngineProfile{Name: "scanonly"}},
			},
		},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	t.Cleanup(func() { _ = store.Close() })

	plan := store.ExplainPlan()

	if !strings.Contains(plan, "memory: k-NN path "+metaengine.VectorPathScan) {
		t.Errorf("ExplainPlan should report the memory engine's k-NN path, got:\n%s", plan)
	}

	if strings.Contains(plan, "scanonly: k-NN path") {
		t.Errorf("engines without VectorPathReporter must not render a path line, got:\n%s", plan)
	}

	report := store.Doctor(ctx)

	if !strings.Contains(report, "memory: k-NN path "+metaengine.VectorPathScan) {
		t.Errorf("Doctor should report the memory engine's k-NN path, got:\n%s", report)
	}
}

// TestExplainPlan_NoVectorEnginesNoSection pins that stores without any
// vector-capable engine keep the old ExplainPlan shape (no Vectors section).
func TestExplainPlan_NoVectorEnginesNoSection(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan(
		[]metaengine.Engine{
			&fakeEngine{profile: metaengine.EngineProfile{
				Name: "plain",
				Supports: map[metaengine.ADT]metaengine.Complexity{
					metaengine.ADTMap: metaengine.ComplexityO1,
				},
			}},
		},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	t.Cleanup(func() { _ = store.Close() })

	if plan := store.ExplainPlan(); strings.Contains(plan, "--- Vectors ---") {
		t.Errorf("store without vector engines must not render a Vectors section, got:\n%s", plan)
	}
}
