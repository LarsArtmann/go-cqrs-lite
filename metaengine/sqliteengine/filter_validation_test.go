package sqliteengine

import (
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestBuildPlannedSelectQueryRejectsInvalidOp pins the SQL-build layer's
// defense in depth (M04/F12): even if an invalid FilterOp bypasses the
// scan-entry validation, the planned-query builder refuses to render it
// instead of splicing raw operator text into the statement.
func TestBuildPlannedSelectQueryRejectsInvalidOp(t *testing.T) {
	t.Parallel()

	plan := metaengine.LayoutPlan{Table: "tasks_layout"}

	for _, op := range []metaengine.FilterOp{"1=1; DROP TABLE tasks_layout; --", "LIKE", ""} {
		_, _, err := buildPlannedSelectQuery(
			plan,
			[]metaengine.FilterSpec{{Column: "status", Op: op, Value: "open"}},
			nil, nil, 10,
		)
		if err == nil {
			t.Errorf("buildPlannedSelectQuery must reject invalid op %q", string(op))
			continue
		}

		if !strings.Contains(err.Error(), "invalid filter operator") {
			t.Errorf("error should name the invalid operator, got: %v", err)
		}
	}

	query, args, err := buildPlannedSelectQuery(
		plan,
		[]metaengine.FilterSpec{{Column: "status", Op: metaengine.FilterEq, Value: "open"}},
		nil, nil, 10,
	)
	if err != nil {
		t.Fatalf("valid op must build, got: %v", err)
	}

	if !strings.Contains(query, `"status" = ?`) || len(args) != 2 {
		t.Errorf("valid filter must render the binary comparison, query=%q args=%v", query, args)
	}
}

// TestBuildStreamQueryRejectsHostileInput pins the standard-path scan builder
// (json_extract paths): hostile operator or column yields an error instead of
// a rendered statement.
func TestBuildStreamQueryRejectsHostileInput(t *testing.T) {
	t.Parallel()

	e := &sqliteEngine{}

	_, _, err := e.buildStreamQuery(
		"tasks",
		[]metaengine.FilterSpec{{Column: "status", Op: "1=1; --", Value: "open"}},
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "invalid filter operator") {
		t.Errorf("buildStreamQuery must reject hostile op, got: %v", err)
	}

	_, _, err = e.buildStreamQuery(
		"tasks",
		[]metaengine.FilterSpec{{Column: "x') OR ('1'='1", Op: metaengine.FilterEq, Value: "open"}},
		nil,
	)
	if err == nil {
		t.Error("buildStreamQuery must reject hostile column")
	}

	query, args, err := e.buildStreamQuery(
		"tasks",
		[]metaengine.FilterSpec{{Column: "status", Op: metaengine.FilterEq, Value: "open"}},
		nil,
	)
	if err != nil {
		t.Fatalf("benign filter must build, got: %v", err)
	}

	if !strings.Contains(query, "json_extract(value, '$.status') = ?") || len(args) != 2 {
		t.Errorf("benign filter must render the json_extract comparison, query=%q args=%v", query, args)
	}
}
