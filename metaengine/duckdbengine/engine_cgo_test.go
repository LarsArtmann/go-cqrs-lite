//go:build cgo

package duckdbengine_test

import (
	"context"
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestDuckDBEngine_MapBackend(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}
	defer eng.Close()

	ctx := context.Background()

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		t.Fatal("engine does not implement MapBackend")
	}

	type Task struct {
		ID    string
		Title string
	}

	if err := mb.MapSet(ctx, "tasks", "t1", Task{ID: "t1", Title: "Buy milk"}); err != nil {
		t.Fatal(err)
	}

	val, found, err := mb.MapGet(ctx, "tasks", "t1")
	if err != nil {
		t.Fatal(err)
	}

	if !found {
		t.Fatal("expected task t1 to exist")
	}

	m, ok := val.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", val)
	}

	if m["Title"] != "Buy milk" {
		t.Errorf("title: got %v, want %q", m["Title"], "Buy milk")
	}
}

func TestDuckDBEngine_CounterBackend(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}
	defer eng.Close()

	ctx := context.Background()

	cb, ok := eng.(metaengine.CounterBackend)
	if !ok {
		t.Fatal("engine does not implement CounterBackend")
	}

	if err := cb.CounterIncrement(ctx, "counts", metaengine.Delta{"open": 3, "closed": 1}); err != nil {
		t.Fatal(err)
	}

	if err := cb.CounterIncrement(ctx, "counts", metaengine.Delta{"open": 2}); err != nil {
		t.Fatal(err)
	}

	result, err := cb.CounterGet(ctx, "counts")
	if err != nil {
		t.Fatal(err)
	}

	if result["open"] != 5 {
		t.Errorf("open: got %d, want 5", result["open"])
	}

	if result["closed"] != 1 {
		t.Errorf("closed: got %d, want 1", result["closed"])
	}
}

func TestDuckDBEngine_Profile(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}
	defer eng.Close()

	profile := eng.Profile()

	if profile.Name != "duckdb" {
		t.Errorf("name: got %q, want %q", profile.Name, "duckdb")
	}

	counterC, ok := profile.Supports[metaengine.ADTCounter]
	if !ok {
		t.Fatal("expected ADTCounter support")
	}

	if counterC != metaengine.ComplexityO1 {
		t.Errorf("counter complexity: got %s, want %s", counterC, metaengine.ComplexityO1)
	}

	layout, ok := profile.Layouts[metaengine.ADTCounter]
	if !ok {
		t.Fatal("expected counter layout declaration")
	}

	if layout != metaengine.LayoutColumnar {
		t.Errorf("counter layout: got %s, want %s", layout, metaengine.LayoutColumnar)
	}
}

func TestDuckDBEngine_MetaenginePlan(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}
	defer eng.Close()

	type ItemCreated struct {
		Category string
		Count    int64
	}

	type CountInput struct{}

	store, err := metaengine.Plan(
		[]metaengine.Engine{eng},
		metaengine.Query[CountInput, map[string]int64]("category_counts",
			metaengine.On(ItemCreated{}, func(e ItemCreated) metaengine.Delta {
				return metaengine.Delta{e.Category: e.Count}
			}),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()

	if err := store.Apply(ctx, "ItemCreated", ItemCreated{Category: "books", Count: 5}); err != nil {
		t.Fatal(err)
	}

	if err := store.Apply(ctx, "ItemCreated", ItemCreated{Category: "books", Count: 3}); err != nil {
		t.Fatal(err)
	}

	result, err := metaengine.ExecuteTyped[CountInput, map[string]int64](ctx, store, CountInput{})
	if err != nil {
		t.Fatal(err)
	}

	if result["books"] != 8 {
		t.Errorf("books count: got %d, want 8", result["books"])
	}
}
