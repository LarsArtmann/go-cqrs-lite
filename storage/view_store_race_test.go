package storage_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/kv/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/v3"
)

func TestSQLViewStore_ConcurrentSetQuery(t *testing.T) {
	t.Parallel()

	db, err := storage.OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("OpenSQLiteInMemory: %v", err)
	}
	db.SetMaxOpenConns(1) // in-memory SQLite needs single connection
	t.Cleanup(func() { _ = db.Close() })

	store, err := storage.NewSQLiteViewStore[testView, testKey](db, testMapper())
	if err != nil {
		t.Fatalf("NewSQLiteViewStore: %v", err)
	}

	ctx := context.Background()

	// Seed with initial data.
	for i := range 20 {
		key := testKey(fmt.Sprintf("seed-%02d", i))
		if err := store.Set(
			ctx,
			key,
			&testView{Name: fmt.Sprintf("Seed%d", i), Age: i},
		); err != nil {
			t.Fatalf("Seed Set %d: %v", i, err)
		}
	}

	const goroutines = 20
	const opsPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// Half the goroutines write.
	for g := range goroutines {
		go func(g int) {
			defer wg.Done()

			for op := range opsPerGoroutine {
				key := testKey(fmt.Sprintf("w-%02d-%02d", g, op))
				view := &testView{Name: fmt.Sprintf("W%d-%d", g, op), Age: op}
				if err := store.Set(ctx, key, view); err != nil {
					t.Errorf("Concurrent Set %s: %v", key, err)
					return
				}
			}
		}(g)
	}

	// Half the goroutines read/query.
	for g := range goroutines {
		go func(g int) {
			defer wg.Done()

			for op := range opsPerGoroutine {
				_, err := store.Query(ctx, kv.ViewQuery{
					Conditions: []kv.Condition{{Column: "age", Op: kv.OpGte, Value: 0}},
					OrderBy:    "key",
					Limit:      10,
				})
				if err != nil {
					t.Errorf("Concurrent Query g=%d op=%d: %v", g, op, err)
					return
				}
			}
		}(g)
	}

	wg.Wait()

	// Verify data integrity after concurrent access.
	count, err := store.Count(ctx, kv.ViewQuery{})
	if err != nil {
		t.Fatalf("Count after concurrent: %v", err)
	}

	// 20 seed + 20*50 writes = 1020.
	if count != 1020 {
		t.Fatalf("Count after concurrent: got %d, want 1020", count)
	}
}

func TestSQLViewStore_ConcurrentBatchAndCount(t *testing.T) {
	t.Parallel()

	db, err := storage.OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("OpenSQLiteInMemory: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	store, err := storage.NewSQLiteViewStore[testView, testKey](db, testMapper())
	if err != nil {
		t.Fatalf("NewSQLiteViewStore: %v", err)
	}

	ctx := context.Background()

	const batches = 10
	const batchSize = 30

	var wg sync.WaitGroup
	wg.Add(batches)

	for b := range batches {
		go func(b int) {
			defer wg.Done()

			items := make([]kv.ViewItem[testView, testKey], 0, batchSize)
			for i := range batchSize {
				items = append(items, kv.ViewItem[testView, testKey]{
					Key:   testKey(fmt.Sprintf("batch-%d-%d", b, i)),
					Value: &testView{Name: fmt.Sprintf("B%dI%d", b, i), Age: i},
				})
			}

			if err := store.BatchSet(ctx, items); err != nil {
				t.Errorf("Concurrent BatchSet %d: %v", b, err)
			}
		}(b)
	}

	wg.Wait()

	count, err := store.Count(ctx, kv.ViewQuery{})
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != batches*batchSize {
		t.Fatalf("Count: got %d, want %d", count, batches*batchSize)
	}
}
