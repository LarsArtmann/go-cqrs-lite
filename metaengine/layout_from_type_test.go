package metaengine_test

import (
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

type layoutFromTypeDoc struct {
	Priority int     `json:"priority"`
	Score    float64 `json:"score"`
	Status   string  `json:"status"`
	Title    string
	hidden   string //nolint:unused // field-type reflection target
}

func TestBuildLayoutPlanFromType_ReflectionDerivedColumns(t *testing.T) {
	plan := metaengine.BuildLayoutPlanFromType[layoutFromTypeDoc](
		"docs", []string{"status"}, []string{"priority", "score"},
	)

	if plan.Collection != "docs" || plan.Table != "meta_planned_docs" {
		t.Fatalf("plan identity: %+v", plan)
	}

	types := map[string]string{}
	for _, c := range plan.Columns {
		types[c.Name] = c.Type
	}

	// Go field types are the truth: int → INTEGER, float → REAL, string → TEXT.
	if types["priority"] != "INTEGER" {
		t.Errorf("priority = %q, want INTEGER (name heuristic would lie here)", types["priority"])
	}

	if types["score"] != "REAL" {
		t.Errorf("score = %q, want REAL", types["score"])
	}

	if types["status"] != "TEXT" {
		t.Errorf("status = %q, want TEXT", types["status"])
	}

	// One index per column, dedup'd (priority appears as sort only once).
	if len(plan.Indexes) != len(plan.Columns) {
		t.Fatalf("indexes = %d, columns = %d", len(plan.Indexes), len(plan.Columns))
	}
}

func TestBuildLayoutPlanFromType_UnknownFieldFallsBackToHeuristic(t *testing.T) {
	// "nonexistent" is not a field of the struct — the name heuristic
	// decides (keeps map-shaped rows working).
	plan := metaengine.BuildLayoutPlanFromType[layoutFromTypeDoc](
		"docs", []string{"nonexistent_count"}, nil,
	)

	if plan.Columns[0].Type != "INTEGER" {
		t.Fatalf("heuristic fallback = %q, want INTEGER", plan.Columns[0].Type)
	}
}

func TestBuildLayoutPlanFromType_MatchesBuildLayoutPlanShape(t *testing.T) {
	fromType := metaengine.BuildLayoutPlanFromType[layoutFromTypeDoc]("d", []string{"status"}, []string{"priority"})
	heuristic := metaengine.BuildLayoutPlan("d", []string{"status"}, []string{"priority"})

	if fromType.Table != heuristic.Table {
		t.Fatalf("table mismatch: %q vs %q", fromType.Table, heuristic.Table)
	}

	if len(fromType.Indexes) != len(heuristic.Indexes) {
		t.Fatalf("index count mismatch")
	}
}
