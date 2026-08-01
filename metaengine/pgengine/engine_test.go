package pgengine_test

import (
	"context"
	"os"
	"testing"

	pgengine "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// pgDSN returns a Postgres DSN for testing. It uses POSTGRES_TEST_DSN if set,
// otherwise skips the test. This mirrors the pattern from stack/postgres.
func pgDSN(t *testing.T) string {
	t.Helper()

	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}

	if dsn == "" {
		t.Skip("postgres not available: set POSTGRES_TEST_DSN or DATABASE_URL")
	}

	return dsn
}

func TestPostgresEngine_MapBackend(t *testing.T) {
	t.Parallel()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
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

func TestPostgresEngine_MapDelete(t *testing.T) {
	t.Parallel()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}

	defer eng.Close()

	ctx := context.Background()

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		t.Fatal("engine does not implement MapBackend")
	}

	if err := mb.MapSet(ctx, "items", "x", map[string]any{"v": float64(1)}); err != nil {
		t.Fatal(err)
	}

	if _, found, _ := mb.MapGet(ctx, "items", "x"); !found {
		t.Fatal("expected item x to exist before delete")
	}

	if err := mb.MapDelete(ctx, "items", "x"); err != nil {
		t.Fatal(err)
	}

	_, found, _ := mb.MapGet(ctx, "items", "x")
	if found {
		t.Fatal("expected item x to be deleted")
	}
}

func TestPostgresEngine_CounterBackend(t *testing.T) {
	t.Parallel()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
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

func TestPostgresEngine_Profile(t *testing.T) {
	t.Parallel()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}

	defer eng.Close()

	profile := eng.Profile()

	if profile.Name != "postgres" {
		t.Errorf("name: got %q, want %q", profile.Name, "postgres")
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

	if layout != metaengine.LayoutRow {
		t.Errorf("counter layout: got %s, want %s", layout, metaengine.LayoutRow)
	}
}

func TestPostgresEngine_MetaenginePlan(t *testing.T) {
	t.Parallel()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
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
