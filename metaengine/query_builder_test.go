package metaengine_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestQueryBuilder verifies the fluent QueryBuilder API produces correct
// filtered/sorted/limited scan results.
func TestQueryBuilder(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	eng, err := metaengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("new sqlite engine: %v", err)
	}

	q := metaengine.Query[testItemInput, testItem](
		"items",
		metaengine.On(testItemCreated{}, func(e testItemCreated) (string, testItem) {
			return e.ID, testItem{ID: e.ID, Status: e.Status, Priority: e.Priority}
		}),
		metaengine.FilterOnField[testItem]("status", metaengine.FilterEq),
	)

	store, err := metaengine.Plan([]metaengine.Engine{metaengine.NewMemoryEngine(), eng}, q)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	defer store.Close()

	// Seed data.
	for _, item := range []testItemCreated{
		{ID: "a", Status: "active", Priority: 3},
		{ID: "b", Status: "active", Priority: 1},
		{ID: "c", Status: "archived", Priority: 2},
		{ID: "d", Status: "active", Priority: 5},
	} {
		if err := store.Apply(ctx, "testItemCreated", item); err != nil {
			t.Fatalf("apply: %v", err)
		}
	}

	reader := metaengine.NewReader[testItem](store, "items")

	// Filter by status=active, limit 2.
	results, err := metaengine.NewQueryBuilder(reader).
		Where("status", metaengine.FilterEq, "active").
		Limit(2).
		Execute(ctx)
	if err != nil {
		t.Fatalf("query builder execute: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	for _, r := range results {
		if r.Status != "active" {
			t.Errorf("expected status=active, got %s for %s", r.Status, r.ID)
		}
	}

	// Filter by status=archived.
	archived, err := metaengine.NewQueryBuilder(reader).
		Where("status", metaengine.FilterEq, "archived").
		Execute(ctx)
	if err != nil {
		t.Fatalf("query builder archived: %v", err)
	}

	if len(archived) != 1 {
		t.Errorf("expected 1 archived, got %d", len(archived))
	}

	if len(archived) > 0 && archived[0].ID != "c" {
		t.Errorf("expected archived item 'c', got %s", archived[0].ID)
	}

	// Get a single item via the builder.
	item, found, err := metaengine.NewQueryBuilder(reader).Get(ctx, "a")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if !found {
		t.Fatal("expected item 'a' to be found")
	}

	if item.Priority != 3 {
		t.Errorf("expected priority 3, got %d", item.Priority)
	}
}

type testItemInput struct {
	Status string
}

type testItem struct {
	ID       string
	Status   string
	Priority int
}

type testItemCreated struct {
	ID       string
	Status   string
	Priority int
}
