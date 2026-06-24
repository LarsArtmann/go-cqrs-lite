package storage_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/kv/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/v3"
)

func BenchmarkSQLViewStore_Set(b *testing.B) {
	db, err := storage.OpenSQLiteInMemory()
	if err != nil {
		b.Fatalf("OpenSQLiteInMemory: %v", err)
	}
	b.Cleanup(func() { _ = db.Close() })

	store, err := storage.NewSQLiteViewStore[testView, testKey](db, testMapper())
	if err != nil {
		b.Fatalf("NewSQLiteViewStore: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()

	for i := range b.N {
		key := testKey(fmt.Sprintf("key-%06d", i))
		if err := store.Set(ctx, key, &testView{Name: "bench", Age: i}); err != nil {
			b.Fatalf("Set: %v", err)
		}
	}
}

func BenchmarkSQLViewStore_Get(b *testing.B) {
	db, err := storage.OpenSQLiteInMemory()
	if err != nil {
		b.Fatalf("OpenSQLiteInMemory: %v", err)
	}
	b.Cleanup(func() { _ = db.Close() })

	store, err := storage.NewSQLiteViewStore[testView, testKey](db, testMapper())
	if err != nil {
		b.Fatalf("NewSQLiteViewStore: %v", err)
	}

	ctx := context.Background()

	for i := range 1000 {
		key := testKey(fmt.Sprintf("key-%06d", i))
		if err := store.Set(ctx, key, &testView{Name: "bench", Age: i}); err != nil {
			b.Fatalf("Seed Set: %v", err)
		}
	}

	b.ResetTimer()

	for i := range b.N {
		key := testKey(fmt.Sprintf("key-%06d", i%1000))
		if _, err := store.Get(ctx, key); err != nil {
			b.Fatalf("Get: %v", err)
		}
	}
}

func BenchmarkSQLViewStore_Query(b *testing.B) {
	db, err := storage.OpenSQLiteInMemory()
	if err != nil {
		b.Fatalf("OpenSQLiteInMemory: %v", err)
	}
	b.Cleanup(func() { _ = db.Close() })

	store, err := storage.NewSQLiteViewStore[testView, testKey](db, testMapper())
	if err != nil {
		b.Fatalf("NewSQLiteViewStore: %v", err)
	}

	ctx := context.Background()

	for i := range 1000 {
		key := testKey(fmt.Sprintf("key-%06d", i))
		if err := store.Set(ctx, key, &testView{Name: "bench", Age: i % 100}); err != nil {
			b.Fatalf("Seed Set: %v", err)
		}
	}

	b.ResetTimer()

	for i := range b.N {
		_, err := store.Query(ctx, kv.ViewQuery{
			Where:   "age = ?",
			Args:    []any{i % 100},
			OrderBy: "name",
			Limit:   10,
		})
		if err != nil {
			b.Fatalf("Query: %v", err)
		}
	}
}

func BenchmarkSQLViewStore_Scan(b *testing.B) {
	db, err := storage.OpenSQLiteInMemory()
	if err != nil {
		b.Fatalf("OpenSQLiteInMemory: %v", err)
	}
	b.Cleanup(func() { _ = db.Close() })

	store, err := storage.NewSQLiteViewStore[testView, testKey](db, testMapper())
	if err != nil {
		b.Fatalf("NewSQLiteViewStore: %v", err)
	}

	ctx := context.Background()

	for i := range 500 {
		key := testKey(fmt.Sprintf("key-%06d", i))
		if err := store.Set(ctx, key, &testView{Name: "bench", Age: i}); err != nil {
			b.Fatalf("Seed Set: %v", err)
		}
	}

	b.ResetTimer()

	for range b.N {
		if _, err := store.Scan(ctx, nil); err != nil {
			b.Fatalf("Scan: %v", err)
		}
	}
}

func BenchmarkSQLViewStore_BatchSet(b *testing.B) {
	db, err := storage.OpenSQLiteInMemory()
	if err != nil {
		b.Fatalf("OpenSQLiteInMemory: %v", err)
	}
	b.Cleanup(func() { _ = db.Close() })

	store, err := storage.NewSQLiteViewStore[testView, testKey](db, testMapper())
	if err != nil {
		b.Fatalf("NewSQLiteViewStore: %v", err)
	}

	ctx := context.Background()

	items := make([]kv.ViewItem[testView, testKey], 100)
	for i := range items {
		items[i] = kv.ViewItem[testView, testKey]{
			Key:   testKey(fmt.Sprintf("batch-%06d", i)),
			Value: &testView{Name: "batch", Age: i},
		}
	}

	b.ResetTimer()

	for range b.N {
		_ = store.DeleteAll(ctx)
		if err := store.BatchSet(ctx, items); err != nil {
			b.Fatalf("BatchSet: %v", err)
		}
	}
}
