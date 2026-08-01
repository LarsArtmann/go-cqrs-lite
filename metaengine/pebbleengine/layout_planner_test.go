package pebbleengine_test

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"testing"

	"github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestPebbleLayoutPlanner_SecondaryIndex(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := pebbleengine.NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	defer eng.Close()

	lp, ok := eng.(metaengine.LayoutPlanner)
	if !ok {
		t.Fatal("expected pebbleEngine to implement LayoutPlanner")
	}

	// Declare a layout plan for "users" with "status" as a filter field.
	if err := lp.ApplyLayout("users", []string{"status"}, nil); err != nil {
		t.Fatalf("ApplyLayout: %v", err)
	}

	mb := eng.(metaengine.MapBackend)

	// Write 5 users: 3 active, 2 inactive.
	users := []struct {
		key    string
		status string
		name   string
	}{
		{"u1", "active", "Alice"},
		{"u2", "active", "Bob"},
		{"u3", "inactive", "Charlie"},
		{"u4", "active", "Diana"},
		{"u5", "inactive", "Eve"},
	}

	for _, u := range users {
		if err := mb.MapSet(ctx, "users", u.key, map[string]any{
			"status": u.status,
			"name":   u.name,
		}); err != nil {
			t.Fatalf("MapSet %s: %v", u.key, err)
		}
	}

	// Scan with filter on "status" = "active" — should use the index.
	rawReader, ok := eng.(metaengine.RawScanReader)
	if !ok {
		t.Fatal("expected pebbleEngine to implement RawScanReader")
	}

	results, err := rawReader.ScanRawValues(
		ctx, "users",
		[]metaengine.FilterSpec{{Column: "status", Op: metaengine.FilterEq, Value: "active"}},
		nil, nil, 0,
	)
	if err != nil {
		t.Fatalf("ScanRawValues: %v", err)
	}

	if len(results.Items) != 3 {
		t.Fatalf("expected 3 active users, got %d", len(results.Items))
	}

	// Verify all results have status = "active".
	for _, raw := range results.Items {
		var decoded map[string]any
		_ = json.Unmarshal(raw, &decoded)
		if decoded["status"] != "active" {
			t.Errorf("expected status 'active', got %v", decoded["status"])
		}
	}
}

func TestPebbleLayoutPlanner_UpdateReindexes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := pebbleengine.NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	defer eng.Close()

	lp := eng.(metaengine.LayoutPlanner)
	if err := lp.ApplyLayout("items", []string{"cat"}, nil); err != nil {
		t.Fatalf("ApplyLayout: %v", err)
	}

	mb := eng.(metaengine.MapBackend)

	// Write with category "a".
	_ = mb.MapSet(ctx, "items", "k1", map[string]any{"cat": "a", "val": 1})

	// Update to category "b".
	_ = mb.MapSet(ctx, "items", "k1", map[string]any{"cat": "b", "val": 2})

	// Scan for cat="a" — should return 0 (old index removed).
	rawReader := eng.(metaengine.RawScanReader)

	resultsA, err := rawReader.ScanRawValues(
		ctx, "items",
		[]metaengine.FilterSpec{{Column: "cat", Op: metaengine.FilterEq, Value: "a"}},
		nil, nil, 0,
	)
	if err != nil {
		t.Fatalf("ScanRawValues cat=a: %v", err)
	}

	if len(resultsA.Items) != 0 {
		t.Errorf("expected 0 items with cat=a after update, got %d", len(resultsA.Items))
	}

	// Scan for cat="b" — should return 1.
	resultsB, err := rawReader.ScanRawValues(
		ctx, "items",
		[]metaengine.FilterSpec{{Column: "cat", Op: metaengine.FilterEq, Value: "b"}},
		nil, nil, 0,
	)
	if err != nil {
		t.Fatalf("ScanRawValues cat=b: %v", err)
	}

	if len(resultsB.Items) != 1 {
		t.Errorf("expected 1 item with cat=b after update, got %d", len(resultsB.Items))
	}
}

func TestPebbleLayoutPlanner_DeleteRemovesIndex(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := pebbleengine.NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	defer eng.Close()

	lp := eng.(metaengine.LayoutPlanner)
	if err := lp.ApplyLayout("users", []string{"status"}, nil); err != nil {
		t.Fatalf("ApplyLayout: %v", err)
	}

	mb := eng.(metaengine.MapBackend)

	// Write 3 active users.
	for _, key := range []string{"u1", "u2", "u3"} {
		if err := mb.MapSet(
			ctx,
			"users",
			key,
			map[string]any{"status": "active", "name": key},
		); err != nil {
			t.Fatalf("MapSet %s: %v", key, err)
		}
	}

	// Delete u2 — its index entry must be cleaned up.
	if err := mb.MapDelete(ctx, "users", "u2"); err != nil {
		t.Fatalf("MapDelete: %v", err)
	}

	// Scan for status="active" — should return 2 (u1, u3), not 3.
	rawReader := eng.(metaengine.RawScanReader)
	results, err := rawReader.ScanRawValues(
		ctx, "users",
		[]metaengine.FilterSpec{{Column: "status", Op: metaengine.FilterEq, Value: "active"}},
		nil, nil, 0,
	)
	if err != nil {
		t.Fatalf("ScanRawValues: %v", err)
	}

	if len(results.Items) != 2 {
		t.Fatalf(
			"expected 2 active users after delete, got %d (orphaned index entry?)",
			len(results.Items),
		)
	}

	// Verify deleted user is not in results.
	for _, raw := range results.Items {
		var decoded map[string]any
		_ = json.Unmarshal(raw, &decoded)
		if decoded["name"] == "u2" {
			t.Error("deleted user u2 appeared in scan results — index entry not cleaned up")
		}
	}
}

func TestPebbleLayoutPlanner_MapUpdateReindexes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := pebbleengine.NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	defer eng.Close()

	lp := eng.(metaengine.LayoutPlanner)
	if err := lp.ApplyLayout("items", []string{"cat"}, nil); err != nil {
		t.Fatalf("ApplyLayout: %v", err)
	}

	mb := eng.(metaengine.MapBackend)

	// Write with cat="a".
	if err := mb.MapSet(ctx, "items", "k1", map[string]any{"cat": "a", "val": 1}); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	// Use MapUpdate to change cat from "a" to "b".
	mu := eng.(metaengine.MapUpdater)
	if err := mu.MapUpdate(ctx, "items", "k1", func(prev any) any {
		m := prev.(map[string]any)
		m["cat"] = "b"
		m["val"] = 2

		return m
	}); err != nil {
		t.Fatalf("MapUpdate: %v", err)
	}

	// Scan for cat="a" — should return 0 (old index entry removed by MapUpdate).
	rawReader := eng.(metaengine.RawScanReader)

	resultsA, err := rawReader.ScanRawValues(
		ctx, "items",
		[]metaengine.FilterSpec{{Column: "cat", Op: metaengine.FilterEq, Value: "a"}},
		nil, nil, 0,
	)
	if err != nil {
		t.Fatalf("ScanRawValues cat=a: %v", err)
	}

	if len(resultsA.Items) != 0 {
		t.Errorf(
			"expected 0 items with cat=a after MapUpdate, got %d (orphaned index?)",
			len(resultsA.Items),
		)
	}

	// Scan for cat="b" — should return 1.
	resultsB, err := rawReader.ScanRawValues(
		ctx, "items",
		[]metaengine.FilterSpec{{Column: "cat", Op: metaengine.FilterEq, Value: "b"}},
		nil, nil, 0,
	)
	if err != nil {
		t.Fatalf("ScanRawValues cat=b: %v", err)
	}

	if len(resultsB.Items) != 1 {
		t.Errorf("expected 1 item with cat=b after MapUpdate, got %d", len(resultsB.Items))
	}
}

func TestPebbleLayoutPlanner_RangeFilters(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := pebbleengine.NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	defer eng.Close()

	lp := eng.(metaengine.LayoutPlanner)
	if err := lp.ApplyLayout("items", []string{"score"}, nil); err != nil {
		t.Fatalf("ApplyLayout: %v", err)
	}

	mb := eng.(metaengine.MapBackend)

	// Write items with scores: 10, 20, 30, 40, 50.
	scores := []struct {
		key   string
		score int
	}{
		{"a", 10}, {"b", 20}, {"c", 30}, {"d", 40}, {"e", 50},
	}

	for _, item := range scores {
		if err := mb.MapSet(ctx, "items", item.key, map[string]any{
			"score": item.score,
			"name":  item.key,
		}); err != nil {
			t.Fatalf("MapSet %s: %v", item.key, err)
		}
	}

	rawReader := eng.(metaengine.RawScanReader)

	// Test FilterGt: score > 20 → should return 3 (30, 40, 50).
	results, err := rawReader.ScanRawValues(
		ctx, "items",
		[]metaengine.FilterSpec{{Column: "score", Op: metaengine.FilterGt, Value: 20}},
		nil, nil, 0,
	)
	if err != nil {
		t.Fatalf("ScanRawValues FilterGt: %v", err)
	}
	if len(results.Items) != 3 {
		t.Errorf("FilterGt(20): expected 3 results, got %d", len(results.Items))
	}

	// Test FilterGe: score >= 30 → should return 3 (30, 40, 50).
	results, err = rawReader.ScanRawValues(
		ctx, "items",
		[]metaengine.FilterSpec{{Column: "score", Op: metaengine.FilterGe, Value: 30}},
		nil, nil, 0,
	)
	if err != nil {
		t.Fatalf("ScanRawValues FilterGe: %v", err)
	}
	if len(results.Items) != 3 {
		t.Errorf("FilterGe(30): expected 3 results, got %d", len(results.Items))
	}

	// Test FilterLt: score < 30 → should return 2 (10, 20).
	results, err = rawReader.ScanRawValues(
		ctx, "items",
		[]metaengine.FilterSpec{{Column: "score", Op: metaengine.FilterLt, Value: 30}},
		nil, nil, 0,
	)
	if err != nil {
		t.Fatalf("ScanRawValues FilterLt: %v", err)
	}
	if len(results.Items) != 2 {
		t.Errorf("FilterLt(30): expected 2 results, got %d", len(results.Items))
	}

	// Test FilterLe: score <= 30 → should return 3 (10, 20, 30).
	results, err = rawReader.ScanRawValues(
		ctx, "items",
		[]metaengine.FilterSpec{{Column: "score", Op: metaengine.FilterLe, Value: 30}},
		nil, nil, 0,
	)
	if err != nil {
		t.Fatalf("ScanRawValues FilterLe: %v", err)
	}
	if len(results.Items) != 3 {
		t.Errorf("FilterLe(30): expected 3 results, got %d", len(results.Items))
	}
}

func TestPebbleLayoutPlanner_NumericRangeMixedDigits(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := pebbleengine.NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	defer eng.Close()

	lp := eng.(metaengine.LayoutPlanner)
	if err := lp.ApplyLayout("items", []string{"score"}, nil); err != nil {
		t.Fatalf("ApplyLayout: %v", err)
	}

	mb := eng.(metaengine.MapBackend)

	// Scores with DIFFERENT digit counts: 5, 10, 100.
	// Without type-aware encoding, "10" < "100" < "5" lexicographically,
	// producing wrong range scan results.
	scores := []struct {
		key   string
		score int
	}{
		{"five", 5}, {"ten", 10}, {"hundred", 100},
	}

	for _, item := range scores {
		if err := mb.MapSet(ctx, "items", item.key, map[string]any{
			"score": item.score,
			"name":  item.key,
		}); err != nil {
			t.Fatalf("MapSet %s: %v", item.key, err)
		}
	}

	rawReader := eng.(metaengine.RawScanReader)

	// FilterGt(5) → 2 results (10, 100). Old encoding returns 0.
	results, err := rawReader.ScanRawValues(
		ctx, "items",
		[]metaengine.FilterSpec{{Column: "score", Op: metaengine.FilterGt, Value: 5}},
		nil, nil, 0,
	)
	if err != nil {
		t.Fatalf("ScanRawValues FilterGt: %v", err)
	}
	if len(results.Items) != 2 {
		t.Errorf("FilterGt(5): expected 2 results (10, 100), got %d", len(results.Items))
	}

	// FilterLt(10) → 1 result (5). Old encoding incorrectly includes 100.
	results, err = rawReader.ScanRawValues(
		ctx, "items",
		[]metaengine.FilterSpec{{Column: "score", Op: metaengine.FilterLt, Value: 10}},
		nil, nil, 0,
	)
	if err != nil {
		t.Fatalf("ScanRawValues FilterLt: %v", err)
	}
	if len(results.Items) != 1 {
		t.Errorf("FilterLt(10): expected 1 result (5), got %d", len(results.Items))
	}

	// FilterGe(100) → 1 result (100).
	results, err = rawReader.ScanRawValues(
		ctx, "items",
		[]metaengine.FilterSpec{{Column: "score", Op: metaengine.FilterGe, Value: 100}},
		nil, nil, 0,
	)
	if err != nil {
		t.Fatalf("ScanRawValues FilterGe: %v", err)
	}
	if len(results.Items) != 1 {
		t.Errorf("FilterGe(100): expected 1 result (100), got %d", len(results.Items))
	}
}

// --- Sort index tests ---

func TestPebbleLayoutPlanner_SortIndexAscending(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := pebbleengine.NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	defer eng.Close()

	lp := eng.(metaengine.LayoutPlanner)
	gomega.NewWithT(t).
		Expect(lp.ApplyLayout("tasks", nil, []string{"priority"})).
		To(gomega.Succeed())

	mb := eng.(metaengine.MapBackend)

	items := []struct {
		key      string
		priority int
	}{
		{"t1", 5}, {"t2", 1}, {"t3", 3}, {"t4", 4}, {"t5", 2},
	}

	for _, item := range items {
		gomega.NewWithT(t).Expect(mb.MapSet(ctx, "tasks", item.key, map[string]any{
			"priority": item.priority,
			"name":     item.key,
		})).To(gomega.Succeed())
	}

	rsr := eng.(metaengine.RawScanReader)
	sortSpec := &metaengine.SortSpec{Column: "priority", Desc: false}

	results, err := rsr.ScanRawValues(ctx, "tasks", nil, sortSpec, nil, 0)
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	gomega.NewWithT(t).Expect(results.Items).To(gomega.HaveLen(5))

	priorities := extractField[float64](t, results.Items, "priority")
	gomega.NewWithT(t).Expect(priorities).To(gomega.Equal([]float64{1, 2, 3, 4, 5}))
}

func TestPebbleLayoutPlanner_SortIndexDescending(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := pebbleengine.NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	defer eng.Close()

	lp := eng.(metaengine.LayoutPlanner)
	gomega.NewWithT(t).
		Expect(lp.ApplyLayout("tasks", nil, []string{"priority"})).
		To(gomega.Succeed())

	mb := eng.(metaengine.MapBackend)

	items := []struct {
		key      string
		priority int
	}{
		{"t1", 5}, {"t2", 1}, {"t3", 3}, {"t4", 4}, {"t5", 2},
	}

	for _, item := range items {
		gomega.NewWithT(t).Expect(mb.MapSet(ctx, "tasks", item.key, map[string]any{
			"priority": item.priority,
			"name":     item.key,
		})).To(gomega.Succeed())
	}

	rsr := eng.(metaengine.RawScanReader)
	sortSpec := &metaengine.SortSpec{Column: "priority", Desc: true}

	results, err := rsr.ScanRawValues(ctx, "tasks", nil, sortSpec, nil, 0)
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	gomega.NewWithT(t).Expect(results.Items).To(gomega.HaveLen(5))

	priorities := extractField[float64](t, results.Items, "priority")
	gomega.NewWithT(t).Expect(priorities).To(gomega.Equal([]float64{5, 4, 3, 2, 1}))
}

func TestPebbleLayoutPlanner_SortIndexCursorAscending(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := pebbleengine.NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	defer eng.Close()

	lp := eng.(metaengine.LayoutPlanner)
	gomega.NewWithT(t).Expect(lp.ApplyLayout("paged", nil, []string{"id"})).To(gomega.Succeed())

	mb := eng.(metaengine.MapBackend)

	for i := range 10 {
		gomega.NewWithT(t).Expect(mb.MapSet(ctx, "paged", i, map[string]any{
			"id": float64(i),
		})).To(gomega.Succeed())
	}

	rsr := eng.(metaengine.RawScanReader)
	sortSpec := &metaengine.SortSpec{Column: "id", Desc: false}

	// First page: items 0..2 + overflow.
	page1, err := rsr.ScanRawValues(ctx, "paged", nil, sortSpec, nil, 3)
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	gomega.NewWithT(t).Expect(page1.Items).To(gomega.HaveLen(4))

	ids1 := extractField[float64](t, page1.Items, "id")
	gomega.NewWithT(t).Expect(ids1).To(gomega.Equal([]float64{0, 1, 2, 3}))

	// Second page: cursor=2, skip items <= 2.
	page2, err := rsr.ScanRawValues(ctx, "paged", nil, sortSpec, float64(2), 3)
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	gomega.NewWithT(t).Expect(page2.Items).To(gomega.HaveLen(4))

	ids2 := extractField[float64](t, page2.Items, "id")
	gomega.NewWithT(t).Expect(ids2).To(gomega.Equal([]float64{3, 4, 5, 6}))
}

func TestPebbleLayoutPlanner_SortIndexCursorDescending(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := pebbleengine.NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	defer eng.Close()

	lp := eng.(metaengine.LayoutPlanner)
	gomega.NewWithT(t).Expect(lp.ApplyLayout("paged", nil, []string{"id"})).To(gomega.Succeed())

	mb := eng.(metaengine.MapBackend)

	for i := range 10 {
		gomega.NewWithT(t).Expect(mb.MapSet(ctx, "paged", i, map[string]any{
			"id": float64(i),
		})).To(gomega.Succeed())
	}

	rsr := eng.(metaengine.RawScanReader)
	sortSpec := &metaengine.SortSpec{Column: "id", Desc: true}

	// Cursor=7: skip items >= 7 in descending → [6, 5, 4, 3]
	results, err := rsr.ScanRawValues(ctx, "paged", nil, sortSpec, float64(7), 3)
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	gomega.NewWithT(t).Expect(results.Items).To(gomega.HaveLen(4))

	ids := extractField[float64](t, results.Items, "id")
	gomega.NewWithT(t).Expect(ids).To(gomega.Equal([]float64{6, 5, 4, 3}))
}

func TestPebbleLayoutPlanner_SortIndexFilterAndSort(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := pebbleengine.NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	defer eng.Close()

	lp := eng.(metaengine.LayoutPlanner)
	// Declare both filter and sort fields — the sort index path is preferred.
	gomega.NewWithT(t).Expect(lp.ApplyLayout("items", []string{"status"}, []string{"priority"})).To(
		gomega.Succeed(),
	)

	mb := eng.(metaengine.MapBackend)

	items := []struct {
		key      string
		status   string
		priority int
	}{
		{"s1", "open", 3},
		{"s2", "done", 5},
		{"s3", "open", 1},
		{"s4", "open", 2},
		{"s5", "done", 4},
	}

	for _, item := range items {
		gomega.NewWithT(t).Expect(mb.MapSet(ctx, "items", item.key, map[string]any{
			"status":   item.status,
			"priority": item.priority,
			"name":     item.key,
		})).To(gomega.Succeed())
	}

	rsr := eng.(metaengine.RawScanReader)
	filters := []metaengine.FilterSpec{{Column: "status", Op: metaengine.FilterEq, Value: "open"}}
	sortSpec := &metaengine.SortSpec{Column: "priority", Desc: false}

	results, err := rsr.ScanRawValues(ctx, "items", filters, sortSpec, nil, 0)
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	gomega.NewWithT(t).Expect(results.Items).To(gomega.HaveLen(3))

	priorities := extractField[float64](t, results.Items, "priority")
	gomega.NewWithT(t).Expect(priorities).To(gomega.Equal([]float64{1, 2, 3}))
}

func TestPebbleLayoutPlanner_SortIndexUpdateReindexes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := pebbleengine.NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	defer eng.Close()

	lp := eng.(metaengine.LayoutPlanner)
	gomega.NewWithT(t).
		Expect(lp.ApplyLayout("items", nil, []string{"priority"})).
		To(gomega.Succeed())

	mb := eng.(metaengine.MapBackend)

	gomega.NewWithT(t).Expect(mb.MapSet(ctx, "items", "k1", map[string]any{"priority": 1})).To(
		gomega.Succeed(),
	)
	gomega.NewWithT(t).Expect(mb.MapSet(ctx, "items", "k2", map[string]any{"priority": 2})).To(
		gomega.Succeed(),
	)

	mu := eng.(metaengine.MapUpdater)
	gomega.NewWithT(t).Expect(mu.MapUpdate(ctx, "items", "k1", func(prev any) any {
		m := prev.(map[string]any)
		m["priority"] = 5

		return m
	})).To(gomega.Succeed())

	rsr := eng.(metaengine.RawScanReader)
	sortSpec := &metaengine.SortSpec{Column: "priority", Desc: false}

	results, err := rsr.ScanRawValues(ctx, "items", nil, sortSpec, nil, 0)
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	gomega.NewWithT(t).Expect(results.Items).To(gomega.HaveLen(2))

	priorities := extractField[float64](t, results.Items, "priority")
	// k1 was updated to 5, so ascending order is [2, 5].
	gomega.NewWithT(t).Expect(priorities).To(gomega.Equal([]float64{2, 5}))
}

func TestPebbleLayoutPlanner_SortIndexDeleteRemovesIndex(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := pebbleengine.NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	defer eng.Close()

	lp := eng.(metaengine.LayoutPlanner)
	gomega.NewWithT(t).
		Expect(lp.ApplyLayout("items", nil, []string{"priority"})).
		To(gomega.Succeed())

	mb := eng.(metaengine.MapBackend)

	priorities := []struct {
		key      string
		priority int
	}{
		{"k1", 1}, {"k2", 2}, {"k3", 3},
	}

	for _, item := range priorities {
		gomega.NewWithT(t).Expect(mb.MapSet(ctx, "items", item.key, map[string]any{
			"priority": item.priority,
		})).To(gomega.Succeed())
	}

	gomega.NewWithT(t).Expect(mb.MapDelete(ctx, "items", "k2")).To(gomega.Succeed())

	rsr := eng.(metaengine.RawScanReader)
	sortSpec := &metaengine.SortSpec{Column: "priority", Desc: false}

	results, err := rsr.ScanRawValues(ctx, "items", nil, sortSpec, nil, 0)
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	gomega.NewWithT(t).Expect(results.Items).To(gomega.HaveLen(2))

	scores := extractField[float64](t, results.Items, "priority")
	gomega.NewWithT(t).Expect(scores).To(gomega.Equal([]float64{1, 3}))
}

func TestPebbleLayoutPlanner_SortIndexEarlyTermination(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := pebbleengine.NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	defer eng.Close()

	lp := eng.(metaengine.LayoutPlanner)
	gomega.NewWithT(t).Expect(lp.ApplyLayout("items", nil, []string{"score"})).To(gomega.Succeed())

	mb := eng.(metaengine.MapBackend)

	for i := range 100 {
		gomega.NewWithT(t).Expect(mb.MapSet(ctx, "items", fmt.Sprintf("k%d", i), map[string]any{
			"score": i,
		})).To(gomega.Succeed())
	}

	rsr := eng.(metaengine.RawScanReader)
	sortSpec := &metaengine.SortSpec{Column: "score", Desc: false}

	// limit=5 → 5+1=6 results (overflow detection).
	results, err := rsr.ScanRawValues(ctx, "items", nil, sortSpec, nil, 5)
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	gomega.NewWithT(t).Expect(results.Items).To(gomega.HaveLen(6))

	scores := extractField[float64](t, results.Items, "score")
	gomega.NewWithT(t).Expect(scores).To(gomega.Equal([]float64{0, 1, 2, 3, 4, 5}))
}

func TestPebbleLayoutPlanner_FilterIndexCursorAscending(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := pebbleengine.NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	defer eng.Close()

	// "score" is a filter field only (not a sort field) so the filter index
	// path is used, not the sort index path.
	lp := eng.(metaengine.LayoutPlanner)
	gomega.NewWithT(t).Expect(lp.ApplyLayout("items", []string{"score"}, nil)).To(gomega.Succeed())

	mb := eng.(metaengine.MapBackend)

	for i := 1; i <= 6; i++ {
		gomega.NewWithT(t).Expect(mb.MapSet(ctx, "items", fmt.Sprintf("k%d", i), map[string]any{
			"score": float64(i * 10),
		})).To(gomega.Succeed())
	}

	rsr := eng.(metaengine.RawScanReader)
	sortSpec := &metaengine.SortSpec{Column: "score", Desc: false}
	filter := []metaengine.FilterSpec{{Column: "score", Op: metaengine.FilterGe, Value: 10}}

	// Page 1: no cursor, limit=2 → 3 results (overflow).
	page1, err := rsr.ScanRawValues(ctx, "items", filter, sortSpec, nil, 2)
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	gomega.NewWithT(t).Expect(page1.Items).To(gomega.HaveLen(3))

	scores1 := extractField[float64](t, page1.Items, "score")
	gomega.NewWithT(t).Expect(scores1).To(gomega.Equal([]float64{10, 20, 30}))

	// Page 2: cursor=20, skip items <= 20.
	page2, err := rsr.ScanRawValues(ctx, "items", filter, sortSpec, float64(20), 2)
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	gomega.NewWithT(t).Expect(page2.Items).To(gomega.HaveLen(3))

	scores2 := extractField[float64](t, page2.Items, "score")
	gomega.NewWithT(t).Expect(scores2).To(gomega.Equal([]float64{30, 40, 50}))

	// Page 3: cursor=50, skip items <= 50 → only 60 remains.
	page3, err := rsr.ScanRawValues(ctx, "items", filter, sortSpec, float64(50), 2)
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	gomega.NewWithT(t).Expect(page3.Items).To(gomega.HaveLen(1))

	scores3 := extractField[float64](t, page3.Items, "score")
	gomega.NewWithT(t).Expect(scores3).To(gomega.Equal([]float64{60}))
}

func TestPebbleLayoutPlanner_FilterIndexCursorDescending(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := pebbleengine.NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	defer eng.Close()

	lp := eng.(metaengine.LayoutPlanner)
	gomega.NewWithT(t).Expect(lp.ApplyLayout("items", []string{"score"}, nil)).To(gomega.Succeed())

	mb := eng.(metaengine.MapBackend)

	for i := 1; i <= 6; i++ {
		gomega.NewWithT(t).Expect(mb.MapSet(ctx, "items", fmt.Sprintf("k%d", i), map[string]any{
			"score": float64(i * 10),
		})).To(gomega.Succeed())
	}

	rsr := eng.(metaengine.RawScanReader)
	sortSpec := &metaengine.SortSpec{Column: "score", Desc: true}
	filter := []metaengine.FilterSpec{{Column: "score", Op: metaengine.FilterGe, Value: 10}}

	// Descending: cursor=50, skip items >= 50 → [40, 30, 20].
	results, err := rsr.ScanRawValues(ctx, "items", filter, sortSpec, float64(50), 2)
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	gomega.NewWithT(t).Expect(results.Items).To(gomega.HaveLen(3))

	scores := extractField[float64](t, results.Items, "score")
	gomega.NewWithT(t).Expect(scores).To(gomega.Equal([]float64{40, 30, 20}))
}

func TestPebbleLayoutPlanner_SortIndexStringValues(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := pebbleengine.NewPebbleEngine("")
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	defer eng.Close()

	lp := eng.(metaengine.LayoutPlanner)
	gomega.NewWithT(t).Expect(lp.ApplyLayout("users", nil, []string{"name"})).To(gomega.Succeed())

	mb := eng.(metaengine.MapBackend)

	for _, name := range []string{"Charlie", "Alice", "Bob"} {
		gomega.NewWithT(t).Expect(mb.MapSet(ctx, "users", name, map[string]any{
			"name": name,
		})).To(gomega.Succeed())
	}

	rsr := eng.(metaengine.RawScanReader)
	sortSpec := &metaengine.SortSpec{Column: "name", Desc: false}

	results, err := rsr.ScanRawValues(ctx, "users", nil, sortSpec, nil, 0)
	gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())
	gomega.NewWithT(t).Expect(results.Items).To(gomega.HaveLen(3))

	names := extractField[string](t, results.Items, "name")
	gomega.NewWithT(t).Expect(names).To(gomega.Equal([]string{"Alice", "Bob", "Charlie"}))
}

// extractField decodes each raw JSON result and extracts the named field as
// the requested type T. Panics via t.Fatal on decode failure.
func extractField[T any](t *testing.T, rawResults [][]byte, field string) []T {
	t.Helper()

	values := make([]T, 0, len(rawResults))

	for _, raw := range rawResults {
		var decoded map[string]any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatalf("unmarshal result: %v", err)
		}

		val, ok := decoded[field].(T)
		if !ok {
			t.Fatalf("field %q is %T, not %T", field, decoded[field], *new(T))
		}

		values = append(values, val)
	}

	return values
}
