//go:build cgo

package duckdbengine_test

import (
	"context"
	"strings"
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// doctorPlannedItem drives the planned-tables Doctor observation: declaring
// FilterOnField/SortOnField makes Plan auto-create a planned table for the
// collection on the engine.
type doctorPlannedItem struct {
	Name   string
	Status string
	Price  float64
}

type doctorPlannedList struct{}

func doctorPlannedQuery() metaengine.QueryDecl[doctorPlannedList, doctorPlannedItem] {
	return metaengine.Query[doctorPlannedList, doctorPlannedItem](
		"planned_doctor",
		metaengine.OnRecord(
			doctorPlannedItem{},
			func(_ record.Record, e doctorPlannedItem) (string, doctorPlannedItem) {
				return e.Name, e
			},
		),
		metaengine.FilterOnField[doctorPlannedItem]("Status", metaengine.FilterEq),
		metaengine.SortOnField[doctorPlannedItem]("Price", false),
	)
}

// TestDoctor_PlannedTablesSection_RendersRowCounts observes the
// "--- Planned tables ---" Doctor section end to end: after two events flow
// through the Store, the section must name the collection and render the
// LIVE row count (the "row counts now cover all four SQL engines" claim,
// verified rather than inferred from the interface).
func TestDoctor_PlannedTablesSection_RendersRowCounts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	eng, err := duckdbengine.New(dir + "/test.duckdb")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}
	t.Cleanup(func() { _ = eng.Close() })

	store, err := metaengine.Plan([]metaengine.Engine{eng}, doctorPlannedQuery())
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()

	for _, item := range []doctorPlannedItem{
		{Name: "a", Status: "open", Price: 1},
		{Name: "b", Status: "open", Price: 2},
	} {
		if err := store.Apply(ctx, "doctorPlannedItem", item); err != nil {
			t.Fatalf("Apply %s: %v", item.Name, err)
		}
	}

	output := store.Doctor(ctx)

	if !strings.Contains(output, "--- Planned tables ---") {
		t.Fatalf("Doctor missing planned tables section:\n%s", output)
	}

	if !strings.Contains(output, "planned_doctor") {
		t.Fatalf("Doctor planned tables section missing collection name:\n%s", output)
	}

	if !strings.Contains(output, "rows=2") {
		t.Fatalf("Doctor planned tables section must render the live row count rows=2:\n%s", output)
	}
}
