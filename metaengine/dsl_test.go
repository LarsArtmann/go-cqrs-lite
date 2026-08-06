package metaengine_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

type dslInput struct{}

type dslFindByID struct {
	ID string
}

type dslItem struct {
	ID    string
	Name  string
	Count int
}

func TestNewMemoryEngine_Works(t *testing.T) {
	t.Parallel()

	eng := metaengine.NewMemoryEngine()
	defer eng.Close()

	p := eng.Profile()
	if p.Name == "" {
		t.Fatal("engine Profile().Name is empty")
	}
}

func TestPlanFromMemory_OneShot(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[dslFindByID, dslItem](
		"dsl_find",
		metaengine.On(dslItem{}, func(e dslItem) (string, dslItem) {
			return e.ID, e
		}),
	)

	store, err := metaengine.PlanFromMemory(q)
	if err != nil {
		t.Fatalf("PlanFromMemory: %v", err)
	}

	defer store.Close()

	if err := store.Apply(
		context.Background(),
		"dslItem",
		dslItem{ID: "x1", Name: "Widget", Count: 5},
	); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	result, err := store.Execute(dslFindByID{ID: "x1"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	item, ok := result.(dslItem)
	if !ok {
		t.Fatalf("result is %T, want dslItem", result)
	}

	if item.Name != "Widget" || item.Count != 5 {
		t.Fatalf("item = %+v, want Name=Widget Count=5", item)
	}
}

func TestPlanFromMemory_PlansAcrossEngines(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[dslInput, map[string]int64](
		"dsl_counts",
		metaengine.On(dslItem{}, func(e dslItem) metaengine.Delta {
			return metaengine.Delta{e.Name: 1}
		}),
	)

	store, err := metaengine.PlanFromMemory(q)
	if err != nil {
		t.Fatalf("PlanFromMemory: %v", err)
	}

	defer store.Close()

	plan := store.Plan()
	if plan == nil {
		t.Fatal("Plan() returned nil")
	}

	if len(plan.Queries) != 1 {
		t.Fatalf("len(Queries) = %d, want 1", len(plan.Queries))
	}
}

func TestStore_LogPlan_OutputsQueryAssignments(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[dslInput, map[string]int64](
		"logplan_counts",
		metaengine.On(dslItem{}, func(e dslItem) metaengine.Delta {
			return metaengine.Delta{e.Name: 1}
		}),
	)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		q,
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	defer store.Close()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	store.LogPlan(logger)

	output := buf.String()
	if !strings.Contains(output, "logplan_counts") {
		t.Fatalf("LogPlan output does not contain query name:\n%s", output)
	}

	if !strings.Contains(output, "query planned") {
		t.Fatalf("LogPlan output does not contain 'query planned':\n%s", output)
	}
}

func TestStore_LogPlan_NilPlan_NoOp(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[dslInput, map[string]int64](
		"nilplan_counts",
		metaengine.On(dslItem{}, func(e dslItem) metaengine.Delta {
			return metaengine.Delta{e.Name: 1}
		}),
	)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		q,
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	defer store.Close()

	store.LogPlan(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
}
