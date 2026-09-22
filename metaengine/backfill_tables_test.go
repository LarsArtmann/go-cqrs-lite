package metaengine_test

import (
	"errors"
	"strings"
	"testing"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// findTaskResults is the collection-shaped read result (a struct with a
// slice field) — the shape ExecuteTyped reconstructs from ScanResult.
type findTaskResults struct {
	Items []FindTaskResult
}

// filteredTaskQuery is findTaskQuery plus a declared filter, so the
// auto-layout rule registers a planned table for the "find_task" collection.
func filteredTaskQuery() metaengine.QueryDecl[FindTask, FindTaskResult] {
	return metaengine.Query[FindTask, FindTaskResult](
		"find_task",
		metaengine.OnRecord(
			TaskCreated{},
			func(_ record.Record, e TaskCreated) (TaskID, FindTaskResult) {
				return e.ID, FindTaskResult{
					ID: e.ID, Title: e.Title, Assignee: e.Assignee,
					Status: e.Status, Priority: e.Priority,
				}
			},
		),
		metaengine.FilterOnField[FindTaskResult]("status", metaengine.FilterEq),
	)
}

// TestBackfillPlannedTables_AfterData pins the register-after-data story
// (G-T12): rows written into meta_map BEFORE the planned table is registered
// are copied into it by Store.BackfillPlannedTables, after which planned-path
// reads see them. Idempotent: a second run re-upserts to the same state.
func TestBackfillPlannedTables_AfterData(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	eng, err := sqliteengine.NewSQLiteEngineFromDSN(t.TempDir() + "/backfill.db")
	if err != nil {
		t.Fatalf("sqliteengine: %v", err)
	}

	defer func() { _ = eng.Close() }()

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		t.Fatal("sqlite engine must implement MapBackend")
	}

	// 1. Data first — the planned table does not exist yet.
	for _, e := range []TaskCreated{
		{ID: "t1", Title: "one", Status: "open", Priority: 1},
		{ID: "t2", Title: "two", Status: "done", Priority: 2},
	} {
		if err := mb.MapSet(ctx, "find_task", e.ID, e); err != nil {
			t.Fatalf("pre-registration MapSet: %v", err)
		}
	}

	// 2. Registration — Plan applies the auto-layout (planned table starts empty).
	store, err := metaengine.Plan([]metaengine.Engine{eng}, filteredTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	defer store.Close()

	// 3. Backfill — the batch helper copies every meta_map row.
	results, err := store.BackfillPlannedTables(ctx, 0)
	if err != nil {
		t.Fatalf("BackfillPlannedTables: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("results = %d entries, want 1: %+v", len(results), results)
	}

	res := results[0]
	if res.Collection != "find_task" || res.Rows != 2 || res.Skipped || res.Err != nil {
		t.Fatalf("unexpected result: %+v", res)
	}

	// 4. Planned-path scan sees the backfilled rows (FilterOnField promotes
	// the query to a filtered scan; the planned table serves it).
	rows, qerr := metaengine.ExecuteTyped[FindTask, findTaskResults](
		ctx, store, FindTask{},
	)
	if qerr != nil {
		t.Fatalf("post-backfill scan: %v", qerr)
	}

	if len(rows.Items) != 2 {
		t.Fatalf("scan rows = %d, want 2: %+v", len(rows.Items), rows.Items)
	}

	byID := map[string]FindTaskResult{}

	for _, r := range rows.Items {
		byID[string(r.ID)] = r
	}

	got := byID["t1"]
	if got.Status != "open" || got.Title != "one" {
		t.Fatalf("backfilled row t1 wrong: %+v", got)
	}

	// 5. Idempotent re-run converges.
	results, err = store.BackfillPlannedTables(ctx, 1)
	if err != nil {
		t.Fatalf("re-backfill: %v", err)
	}

	if results[0].Rows != 2 {
		t.Fatalf("re-backfill rows = %d, want 2", results[0].Rows)
	}

	// 6. Doctor shows the backfill state.
	if !strings.Contains(store.Doctor(ctx), "backfill: rows=2 engine=sqlite") {
		t.Errorf("Doctor missing backfill state:\n%s", store.Doctor(ctx))
	}
}

// plannedFakeEngine declares layout planning but implements neither
// KeyScanBackend nor MapBackend — the no-capability branch.
type plannedFakeEngine struct {
	fakeEngine
}

func (e *plannedFakeEngine) ApplyLayout(string, []string, []string) error { return nil }

func (e *plannedFakeEngine) ApplyLayoutPlan(metaengine.LayoutPlan) error { return nil }

// TestBackfillPlannedTables_SkipsWithoutCapability pins the loud-skip rule:
// engines without the KeyScanBackend+MapBackend pair are reported as Skipped
// (never silent), and the batch still succeeds for the capable engines.
func TestBackfillPlannedTables_SkipsWithoutCapability(t *testing.T) {
	t.Parallel()

	eng := &plannedFakeEngine{fakeEngine: fakeEngine{profile: metaengine.EngineProfile{
		Name: "fake-planned",
		Layouts: map[metaengine.ADT]metaengine.StorageLayout{
			metaengine.ADTMap: metaengine.LayoutRow,
		},
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityOLogN,
		},
	}}}

	store, err := metaengine.Plan([]metaengine.Engine{eng}, filteredTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	defer store.Close()

	results, err := store.BackfillPlannedTables(t.Context(), 0)
	if err != nil {
		t.Fatalf("skip must not fail the batch: %v", err)
	}

	if len(results) != 1 || !results[0].Skipped {
		t.Fatalf("expected skipped result, got %+v", results)
	}

	if !errors.Is(results[0].Err, metaengine.ErrBackfillUnsupported) {
		t.Fatalf("skip reason = %v, want ErrBackfillUnsupported", results[0].Err)
	}
}
