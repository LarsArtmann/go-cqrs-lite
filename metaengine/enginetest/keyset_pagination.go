package enginetest

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

type keysetRow struct {
	ID       string
	Status   string
	Priority int
}

type keysetListInput struct {
	Status string
}

// KeysetPaginationQuery returns the query declaration backing
// RunKeysetPaginationTest so engine suites can reuse the same projection
// shape (keyed by ID, sorted by Priority).
func KeysetPaginationQuery(name string) metaengine.QueryDecl[keysetListInput, keysetRow] {
	return metaengine.Query[keysetListInput, keysetRow](
		name,
		metaengine.OnRecord(
			keysetRow{},
			func(_ record.Record, e keysetRow) (string, keysetRow) { return e.ID, e },
		),
		metaengine.FilterOnField[keysetRow]("Status", metaengine.FilterEq),
		metaengine.SortOnField[keysetRow]("Priority", true),
	)
}

// RunKeysetPaginationTest pins the compound-cursor pagination contract at the
// TypedReader level for ANY engine: tie-heavy datasets (Priority = i%3) must
// walk exactly once through ScanPage with the engine-issued SortKeyCursor,
// a full final page must end with a nil cursor (no phantom empty page), and
// DESC walks must page with the same exactness.
func RunKeysetPaginationTest(t *testing.T, eng metaengine.Engine) {
	t.Helper()

	queryName := ScopedCollection("keyset_pagination")

	store, err := metaengine.Plan([]metaengine.Engine{eng}, KeysetPaginationQuery(queryName))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	t.Cleanup(func() { _ = store.Close() })

	seedKeysetRows(t, store, "keysetRow", 13)

	reader := metaengine.NewReader[keysetRow](store, queryName)

	wantAsc := []string{
		"item-000", "item-003", "item-006", "item-009", "item-012",
		"item-001", "item-004", "item-007", "item-010", "item-002",
		"item-005", "item-008", "item-011",
	}

	wantDesc := []string{
		"item-002", "item-005", "item-008", "item-011", "item-001",
		"item-004", "item-007", "item-010", "item-000", "item-003",
		"item-006", "item-009", "item-012",
	}

	t.Run("tie heavy exact once asc", func(t *testing.T) {
		walk := keysetWalk(t, reader, false, 5)

		if !slices.Equal(walk.visited, wantAsc) {
			t.Fatalf("walk visited %v\nwant           %v", walk.visited, wantAsc)
		}

		if !slices.Equal(walk.pages, []int{5, 5, 3}) {
			t.Fatalf("page sizes %v, want [5 5 3]", walk.pages)
		}

		if walk.firstCursor == nil {
			t.Fatal("page 1 cursor is nil; sorted scans must issue a compound cursor")
		}

		skc, ok := walk.firstCursor.(metaengine.SortKeyCursor)
		if !ok {
			t.Fatalf("page 1 cursor %T, want SortKeyCursor", walk.firstCursor)
		}

		if fmt.Sprintf("%v", skc.Sort) != "0" || string(skc.Key) != "item-012" {
			t.Fatalf("page 1 cursor %+v, want Sort=0 Key=item-012", skc)
		}
	})

	t.Run("desc exact once", func(t *testing.T) {
		walk := keysetWalk(t, reader, true, 5)

		if !slices.Equal(walk.visited, wantDesc) {
			t.Fatalf("walk visited %v\nwant           %v", walk.visited, wantDesc)
		}
	})

	t.Run("engine pushdown surface", func(t *testing.T) {
		pd, ok := eng.(metaengine.PushdownScan)
		if !ok {
			t.Skipf("engine %T does not implement PushdownScan", eng)
		}

		asc := keysetEngineWalk(t, pd, queryName, false, 5)

		if !slices.Equal(asc.visited, wantAsc) {
			t.Fatalf("pushdown asc visited %v\nwant                %v", asc.visited, wantAsc)
		}

		if !slices.Equal(asc.pages, []int{5, 5, 3}) {
			t.Fatalf("pushdown asc page sizes %v, want [5 5 3]", asc.pages)
		}

		skc, ok := asc.firstCursor.(metaengine.SortKeyCursor)
		if !ok {
			t.Fatalf("pushdown cursor %T, want SortKeyCursor", asc.firstCursor)
		}

		if fmt.Sprintf("%v", skc.Sort) != "0" || string(skc.Key) != "item-012" {
			t.Fatalf("pushdown cursor %+v, want Sort=0 Key=item-012", skc)
		}

		desc := keysetEngineWalk(t, pd, queryName, true, 5)

		if !slices.Equal(desc.visited, wantDesc) {
			t.Fatalf("pushdown desc visited %v\nwant                 %v", desc.visited, wantDesc)
		}
	})
}

// RunKeysetExactEndTest pins the phantom-page kill on a dataset whose length
// is an exact multiple of the limit: the final page is FULL yet must end the
// walk with a nil cursor (the engine's has-more probe reports exhaustion).
func RunKeysetExactEndTest(t *testing.T, eng metaengine.Engine) {
	t.Helper()

	queryName := ScopedCollection("keyset_exact_end")

	store, err := metaengine.Plan([]metaengine.Engine{eng}, KeysetPaginationQuery(queryName))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	t.Cleanup(func() { _ = store.Close() })

	seedKeysetRows(t, store, "keysetRow", 12)

	reader := metaengine.NewReader[keysetRow](store, queryName)

	walk := keysetWalk(t, reader, false, 4)

	want := []string{
		"item-000", "item-003", "item-006", "item-009",
		"item-001", "item-004", "item-007", "item-010",
		"item-002", "item-005", "item-008", "item-011",
	}

	if !slices.Equal(walk.visited, want) {
		t.Fatalf("walk visited %v\nwant           %v", walk.visited, want)
	}

	if !slices.Equal(walk.pages, []int{4, 4, 4}) {
		t.Fatalf("page sizes %v, want [4 4 4] with no trailing empty page", walk.pages)
	}

	if pd, ok := eng.(metaengine.PushdownScan); ok {
		engineWalk := keysetEngineWalk(t, pd, queryName, false, 4)

		if !slices.Equal(engineWalk.visited, want) {
			t.Fatalf("pushdown visited %v\nwant             %v", engineWalk.visited, want)
		}

		if !slices.Equal(engineWalk.pages, []int{4, 4, 4}) {
			t.Fatalf("pushdown page sizes %v, want [4 4 4]", engineWalk.pages)
		}
	}
}

// keysetEngineWalk drives PushdownMapScan directly to exhaustion, passing
// ScanResult.NextCursor back raw. Bypassing the TypedReader pins the engine's
// OWN SQL keyset emission and consumption even when the reader would route
// the query through another path (closure or raw). The walk terminates on
// HasMore=false, so an engine that mints a cursor past the exact end still
// passes here as long as its has-more probe is honest — the reader-level walk
// pins the nil-cursor-at-exact-end half of the contract.
func keysetEngineWalk(
	t *testing.T,
	pd metaengine.PushdownScan,
	collection string,
	desc bool,
	limit int,
) keysetWalkResult {
	t.Helper()

	ctx := context.Background()

	var walk keysetWalkResult

	sortSpec := metaengine.SortSpec{Column: "Priority", Desc: desc}

	var cursorVal any

	for range 50 {
		res, err := pd.PushdownMapScan(ctx, collection, nil, &sortSpec, cursorVal, limit)
		if err != nil {
			t.Fatalf("PushdownMapScan: %v", err)
		}

		for _, item := range res.Items {
			walk.visited = append(
				walk.visited,
				fmt.Sprintf("%v", metaengine.ItemFieldByName(item, "ID")),
			)
		}

		walk.pages = append(walk.pages, len(res.Items))

		if walk.firstCursor == nil {
			walk.firstCursor = res.NextCursor
		}

		if !res.HasMore {
			return walk
		}

		if res.NextCursor == nil {
			t.Fatal("HasMore=true without NextCursor: keyset paging cannot continue")
		}

		cursorVal = res.NextCursor
	}

	t.Fatalf(
		"pushdown pagination did not terminate within 50 pages (cursor loop?): pages %v, visited %v",
		walk.pages,
		walk.visited,
	)

	return walk
}

func seedKeysetRows(t *testing.T, store *metaengine.Store, collection string, n int) {
	t.Helper()

	ctx := context.Background()

	for i := range n {
		row := keysetRow{
			ID:       fmt.Sprintf("item-%03d", i),
			Status:   "open",
			Priority: i % 3,
		}

		if err := store.Apply(ctx, collection, row); err != nil {
			t.Fatalf("seed Apply[%d]: %v", i, err)
		}
	}
}

type keysetWalkResult struct {
	visited     []string
	pages       []int
	firstCursor any
}

// keysetWalk drives ScanPage to exhaustion over the seeded collection,
// passing each cursor back raw. The 50-page bound turns a cursor loop into a
// test failure instead of a hang.
func keysetWalk(
	t *testing.T,
	reader *metaengine.TypedReader[keysetRow],
	desc bool,
	limit int,
) keysetWalkResult {
	t.Helper()

	ctx := context.Background()

	var walk keysetWalkResult

	var cursorVal any

	for range 50 {
		opts := []metaengine.ScanOption{
			metaengine.WithSort("Priority", desc),
			metaengine.WithLimit(limit),
		}

		if cursorVal != nil {
			opts = append(opts, metaengine.WithCursor(cursorVal))
		}

		rows, cursor, err := reader.ScanPage(ctx, opts...)
		if err != nil {
			t.Fatalf("ScanPage: %v", err)
		}

		for _, row := range rows {
			walk.visited = append(walk.visited, row.ID)
		}

		walk.pages = append(walk.pages, len(rows))

		if cursor == nil {
			return walk
		}

		if walk.firstCursor == nil {
			walk.firstCursor = cursor.Value
		}

		cursorVal = cursor.Value
	}

	t.Fatalf("pagination did not terminate within 50 pages (cursor loop?): pages %v", walk.pages)

	return walk
}
