package mysqlengine

import (
	"errors"
	"fmt"
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
		"SELECT CAST(value AS CHAR), `key` FROM `meta_planned_planned_scan`",
		"`status` = ?",
		"`priority` < ?",
		"ORDER BY `priority` DESC, `key`",
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

func TestBuildPlannedScanQuery_CompoundCursor(t *testing.T) {
	plan := testPlannedScanPlan()

	query, args, err := buildPlannedScanQuery(
		plan,
		[]metaengine.FilterSpec{
			{Column: "status", Op: metaengine.FilterEq, Value: "open"},
		},
		&metaengine.SortSpec{Column: "priority", Desc: true},
		metaengine.SortKeyCursor{Sort: int64(7), Key: []byte("k-1")},
		10,
	)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	for _, want := range []string{
		"SELECT CAST(value AS CHAR), `key` FROM `meta_planned_planned_scan`",
		"(`priority` < ? OR `priority` = ? AND `key` > ?)",
		"ORDER BY `priority` DESC, `key`",
	} {
		if !strings.Contains(query, want) {
			t.Fatalf("query missing %q:\n%s", want, query)
		}
	}

	wantArgs := []any{"open", int64(7), int64(7), "k-1"}
	if len(args) != len(wantArgs) {
		t.Fatalf("args = %#v, want %#v", args, wantArgs)
	}

	for i := range wantArgs {
		if fmt.Sprintf("%v", args[i]) != fmt.Sprintf("%v", wantArgs[i]) {
			t.Fatalf("args[%d] = %#v, want %#v", i, args[i], wantArgs[i])
		}
	}
}

func TestBuildPlannedScanQuery_CompoundCursorSortValidated(t *testing.T) {
	plan := testPlannedScanPlan()

	if _, _, err := buildPlannedScanQuery(
		plan, nil,
		&metaengine.SortSpec{Column: "priority"},
		metaengine.SortKeyCursor{Sort: "not-a-number", Key: []byte("k-1")},
		0,
	); !errors.Is(err, metaengine.ErrPlannedColumnTypeMismatch) {
		t.Fatalf("compound sort: want mismatch, got %v", err)
	}
}

func TestBuildPlannedScanQuery_CompoundCursorNormalizesFloatSort(t *testing.T) {
	plan := testPlannedScanPlan()

	_, args, err := buildPlannedScanQuery(
		plan, nil,
		&metaengine.SortSpec{Column: "priority"},
		metaengine.SortKeyCursor{Sort: float64(7), Key: []byte("k-1")},
		0,
	)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	// JSON round-trips turn 7 into float64; the bind must come back as int64
	// so it compares cleanly against the INTEGER column.
	for i, arg := range args {
		if _, ok := arg.(float64); ok {
			t.Fatalf("args[%d] = float64(%v), want int64 (normalized)", i, arg)
		}
	}
}
