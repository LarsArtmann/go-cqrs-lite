package memory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func TestMemoryQueryStore(t *testing.T) {
	t.Parallel()

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

func TestMemoryQueryStore_LoadQueriesAfterTime(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryQueryStore()
	t.Cleanup(func() { _ = store.Close() })

	t1 := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	cutoff := time.Date(2025, 2, 1, 10, 0, 0, 0, time.UTC)

	old, err := query.NewPersistedQuery("user.search", nil, query.WithQueryReceivedAt(t1))
	if err != nil {
		t.Fatal(err)
	}

	recent, err := query.NewPersistedQuery("user.count", nil, query.WithQueryReceivedAt(cutoff))
	if err != nil {
		t.Fatal(err)
	}

	if err := store.SaveQuery(ctx, old); err != nil {
		t.Fatal(err)
	}

	if err := store.SaveQuery(ctx, recent); err != nil {
		t.Fatal(err)
	}

	result, err := store.LoadQueries(ctx, time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 query after cutoff, got %d", len(result))
	}

	if result[0].Type() != "user.count" {
		t.Errorf("expected user.count, got %s", result[0].Type())
	}
}

func TestMemoryQueryStore_ReadQueriesFromZeroID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryQueryStore()
	t.Cleanup(func() { _ = store.Close() })

	for range 3 {
		q, err := query.NewPersistedQuery("user.search", nil)
		if err != nil {
			t.Fatal(err)
		}

		if err := store.SaveQuery(ctx, q); err != nil {
			t.Fatal(err)
		}
	}

	result, err := store.ReadQueriesFrom(ctx, id.RequestID{}, 2)
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 queries from start, got %d", len(result))
	}
}

func TestMemoryQueryStore_ReadQueriesFromNonExistentID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryQueryStore()
	t.Cleanup(func() { _ = store.Close() })

	q, err := query.NewPersistedQuery("user.search", nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := store.SaveQuery(ctx, q); err != nil {
		t.Fatal(err)
	}

	otherID := id.NewRequestID()

	result, err := store.ReadQueriesFrom(ctx, otherID, 10)
	if err != nil {
		t.Fatal(err)
	}

	if result != nil {
		t.Fatalf("expected nil for non-existent query ID, got %d results", len(result))
	}
}

func TestMemoryQueryStore_EmptyStore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryQueryStore()
	t.Cleanup(func() { _ = store.Close() })

	all, err := store.ReadAllQueries(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(all) != 0 {
		t.Fatalf("expected 0 queries from empty store, got %d", len(all))
	}

	result, err := store.LoadQueries(ctx, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 0 {
		t.Fatalf("expected 0 queries from empty store LoadQueries, got %d", len(result))
	}

	from, err := store.ReadQueriesFrom(ctx, id.RequestID{}, 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(from) != 0 {
		t.Fatalf("expected 0 queries from empty store ReadQueriesFrom, got %d", len(from))
	}
}

func TestMemoryQueryStore_ClosedStore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryQueryStore()

	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	q, err := query.NewPersistedQuery("user.search", nil)
	if err != nil {
		t.Fatal(err)
	}

	err = store.SaveQuery(ctx, q)
	if err == nil {
		t.Fatal("expected error on SaveQuery after close")
	}

	if !errors.Is(err, query.ErrStoreClosed) {
		t.Errorf("expected ErrStoreClosed, got: %v", err)
	}

	_, err = store.LoadQueries(ctx, time.Now())
	if err == nil {
		t.Fatal("expected error on LoadQueries after close")
	}

	if !errors.Is(err, query.ErrStoreClosed) {
		t.Errorf("expected ErrStoreClosed, got: %v", err)
	}

	_, err = store.ReadAllQueries(ctx)
	if err == nil {
		t.Fatal("expected error on ReadAllQueries after close")
	}

	if !errors.Is(err, query.ErrStoreClosed) {
		t.Errorf("expected ErrStoreClosed, got: %v", err)
	}

	_, err = store.ReadQueriesFrom(ctx, id.RequestID{}, 10)
	if err == nil {
		t.Fatal("expected error on ReadQueriesFrom after close")
	}

	if !errors.Is(err, query.ErrStoreClosed) {
		t.Errorf("expected ErrStoreClosed, got: %v", err)
	}
}
