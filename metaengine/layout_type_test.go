package metaengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

type layoutSample struct {
	Status   string
	Priority int
	Score    float64
	Active   bool
	Name     string
}

func TestBuildLayoutPlanFromType_InfersColumnTypes(t *testing.T) {
	t.Parallel()

	plan := metaengine.BuildLayoutPlanFromType[layoutSample](
		"tasks", []string{"status", "priority"}, []string{"score", "name"},
	)

	got := map[string]string{}
	for _, c := range plan.Columns {
		got[c.Name] = c.Type
	}

	cases := map[string]string{
		"status":   "TEXT",    // string
		"priority": "INTEGER", // int
		"score":    "DOUBLE",  // float64
		"name":     "TEXT",    // string
	}

	for field, want := range cases {
		if got[field] != want {
			t.Errorf("column %s: got %s, want %s", field, got[field], want)
		}
	}

	// All four unique fields are present (status+priority+score+name).
	if len(plan.Columns) != 4 {
		t.Fatalf("expected 4 columns, got %d", len(plan.Columns))
	}
}

func TestBuildLayoutPlanFromType_FallsBackToTextForUnknown(t *testing.T) {
	t.Parallel()

	// "unknown" is not a field on layoutSample → name heuristic → TEXT.
	plan := metaengine.BuildLayoutPlanFromType[layoutSample](
		"tasks", []string{"unknown"}, nil,
	)

	if len(plan.Columns) != 1 || plan.Columns[0].Type != "TEXT" {
		t.Fatalf("unknown field should fall back to TEXT, got %+v", plan.Columns)
	}
}
