package memory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4/querytest"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func TestMemoryQueryStore_Suite(t *testing.T) {
	t.Parallel()

	querytest.RunStoreSuite(t, func(t *testing.T) querytest.StoreSuite {
		store := memory.NewMemoryQueryStore()
		t.Cleanup(func() { _ = store.Close() })

		return store
	})
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
