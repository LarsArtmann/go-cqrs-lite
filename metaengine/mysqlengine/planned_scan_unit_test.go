package mysqlengine

import (
	"errors"
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Unit tests for the planned-table pushdown query builder and the
// mis-type validation (no live MariaDB required).

func testPlannedScanPlan() metaengine.LayoutPlan {
	return metaengine.LayoutPlan{
		Collection: "planned_scan",
		Table:      "meta_planned_planned_scan",
		Columns: []metaengine.PlannedColumn{
			{Name: "priority", Type: "INTEGER"},
			{Name: "status", Type: "TEXT"},
		},
	}
}

func TestBuildPlannedScanQuery_FilterSortKeysetLimit(t *testing.T) {
	plan := testPlannedScanPlan()

	query, args, err := buildPlannedScanQuery(
		plan,
		[]metaengine.FilterSpec{
			{Column: "status", Op: metaengine.FilterEq, Value: "open"},
		},
		&metaengine.SortSpec{Column: "priority", Desc: true},
		float64(7),
		10,
	)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	for _, want := range []string{
		"SELECT CAST(value AS CHAR) FROM `meta_planned_planned_scan`",
		"`status` = ?",
		"`priority` < ?",
		"ORDER BY `priority` DESC",
		"LIMIT 11",
	} {
		if !strings.Contains(query, want) {
			t.Fatalf("query missing %q:\n%s", want, query)
		}
	}

	if len(args) != 2 || args[0] != "open" || args[1] != float64(7) {
		t.Fatalf("args = %#v", args)
	}
}

func TestBuildPlannedScanQuery_MisTypedFilterIsClassifiedRejection(t *testing.T) {
	plan := testPlannedScanPlan()

	_, _, err := buildPlannedScanQuery(
		plan,
		[]metaengine.FilterSpec{
			{Column: "priority", Op: metaengine.FilterEq, Value: "high"},
		},
		nil, nil, 0,
	)
	if !errors.Is(err, metaengine.ErrPlannedColumnTypeMismatch) {
		t.Fatalf("want ErrPlannedColumnTypeMismatch, got %v", err)
	}

	if _, _, err := buildPlannedScanQuery(
		plan, nil,
		&metaengine.SortSpec{Column: "priority"},
		"not-a-number", 0,
	); !errors.Is(err, metaengine.ErrPlannedColumnTypeMismatch) {
		t.Fatalf("cursor: want mismatch, got %v", err)
	}

	if _, _, err := buildPlannedScanQuery(
		plan,
		[]metaengine.FilterSpec{
			{Column: "status", Op: metaengine.FilterIn, Value: []any{"a", float64(1)}},
		},
		nil, nil, 0,
	); !errors.Is(err, metaengine.ErrPlannedColumnTypeMismatch) {
		t.Fatalf("IN member: want mismatch, got %v", err)
	}
}
