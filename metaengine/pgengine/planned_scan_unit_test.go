package pgengine

import (
	"errors"
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Unit tests for the planned-table pushdown query builder and the
// mis-type validation (no live Postgres required).

func testPlannedScanPlan() metaengine.LayoutPlan {
	return metaengine.LayoutPlan{
		Collection: "planned_scan",
		Table:      "meta_planned_planned_scan",
		Columns: []metaengine.PlannedColumn{
			{Name: "priority", Type: "INTEGER"},
			{Name: "score", Type: "REAL"},
			{Name: "status", Type: "TEXT"},
		},
	}
}

func TestBuildPGPlannedScanQuery_FilterSortKeysetLimit(t *testing.T) {
	plan := testPlannedScanPlan()

	query, args, err := buildPGPlannedScanQuery(
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
		`SELECT value::text FROM "meta_planned_planned_scan"`,
		`"status" = $1`,
		`"priority" < $2`,
		`ORDER BY "priority" DESC`,
		`LIMIT 11`,
	} {
		if !strings.Contains(query, want) {
			t.Fatalf("query missing %q:\n%s", want, query)
		}
	}

	if len(args) != 2 || args[0] != "open" || args[1] != float64(7) {
		t.Fatalf("args = %#v", args)
	}
}

func TestBuildPGPlannedScanQuery_AscKeysetAndIN(t *testing.T) {
	plan := testPlannedScanPlan()

	query, args, err := buildPGPlannedScanQuery(
		plan,
		[]metaengine.FilterSpec{
			{Column: "status", Op: metaengine.FilterIn, Value: []any{"a", "b"}},
		},
		&metaengine.SortSpec{Column: "priority"},
		int64(3),
		0,
	)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	for _, want := range []string{
		`"status" IN ($1, $2)`,
		`"priority" > $3`,
		`ORDER BY "priority"`,
	} {
		if !strings.Contains(query, want) {
			t.Fatalf("query missing %q:\n%s", want, query)
		}
	}

	if len(args) != 3 {
		t.Fatalf("args = %#v", args)
	}
}

func TestBuildPGPlannedScanQuery_NoFiltersNoSort(t *testing.T) {
	plan := testPlannedScanPlan()

	query, args, err := buildPGPlannedScanQuery(plan, nil, nil, nil, 5)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if strings.Contains(query, "WHERE") || strings.Contains(query, "ORDER BY") {
		t.Fatalf("unexpected clauses: %s", query)
	}

	if !strings.Contains(query, "LIMIT 6") || len(args) != 0 {
		t.Fatalf("query = %q, args = %#v", query, args)
	}
}

func TestBuildPGPlannedScanQuery_MisTypedFilterIsClassifiedRejection(t *testing.T) {
	plan := testPlannedScanPlan()

	cases := []struct {
		name   string
		column string
		val    any
	}{
		{"string on INTEGER", "priority", "high"},
		{"string on REAL", "score", "7.5"},
		{"float on TEXT", "status", float64(1)},
		{"bool on TEXT", "status", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := buildPGPlannedScanQuery(
				plan,
				[]metaengine.FilterSpec{
					{Column: tc.column, Op: metaengine.FilterEq, Value: tc.val},
				},
				nil, nil, 0,
			)
			if !errors.Is(err, metaengine.ErrPlannedColumnTypeMismatch) {
				t.Fatalf("want ErrPlannedColumnTypeMismatch, got %v", err)
			}
		})
	}
}

func TestBuildPGPlannedScanQuery_MisTypedCursorAndINMembers(t *testing.T) {
	plan := testPlannedScanPlan()

	if _, _, err := buildPGPlannedScanQuery(
		plan, nil,
		&metaengine.SortSpec{Column: "priority"},
		"not-a-number", 0,
	); !errors.Is(err, metaengine.ErrPlannedColumnTypeMismatch) {
		t.Fatalf("cursor: want mismatch, got %v", err)
	}

	if _, _, err := buildPGPlannedScanQuery(
		plan,
		[]metaengine.FilterSpec{
			{Column: "status", Op: metaengine.FilterIn, Value: []any{"ok", float64(3)}},
		},
		nil, nil, 0,
	); !errors.Is(err, metaengine.ErrPlannedColumnTypeMismatch) {
		t.Fatalf("IN member: want mismatch, got %v", err)
	}
}

func TestBuildPGPlannedScanQuery_TypedValuesPass(t *testing.T) {
	plan := testPlannedScanPlan()

	if _, _, err := buildPGPlannedScanQuery(
		plan,
		[]metaengine.FilterSpec{
			{Column: "priority", Op: metaengine.FilterGe, Value: int64(3)},
			{Column: "score", Op: metaengine.FilterLt, Value: 9.5},
			{Column: "status", Op: metaengine.FilterEq, Value: "open"},
		},
		&metaengine.SortSpec{Column: "score", Desc: true},
		1.5,
		25,
	); err != nil {
		t.Fatalf("typed values must pass: %v", err)
	}
}
