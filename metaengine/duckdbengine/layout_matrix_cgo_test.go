//go:build cgo

package duckdbengine_test

import (
	"context"
	"strings"
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// layout_matrix_cgo_test.go runs the LayoutPlanner capability matrix on the
// DuckDB engine. The memory engine is included but auto-skipped (it does not
// implement LayoutPlanner). Cross-engine parity applies when 2+ engines
// implement the capability.

func TestDuckDBLayoutMatrix(t *testing.T) {
	t.Parallel()

	adttest.RunLayoutMatrix(t, []adttest.Factory{
		{
			Name:   "memory",
			Create: func(t *testing.T) metaengine.Engine { return metaengine.NewMemoryEngine() },
		},
		{
			Name: "duckdb",
			Create: func(t *testing.T) metaengine.Engine {
				t.Helper()

				eng, err := duckdbengine.New("")
				if err != nil {
					t.Skipf("DuckDB not available: %v", err)
				}

				return eng
			},
		},
	})
}

func TestDuckDBLayoutConflict(t *testing.T) {
	t.Parallel()

	adttest.RunLayoutConflictTest(t, []adttest.Factory{
		{
			Name:   "memory",
			Create: func(t *testing.T) metaengine.Engine { return metaengine.NewMemoryEngine() },
		},
		{
			Name: "duckdb",
			Create: func(t *testing.T) metaengine.Engine {
				t.Helper()

				eng, err := duckdbengine.New("")
				if err != nil {
					t.Skipf("DuckDB not available: %v", err)
				}

				return eng
			},
		},
	})
}

func TestDuckDBExplainScanQuery(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}
	defer eng.Close()

	// Standard path (no layout): should use json_extract.
	stdSQL, stdArgs := eng.(metaengine.ExplainableScan).ExplainScanQuery(
		ctx, "std_col", metaengine.ExplainOptions{
			Filters: []metaengine.FilterSpec{
				{Column: "status", Op: metaengine.FilterEq, Value: "active"},
			},
			Limit: 10,
		},
	)

	if !strings.Contains(stdSQL, "json_extract") {
		t.Errorf("standard explain: expected json_extract in SQL, got: %s", stdSQL)
	}

	if len(stdArgs) < 2 {
		t.Errorf("standard explain: expected at least 2 args, got %d", len(stdArgs))
	}

	// Planned path: should use direct column references (no json_extract).
	lp := eng.(metaengine.LayoutPlanner)
	if err := lp.ApplyLayout("explain_col", []string{"status"}, []string{"price"}); err != nil {
		t.Fatalf("ApplyLayout: %v", err)
	}

	plannedSQL, _ := eng.(metaengine.ExplainableScan).ExplainScanQuery(
		ctx, "explain_col", metaengine.ExplainOptions{
			Filters: []metaengine.FilterSpec{
				{Column: "status", Op: metaengine.FilterEq, Value: "active"},
			},
			Limit: 10,
		},
	)

	if strings.Contains(plannedSQL, "json_extract") {
		t.Errorf("planned explain: expected NO json_extract in SQL, got: %s", plannedSQL)
	}

	if !strings.Contains(plannedSQL, "meta_planned_explain_col") {
		t.Errorf("planned explain: expected planned table name, got: %s", plannedSQL)
	}
}
