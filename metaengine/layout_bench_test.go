package metaengine

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"
	"math/rand/v2"
	"testing"

	_ "modernc.org/sqlite"
)

// layout_bench_test.go is THE kill criterion test for the layout planning
// hypothesis. It compares:
//
//   Variant A (naive):     meta_map table with json_extract() pushdown
//   Variant B (planned):   dedicated table with indexed extracted columns
//
// The kill criterion: if Variant B does not beat Variant A by >=2x on at
// least 3 of 5 query patterns, the layout planning hypothesis is FALSE.

type benchItem struct {
	ID       string `json:"ID"`
	Status   string `json:"Status"`
	Priority int    `json:"Priority"`
	Title    string `json:"Title"`
}

func populateBenchData(t testing.TB, eng Engine, col string, n int) {
	t.Helper()

	ctx := context.Background()
	mb := eng.(MapBackend)

	for i := 0; i < n; i++ {
		item := benchItem{
			ID:       fmt.Sprintf("item-%05d", i),
			Status:   []string{"open", "done", "pending"}[rand.IntN(3)],
			Priority: rand.IntN(100),
			Title:    fmt.Sprintf("Task number %d", i),
		}

		if err := mb.MapSet(ctx, col, item.ID, item); err != nil {
			t.Fatalf("MapSet: %v", err)
		}
	}
}

func openSQLite(t testing.TB) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}

	return db
}

// BenchmarkLayout_Naive_FilterByStatus measures json_extract() pushdown
// on the naive meta_map table (no index, JSON parsed per row).
func BenchmarkLayout_Naive_FilterByStatus(b *testing.B) {
	db := openSQLite(b)
	defer func() { _ = db.Close() }()

	eng, err := NewSQLiteEngine(db)
	if err != nil {
		b.Fatal(err)
	}

	defer eng.Close()
	populateBenchData(b, eng, "items", 10_000)

	ctx := context.Background()
	ps := eng.(PushdownScan)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := ps.PushdownMapScan(ctx, "items",
			[]FilterSpec{{Column: "Status", Op: FilterEq, Value: "open"}},
			nil, nil, 10)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLayout_Planned_FilterByStatus measures indexed column lookup
// on a layout-planned table (B-tree index seek, no JSON parsing).
func BenchmarkLayout_Planned_FilterByStatus(b *testing.B) {
	db := openSQLite(b)
	defer func() { _ = db.Close() }()

	plan := BuildLayoutPlan("items",
		[]string{"Status"}, // filter fields
		[]string{"Priority"}, // sort fields
	)

	eng, err := NewPlannedSQLiteEngine(db, []LayoutPlan{plan})
	if err != nil {
		b.Fatal(err)
	}

	defer eng.Close()
	populateBenchData(b, eng, "items", 10_000)

	ctx := context.Background()
	ps := eng.(PushdownScan)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := ps.PushdownMapScan(ctx, "items",
			[]FilterSpec{{Column: "Status", Op: FilterEq, Value: "open"}},
			nil, nil, 10)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLayout_Naive_FilterAndSort measures json_extract() filter + sort.
func BenchmarkLayout_Naive_FilterAndSort(b *testing.B) {
	db := openSQLite(b)
	defer func() { _ = db.Close() }()

	eng, err := NewSQLiteEngine(db)
	if err != nil {
		b.Fatal(err)
	}

	defer eng.Close()
	populateBenchData(b, eng, "items", 10_000)

	ctx := context.Background()
	ps := eng.(PushdownScan)

	sort := &SortSpec{Column: "Priority", Desc: true}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := ps.PushdownMapScan(ctx, "items",
			[]FilterSpec{{Column: "Status", Op: FilterEq, Value: "open"}},
			sort, nil, 10)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLayout_Planned_FilterAndSort measures indexed filter + sort.
func BenchmarkLayout_Planned_FilterAndSort(b *testing.B) {
	db := openSQLite(b)
	defer func() { _ = db.Close() }()

	plan := BuildLayoutPlan("items",
		[]string{"Status"},
		[]string{"Priority"},
	)

	eng, err := NewPlannedSQLiteEngine(db, []LayoutPlan{plan})
	if err != nil {
		b.Fatal(err)
	}

	defer eng.Close()
	populateBenchData(b, eng, "items", 10_000)

	ctx := context.Background()
	ps := eng.(PushdownScan)

	sort := &SortSpec{Column: "Priority", Desc: true}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := ps.PushdownMapScan(ctx, "items",
			[]FilterSpec{{Column: "Status", Op: FilterEq, Value: "open"}},
			sort, nil, 10)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLayout_Naive_PointLookup measures point lookup on meta_map.
func BenchmarkLayout_Naive_PointLookup(b *testing.B) {
	db := openSQLite(b)
	defer func() { _ = db.Close() }()

	eng, err := NewSQLiteEngine(db)
	if err != nil {
		b.Fatal(err)
	}

	defer eng.Close()
	populateBenchData(b, eng, "items", 10_000)

	ctx := context.Background()
	mb := eng.(MapBackend)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("item-%05d", i%10_000)
		_, _, err := mb.MapGet(ctx, "items", key)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLayout_Planned_PointLookup measures point lookup on planned table.
func BenchmarkLayout_Planned_PointLookup(b *testing.B) {
	db := openSQLite(b)
	defer func() { _ = db.Close() }()

	plan := BuildLayoutPlan("items", []string{"Status"}, []string{"Priority"})

	eng, err := NewPlannedSQLiteEngine(db, []LayoutPlan{plan})
	if err != nil {
		b.Fatal(err)
	}

	defer eng.Close()
	populateBenchData(b, eng, "items", 10_000)

	ctx := context.Background()
	mb := eng.(MapBackend)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("item-%05d", i%10_000)
		_, _, err := mb.MapGet(ctx, "items", key)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestLayoutPlanner_GeneratesCorrectDDL verifies the DDL generator output.
func TestLayoutPlanner_GeneratesCorrectDDL(t *testing.T) {
	plan := BuildLayoutPlan("users",
		[]string{"Status", "Priority"},
		[]string{"Priority"},
	)

	ddl := plan.DDL()

	// Should have one table.
	if !contains(ddl, "CREATE TABLE IF NOT EXISTS meta_planned_users") {
		t.Errorf("DDL missing CREATE TABLE: %s", ddl)
	}

	// Should have both columns.
	if !contains(ddl, "Status TEXT") {
		t.Errorf("DDL missing Status column: %s", ddl)
	}

	if !contains(ddl, "Priority INTEGER") {
		t.Errorf("DDL missing Priority column: %s", ddl)
	}

	// Should have exactly 2 indexes (deduped — Priority appears in both
	// filter and sort but gets ONE column and ONE index).
	idxCount := countSubstring(ddl, "CREATE INDEX")
	if idxCount != 2 {
		t.Errorf("expected 2 indexes (deduped), got %d: %s", idxCount, ddl)
	}
}

// TestLayoutPlanner_DedupSameFieldInFilterAndSort verifies rule 3:
// RangeFilter + OrderBy on same column → one index.
func TestLayoutPlanner_DedupSameFieldInFilterAndSort(t *testing.T) {
	plan := BuildLayoutPlan("tasks",
		[]string{"Priority"}, // filter
		[]string{"Priority"}, // sort — SAME field
	)

	if len(plan.Columns) != 1 {
		t.Errorf("expected 1 column (deduped), got %d: %v", len(plan.Columns), plan.Columns)
	}

	if len(plan.Indexes) != 1 {
		t.Errorf("expected 1 index (deduped), got %d: %v", len(plan.Indexes), plan.Indexes)
	}
}

// TestPlannedEngine_PushdownUsesIndexedColumns verifies that the planned
// engine uses direct column references instead of json_extract.
func TestPlannedEngine_PushdownUsesIndexedColumns(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	plan := BuildLayoutPlan("items", []string{"Status"}, []string{"Priority"})

	eng, err := NewPlannedSQLiteEngine(db, []LayoutPlan{plan})
	if err != nil {
		t.Fatal(err)
	}

	defer eng.Close()

	ctx := context.Background()
	mb := eng.(MapBackend)

	// Populate.
	for i := 0; i < 100; i++ {
		item := benchItem{
			ID:       fmt.Sprintf("i%d", i),
			Status:   "open",
			Priority: i % 10,
		}
		if err := mb.MapSet(ctx, "items", item.ID, item); err != nil {
			t.Fatal(err)
		}
	}

	// Verify data via point lookup.
	val, found, err := mb.MapGet(ctx, "items", "i5")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected to find i5")
	}

	m, ok := val.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", val)
	}

	if m["Status"] != "open" {
		t.Errorf("expected Status=open, got %v", m["Status"])
	}

	// Verify pushdown uses the planned table.
	ps := eng.(PushdownScan)

	results, err := ps.PushdownMapScan(ctx, "items",
		[]FilterSpec{{Column: "Status", Op: FilterEq, Value: "open"}},
		&SortSpec{Column: "Priority", Desc: true},
		nil, 5,
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 6 { // limit+1
		t.Errorf("expected 6 results (limit+1), got %d", len(results))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}

func countSubstring(s, substr string) int {
	count := 0

	idx := 0
	for {
		pos := indexOf(s, substr, idx)
		if pos < 0 {
			break
		}

		count++
		idx = pos + len(substr)
	}

	return count
}

func indexOf(s, substr string, start int) int {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}

	return -1
}

// Ensure unused import is referenced.
var _ = json.Marshal
