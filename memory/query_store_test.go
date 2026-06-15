package memory_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

func TestMemoryQueryStore(t *testing.T) {
	ctx := context.Background()
	store := memory.NewMemoryQueryStore()

	t.Cleanup(func() { _ = store.Close() })

	search, err := query.NewPersistedQuery("user.search", []byte(`{"q":"alice"}`))
	if err != nil {
		t.Fatal(err)
	}

	count, err := query.NewPersistedQuery("user.count", []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}

	err = store.SaveQuery(ctx, search)
	if err != nil {
		t.Fatal(err)
	}

	err = store.SaveQuery(ctx, count)
	if err != nil {
		t.Fatal(err)
	}

	all, err := store.ReadAllQueries(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(all) != 2 {
		t.Fatalf("expected 2 queries, got %d", len(all))
	}

	result, err := store.ReadQueriesFrom(ctx, search.ID(), 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 query after first, got %d", len(result))
	}

	if result[0].Type() != "user.count" {
		t.Fatalf("expected user.count, got %s", result[0].Type())
	}
}
