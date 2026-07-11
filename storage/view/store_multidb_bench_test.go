package view

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

// BenchmarkSQLViewStore_MultiDB_vs_SingleDB compares read-model Set/Get
// performance when the view table shares a database file vs has its own.
// Multi-DB isolates view scans from concurrent event writes.
func BenchmarkSQLViewStore_MultiDB_vs_SingleDB(b *testing.B) {
	mapper := testMapper()

	b.Run("SingleDB", func(b *testing.B) {
		db, err := openSQLiteInMemory()
		if err != nil {
			b.Fatal(err)
		}
		db.SetMaxOpenConns(1)
		b.Cleanup(func() { _ = db.Close() })

		store, err := NewSQLiteViewStore[testView, testKey](db, mapper)
		if err != nil {
			b.Fatal(err)
		}

		ctx := context.Background()
		b.ResetTimer()

		for i := range b.N {
			key := testKey(fmt.Sprintf("k-%06d", i))
			_ = store.Set(ctx, key, &testView{Name: "bench", Age: i})
		}
	})

	b.Run("MultiDB_ViewStore", func(b *testing.B) {
		dir := b.TempDir()

		viewDB, err := sql.Open("sqlite", filepath.Join(dir, "views.db"))
		if err != nil {
			b.Fatal(err)
		}
		viewDB.SetMaxOpenConns(1)
		b.Cleanup(func() { _ = viewDB.Close() })

		store, err := NewSQLiteViewStore[testView, testKey](viewDB, mapper)
		if err != nil {
			b.Fatal(err)
		}

		ctx := context.Background()
		b.ResetTimer()

		for i := range b.N {
			key := testKey(fmt.Sprintf("k-%06d", i))
			_ = store.Set(ctx, key, &testView{Name: "bench", Age: i})
		}
	})
}

// BenchmarkKV_vs_SQL_Comparison provides side-by-side Set/Get/Scan benchmarks
// for KV-backed (TypedStore over MemStore) vs SQL-backed (SQLViewStore) view
// stores. Run with: go test -bench=BenchmarkKV_vs_SQL -benchmem.
func BenchmarkKV_vs_SQL_Comparison(b *testing.B) {
	ctx := context.Background()

	b.Run("KV/Set", func(b *testing.B) {
		memStore := kv.NewMemStore()
		b.Cleanup(func() { _ = memStore.Close() })

		store := kv.NewTypedStore[testView, testKey](memStore)
		b.ResetTimer()

		for i := range b.N {
			_ = store.Set(ctx, testKey(fmt.Sprintf("k-%06d", i)), &testView{Name: "bench", Age: i})
		}
	})

	b.Run("SQL/Set", func(b *testing.B) {
		db, err := openSQLiteInMemory()
		if err != nil {
			b.Fatal(err)
		}
		db.SetMaxOpenConns(1)
		b.Cleanup(func() { _ = db.Close() })

		store, err := NewSQLiteViewStore[testView, testKey](db, testMapper())
		if err != nil {
			b.Fatal(err)
		}
		b.ResetTimer()

		for i := range b.N {
			_ = store.Set(ctx, testKey(fmt.Sprintf("k-%06d", i)), &testView{Name: "bench", Age: i})
		}
	})

	// Seed for Get/Scan benchmarks.
	const seedCount = 500

	b.Run("KV/Get", func(b *testing.B) {
		memStore := kv.NewMemStore()
		b.Cleanup(func() { _ = memStore.Close() })

		store := kv.NewTypedStore[testView, testKey](memStore)
		for i := range seedCount {
			_ = store.Set(ctx, testKey(fmt.Sprintf("k-%06d", i)), &testView{Name: "bench", Age: i})
		}
		b.ResetTimer()

		for i := range b.N {
			_, _ = store.Get(ctx, testKey(fmt.Sprintf("k-%06d", i%seedCount)))
		}
	})

	b.Run("SQL/Get", func(b *testing.B) {
		db, err := openSQLiteInMemory()
		if err != nil {
			b.Fatal(err)
		}
		db.SetMaxOpenConns(1)
		b.Cleanup(func() { _ = db.Close() })

		store, err := NewSQLiteViewStore[testView, testKey](db, testMapper())
		if err != nil {
			b.Fatal(err)
		}
		for i := range seedCount {
			_ = store.Set(ctx, testKey(fmt.Sprintf("k-%06d", i)), &testView{Name: "bench", Age: i})
		}
		b.ResetTimer()

		for i := range b.N {
			_, _ = store.Get(ctx, testKey(fmt.Sprintf("k-%06d", i%seedCount)))
		}
	})

	b.Run("KV/Scan", func(b *testing.B) {
		memStore := kv.NewMemStore()
		b.Cleanup(func() { _ = memStore.Close() })

		store := kv.NewTypedStore[testView, testKey](memStore)
		for i := range seedCount {
			_ = store.Set(ctx, testKey(fmt.Sprintf("k-%06d", i)), &testView{Name: "bench", Age: i})
		}
		b.ResetTimer()

		for range b.N {
			_, _ = store.Scan(ctx, nil)
		}
	})

	b.Run("SQL/Scan", func(b *testing.B) {
		db, err := openSQLiteInMemory()
		if err != nil {
			b.Fatal(err)
		}
		db.SetMaxOpenConns(1)
		b.Cleanup(func() { _ = db.Close() })

		store, err := NewSQLiteViewStore[testView, testKey](db, testMapper())
		if err != nil {
			b.Fatal(err)
		}
		for i := range seedCount {
			_ = store.Set(ctx, testKey(fmt.Sprintf("k-%06d", i)), &testView{Name: "bench", Age: i})
		}
		b.ResetTimer()

		for range b.N {
			_, _ = store.Scan(ctx, nil)
		}
	})
}
