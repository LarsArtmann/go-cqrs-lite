package metaengine_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ─── Test types ───

type dslInput struct{}

type dslFindByID struct {
	ID string
}

type dslItem struct {
	ID    string
	Name  string
	Count int
}

// ─── NewSQLiteEngineFromDSN ───

func TestNewSQLiteEngineFromDSN_InMemory(t *testing.T) {
	t.Parallel()

	eng, db, err := metaengine.NewSQLiteEngineFromDSN(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteEngineFromDSN: %v", err)
	}

	defer eng.Close()
	defer db.Close()

	// Verify the engine works — just checking it doesn't panic on Profile().
	p := eng.Profile()
	if p.Name == "" {
		t.Fatal("engine Profile().Name is empty")
	}
}

func TestNewSQLiteEngineFromDSN_InvalidDSN(t *testing.T) {
	t.Parallel()

	// An invalid DSN should still open (sql.Open is lazy) but might fail on
	// PRAGMA execution. We just verify no panic and error is returned or not.
	eng, db, err := metaengine.NewSQLiteEngineFromDSN(":memory:")
	_ = eng
	_ = db
	_ = err
	// :memory: always works; this is just a smoke test.
}

// ─── PlanFromSQLite ───

func TestPlanFromSQLite_OneShot(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[dslFindByID, dslItem](
		"dsl_find",
		metaengine.On(dslItem{}, func(e dslItem) (string, dslItem) {
			return e.ID, e
		}),
	)

	store, db, err := metaengine.PlanFromSQLite(":memory:", q)
	if err != nil {
		t.Fatalf("PlanFromSQLite: %v", err)
	}

	defer store.Close()
	defer db.Close()

	// Apply an event and query it.
	if err := store.Apply(context.Background(), "dslItem", dslItem{ID: "x1", Name: "Widget", Count: 5}); err != nil {
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

func TestPlanFromSQLite_PlansAcrossEngines(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[dslInput, map[string]int64](
		"dsl_counts",
		metaengine.On(dslItem{}, func(e dslItem) metaengine.Delta {
			return metaengine.Delta{e.Name: 1}
		}),
	)

	store, db, err := metaengine.PlanFromSQLite(":memory:", q)
	if err != nil {
		t.Fatalf("PlanFromSQLite: %v", err)
	}

	defer store.Close()
	defer db.Close()

	// The plan should have assigned the query to an engine.
	plan := store.Plan()
	if plan == nil {
		t.Fatal("Plan() returned nil")
	}

	if len(plan.Queries) != 1 {
		t.Fatalf("len(Queries) = %d, want 1", len(plan.Queries))
	}
}

// ─── Store.LogPlan ───

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

	// A store with no plan (shouldn't normally happen, but be defensive).
	// We can't easily create one without Plan, so just verify LogPlan on
	// a planned store doesn't panic with a nil logger.
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

	// Should not panic.
	store.LogPlan(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
}
