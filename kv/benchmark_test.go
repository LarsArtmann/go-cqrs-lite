package kv

import (
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
		_ = s.Set(key, val)
	}
}

func BenchmarkMemStore_Get(b *testing.B) {
	s := NewMemStore()
	defer func() { _ = s.Close() }()

	_ = s.Set([]byte("bench-key"), []byte("bench-value"))

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = s.Get([]byte("bench-key"))
	}
}

func BenchmarkMemStore_Has(b *testing.B) {
	s := NewMemStore()
	defer func() { _ = s.Close() }()

	_ = s.Set([]byte("bench-key"), []byte("bench-value"))

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = s.Has([]byte("bench-key"))
	}
}

func BenchmarkMemStore_Delete(b *testing.B) {
	s := NewMemStore()
	defer func() { _ = s.Close() }()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = s.Set([]byte("k"), []byte("v"))
		_ = s.Delete([]byte("k"))
	}
}

func BenchmarkMemStore_BatchCommit(b *testing.B) {
	s := NewMemStore()
	defer func() { _ = s.Close() }()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		batch, _ := s.Batch()
		for i := range 10 {
			_ = batch.Set([]byte(fmt.Sprintf("key-%d", i)), []byte("val"))
		}
		_ = batch.Commit()
	}
}

func BenchmarkMemStore_Iterator(b *testing.B) {
	s := NewMemStore()
	defer func() { _ = s.Close() }()

	for i := range 100 {
		_ = s.Set([]byte(fmt.Sprintf("key:%03d", i)), []byte("value"))
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		iter, _ := s.NewIterator([]byte("key:"))
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
