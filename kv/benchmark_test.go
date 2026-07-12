package kv

import (
	"context"
	"fmt"
	"testing"
)

func BenchmarkMemStore_Set(b *testing.B) {
	s := NewMemStore()
	defer func() { _ = s.Close() }()

	key := []byte("bench-key")
	val := []byte("bench-value")

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = s.Set(context.Background(), key, val)
	}
}

func BenchmarkMemStore_Get(b *testing.B) {
	s := NewMemStore()
	defer func() { _ = s.Close() }()

	_ = s.Set(context.Background(), []byte("bench-key"), []byte("bench-value"))

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = s.Get(context.Background(), []byte("bench-key"))
	}
}

func BenchmarkMemStore_Has(b *testing.B) {
	s := NewMemStore()
	defer func() { _ = s.Close() }()

	_ = s.Set(context.Background(), []byte("bench-key"), []byte("bench-value"))

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = s.Has(context.Background(), []byte("bench-key"))
	}
}

func BenchmarkMemStore_Delete(b *testing.B) {
	s := NewMemStore()
	defer func() { _ = s.Close() }()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = s.Set(context.Background(), []byte("k"), []byte("v"))
		_ = s.Delete(context.Background(), []byte("k"))
	}
}

func BenchmarkMemStore_BatchCommit(b *testing.B) {
	s := NewMemStore()
	defer func() { _ = s.Close() }()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		batch, _ := s.Batch(context.Background())
		for i := range 10 {
			_ = batch.Set(context.Background(), fmt.Appendf(nil, "key-%d", i), []byte("val"))
		}
		_ = batch.Commit(context.Background())
	}
}

func BenchmarkMemStore_Iterator(b *testing.B) {
	s := NewMemStore()
	defer func() { _ = s.Close() }()

	for i := range 100 {
		_ = s.Set(context.Background(), fmt.Appendf(nil, "key:%03d", i), []byte("value"))
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		iter, _ := s.NewIterator(context.Background(), []byte("key:"))
		count := 0
		for iter.Next() {
			count++
		}
		_ = iter.Close()
		if count != 100 {
			b.Fatalf("expected 100 keys, got %d", count)
		}
	}
}
