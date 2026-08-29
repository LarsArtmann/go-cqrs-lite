package metaengine_test

import (
	"context"
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// fakeVectorOnlyEngine implements VectorBackend but NOT VectorCounter: it is
// the full-scan, no-size-introspection shape the WARN exists for.
type fakeVectorOnlyEngine struct {
	*fakeEngine
}

func (e *fakeVectorOnlyEngine) VectorInsert(
	_ context.Context, _ string, _ metaengine.Embedding,
) error {
	return nil
}

func (e *fakeVectorOnlyEngine) VectorSearch(
	_ context.Context, _ string, _ []float32, _ int, _ string,
) ([]metaengine.VectorResult, error) {
	return nil, nil
}

func TestVectorCounter_MemoryEngineCountsAndLists(t *testing.T) {
	t.Parallel()

	eng := metaengine.NewMemoryEngine()
	ctx := context.Background()

	vc, ok := eng.(metaengine.VectorCounter)
	if !ok {
		t.Fatal("memory engine must implement metaengine.VectorCounter")
	}

	vb := eng.(metaengine.VectorBackend)

	n, err := vc.VectorCount(ctx, "missing")
	if err != nil || n != 0 {
		t.Fatalf("unknown collection count = (%d, %v), want (0, nil)", n, err)
	}

	for i := range 3 {
		if err := vb.VectorInsert(ctx, "docs", metaengine.Embedding{
			ID: string(rune('a' + i)), Values: []float32{1, 2},
		}); err != nil {
			t.Fatalf("VectorInsert: %v", err)
		}
	}

	if err := vb.VectorInsert(ctx, "other", metaengine.Embedding{
		ID: "x", Values: []float32{3},
	}); err != nil {
		t.Fatalf("VectorInsert: %v", err)
	}

	if n, err := vc.VectorCount(ctx, "docs"); err != nil || n != 3 {
		t.Fatalf("docs count = (%d, %v), want (3, nil)", n, err)
	}

	cols, err := vc.VectorCollections(ctx)
	if err != nil {
		t.Fatalf("VectorCollections: %v", err)
	}

	if len(cols) != 2 || cols[0] != "docs" || cols[1] != "other" {
		t.Fatalf("collections = %v, want [docs other]", cols)
	}
}

func TestDoctor_VectorSectionCountsAndWarns(t *testing.T) {
	t.Parallel()

	memory := metaengine.NewMemoryEngine()
	ctx := context.Background()

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

	report := store.Doctor(ctx)

	if !strings.Contains(report, "memory/docs: 1 vectors") {
		t.Errorf("Doctor should report memory vector count, got:\n%s", report)
	}

	if !strings.Contains(report, "scanonly: WARN full-scan vector search") {
		t.Errorf("Doctor should WARN for counter-less vector engine, got:\n%s", report)
	}
}

func TestExplainPlan_VectorFullScanWarning(t *testing.T) {
	t.Parallel()

	scanOnly := &fakeVectorOnlyEngine{
		fakeEngine: &fakeEngine{profile: metaengine.EngineProfile{
			Name: "scanonly",
			Supports: map[metaengine.ADT]metaengine.Complexity{
				metaengine.ADTMap: metaengine.ComplexityO1,
			},
		}},
	}

	store, err := metaengine.Plan(
		[]metaengine.Engine{scanOnly},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	t.Cleanup(func() { _ = store.Close() })

	plan := store.ExplainPlan()

	if !strings.Contains(plan, "WARN vector: scanonly serves k-NN by full scan") {
		t.Errorf("ExplainPlan should carry the full-scan vector WARN, got:\n%s", plan)
	}

	memoryStore, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	t.Cleanup(func() { _ = memoryStore.Close() })

	if plan := memoryStore.ExplainPlan(); strings.Contains(plan, "WARN vector") {
		t.Errorf("memory engine has VectorCounter; plan must not WARN, got:\n%s", plan)
	}
}
