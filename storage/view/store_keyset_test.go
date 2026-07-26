package view

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

// TestSQLViewStore_Query_KeysetDesc verifies keyset (seek) pagination in
// descending order: page 2 starts strictly after the last row of page 1,
// determined by a composite cursor on (age, key).
func TestSQLViewStore_Query_KeysetDesc(t *testing.T) {
	store := parallelViewStore(t)
	ctx := context.Background()

	for i, age := range []int{10, 20, 30, 40, 50} {
		key := fmt.Sprintf("u%d", i)
		if err := store.Set(ctx, testKey(key), &testView{
			Name: key, Email: "x@ex.com", Age: age,
		}); err != nil {
			t.Fatalf("Set %s: %v", key, err)
		}
	}

	// Page 1: two oldest.
	page1, err := store.Query(ctx, kv.ViewQuery{OrderBy: "age", Desc: true, Limit: 2})
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if len(page1) != 2 || page1[0].Age != 50 || page1[1].Age != 40 {
		t.Fatalf("page1: got ages %d,%d; want 50,40", page1[0].Age, page1[1].Age)
	}

	// Page 2: cursor at the last row of page 1 (age=40, key="u3").
	page2, err := store.Query(ctx, kv.ViewQuery{
		OrderBy: "age",
		Desc:    true,
		Limit:   2,
		Keyset:  &kv.Keyset{Columns: []string{"age", "key"}, Values: []any{40, "u3"}},
	})
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	if len(page2) != 2 || page2[0].Age != 30 || page2[1].Age != 20 {
		t.Fatalf("page2: got ages %d,%d; want 30,20", page2[0].Age, page2[1].Age)
	}

	// Page 3: cursor at the last row of page 2 (age=20, key="u1").
	page3, err := store.Query(ctx, kv.ViewQuery{
		OrderBy: "age",
		Desc:    true,
		Limit:   2,
		Keyset:  &kv.Keyset{Columns: []string{"age", "key"}, Values: []any{20, "u1"}},
	})
	if err != nil {
		t.Fatalf("page3: %v", err)
	}
	if len(page3) != 1 || page3[0].Age != 10 {
		t.Fatalf(
			"page3: got %d row(s), age %d; want 1 row, age 10",
			len(page3),
			page3SafeAge(page3),
		)
	}
}

// TestSQLViewStore_Query_KeysetAsc verifies keyset pagination ascending.
func TestSQLViewStore_Query_KeysetAsc(t *testing.T) {
	store := parallelViewStore(t)
	ctx := context.Background()

	for i, age := range []int{10, 20, 30, 40, 50} {
		key := fmt.Sprintf("u%d", i)
		if err := store.Set(ctx, testKey(key), &testView{
			Name: key, Email: "x@ex.com", Age: age,
		}); err != nil {
			t.Fatalf("Set %s: %v", key, err)
		}
	}

	// Page 1: two youngest (ascending).
	page1, err := store.Query(ctx, kv.ViewQuery{OrderBy: "age", Limit: 2})
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if len(page1) != 2 || page1[0].Age != 10 || page1[1].Age != 20 {
		t.Fatalf("page1: got ages %d,%d; want 10,20", page1[0].Age, page1[1].Age)
	}

	// Page 2: cursor after (age=20, key="u1") ascending → rows greater.
	page2, err := store.Query(ctx, kv.ViewQuery{
		OrderBy: "age",
		Limit:   2,
		Keyset:  &kv.Keyset{Columns: []string{"age", "key"}, Values: []any{20, "u1"}},
	})
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	if len(page2) != 2 || page2[0].Age != 30 || page2[1].Age != 40 {
		t.Fatalf("page2: got ages %d,%d; want 30,40", page2[0].Age, page2[1].Age)
	}
}

// TestSQLViewStore_Query_KeysetTiebreaker verifies the key tiebreaker: rows
// with equal sort-column values are paginated deterministically (no skip, no
// repeat) when the cursor includes the unique key column.
func TestSQLViewStore_Query_KeysetTiebreaker(t *testing.T) {
	store := parallelViewStore(t)
	ctx := context.Background()

	// Five rows, all age=25, distinct keys u1..u5.
	for _, key := range []string{"u1", "u2", "u3", "u4", "u5"} {
		if err := store.Set(ctx, testKey(key), &testView{
			Name: key, Email: "x@ex.com", Age: 25,
		}); err != nil {
			t.Fatalf("Set %s: %v", key, err)
		}
	}

	// Page 1 ascending by key.
	page1, err := store.Query(ctx, kv.ViewQuery{OrderBy: "key", Limit: 2})
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if len(page1) != 2 || page1[0].Name != "u1" || page1[1].Name != "u2" {
		t.Fatalf("page1: got %s,%s; want u1,u2", page1[0].Name, page1[1].Name)
	}

	// Page 2: cursor after ("u2").
	page2, err := store.Query(ctx, kv.ViewQuery{
		OrderBy: "key",
		Limit:   2,
		Keyset:  &kv.Keyset{Columns: []string{"key"}, Values: []any{"u2"}},
	})
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	if len(page2) != 2 || page2[0].Name != "u3" || page2[1].Name != "u4" {
		t.Fatalf("page2: got %s,%s; want u3,u4", page2[0].Name, page2[1].Name)
	}

	// Page 3: cursor after ("u4") → last row only.
	page3, err := store.Query(ctx, kv.ViewQuery{
		OrderBy: "key",
		Limit:   2,
		Keyset:  &kv.Keyset{Columns: []string{"key"}, Values: []any{"u4"}},
	})
	if err != nil {
		t.Fatalf("page3: %v", err)
	}
	if len(page3) != 1 || page3[0].Name != "u5" {
		t.Fatalf("page3: got %d rows, first %q; want 1 row, u5", len(page3), page3SafeName(page3))
	}
}

// TestSQLViewStore_Query_KeysetDefaultColumns verifies that an empty
// Keyset.Columns defaults to the ORDER BY column plus the key tiebreaker.
func TestSQLViewStore_Query_KeysetDefaultColumns(t *testing.T) {
	store := parallelViewStore(t)
	ctx := context.Background()

	for i, age := range []int{10, 20, 30} {
		key := fmt.Sprintf("u%d", i)
		if err := store.Set(ctx, testKey(key), &testView{
			Name: key, Email: "x@ex.com", Age: age,
		}); err != nil {
			t.Fatalf("Set %s: %v", key, err)
		}
	}

	// No explicit Columns — defaults to [age, key]. Cursor after (20, "u1") desc.
	page2, err := store.Query(ctx, kv.ViewQuery{
		OrderBy: "age",
		Desc:    true,
		Limit:   5,
		Keyset:  &kv.Keyset{Values: []any{20, "u1"}}, // Columns empty
	})
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	// Descending after (20,"u1") → only age 10 remains.
	if len(page2) != 1 || page2[0].Age != 10 {
		t.Fatalf("default cols: got %d rows, age %d; want 1 row, age 10",
			len(page2), page3SafeAge(page2))
	}
}

// TestSQLViewStore_Query_KeysetWithConditions verifies the cursor predicate is
// AND-joined with structured Conditions and that placeholder numbering stays
// correct across both arg sources.
func TestSQLViewStore_Query_KeysetWithConditions(t *testing.T) {
	store := parallelViewStore(t)
	ctx := context.Background()

	for i, age := range []int{10, 20, 30, 40, 50} {
		key := fmt.Sprintf("u%d", i)
		if err := store.Set(ctx, testKey(key), &testView{
			Name: key, Email: "x@ex.com", Age: age,
		}); err != nil {
			t.Fatalf("Set %s: %v", key, err)
		}
	}

	// Filter age > 15, then keyset after (30, "u2") descending.
	results, err := store.Query(ctx, kv.ViewQuery{
		Conditions: []kv.Condition{{Column: "age", Op: kv.OpGt, Value: 15}},
		OrderBy:    "age",
		Desc:       true,
		Limit:      5,
		Keyset:     &kv.Keyset{Columns: []string{"age", "key"}, Values: []any{30, "u2"}},
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	// age>15 AND (age,key)<(30,"u2") desc → ages 20 (u1). Only u1 qualifies.
	if len(results) != 1 || results[0].Age != 20 {
		t.Fatalf("keyset+conditions: got %d rows, age %d; want 1 row, age 20",
			len(results), page3SafeAge(results))
	}
}

// TestSQLViewStore_Query_MultiColumnOrder verifies Order with per-column
// direction (mixed ASC/DESC), which a single OrderBy+Desc cannot express.
func TestSQLViewStore_Query_MultiColumnOrder(t *testing.T) {
	store := parallelViewStore(t)
	ctx := context.Background()

	// Rows where age ties; the key direction decides the order within ties.
	rows := []struct {
		key string
		age int
	}{
		{"u1", 25},
		{"u2", 25},
		{"u3", 25},
		{"u4", 30},
	}
	for _, r := range rows {
		if err := store.Set(ctx, testKey(r.key), &testView{
			Name: r.key, Email: "x@ex.com", Age: r.age,
		}); err != nil {
			t.Fatalf("Set %s: %v", r.key, err)
		}
	}

	// age ASC, key DESC: u4(30) last; within age=25 → u3,u2,u1.
	asc, err := store.Query(ctx, kv.ViewQuery{
		Order: []kv.OrderClause{{Column: "age"}, {Column: "key", Desc: true}},
	})
	if err != nil {
		t.Fatalf("Query asc/desc: %v", err)
	}
	want := []string{"u3", "u2", "u1", "u4"}
	for i, w := range want {
		if asc[i].Name != w {
			t.Fatalf("age ASC key DESC[%d]: got %s, want %s", i, asc[i].Name, w)
		}
	}

	// age DESC, key ASC: u4(30) first; within age=25 → u1,u2,u3.
	desc, err := store.Query(ctx, kv.ViewQuery{
		Order: []kv.OrderClause{{Column: "age", Desc: true}, {Column: "key"}},
	})
	if err != nil {
		t.Fatalf("Query desc/asc: %v", err)
	}
	wantDesc := []string{"u4", "u1", "u2", "u3"}
	for i, w := range wantDesc {
		if desc[i].Name != w {
			t.Fatalf("age DESC key ASC[%d]: got %s, want %s", i, desc[i].Name, w)
		}
	}
}

// TestSQLViewStore_PartialIndex verifies that an IndexSpec with a Where
// predicate creates a partial index (CREATE INDEX ... WHERE).
func TestSQLViewStore_PartialIndex(t *testing.T) {
	t.Parallel()

	db, err := openSQLiteInMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mapper := testMapper()
	mapper.Indexes = []IndexSpec{
		{Name: "idx_active_age", Columns: []string{"age"}, Where: "tombstoned = 0"},
		{Name: "idx_email", Columns: []string{"email"}},
	}

	if _, err := NewSQLiteViewStore[testView, testKey](db, mapper); err != nil {
		t.Fatalf("NewSQLiteViewStore: %v", err)
	}

	ctx := context.Background()

	var sqlPartial, sqlFull string
	if err := db.QueryRowContext(
		ctx,
		"SELECT sql FROM sqlite_master WHERE type='index' AND name='idx_active_age'",
	).Scan(&sqlPartial); err != nil {
		t.Fatalf("query partial index: %v", err)
	}
	if err := db.QueryRowContext(
		ctx,
		"SELECT sql FROM sqlite_master WHERE type='index' AND name='idx_email'",
	).Scan(&sqlFull); err != nil {
		t.Fatalf("query full index: %v", err)
	}

	if !strings.Contains(sqlPartial, "WHERE tombstoned = 0") {
		t.Fatalf("partial index missing WHERE: %s", sqlPartial)
	}
	if strings.Contains(sqlFull, "WHERE") {
		t.Fatalf("full index should have no WHERE: %s", sqlFull)
	}
}

// page3SafeAge / page3SafeName keep t.Fatalf arg evaluation safe for empty
// slices (avoid index-out-of-range in the failure message itself).
func page3SafeAge(views []*testView) int {
	if len(views) == 0 {
		return -1
	}
	return views[0].Age
}

func page3SafeName(views []*testView) string {
	return safeName(views)
}
