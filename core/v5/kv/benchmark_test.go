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
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		if err := s.Set(ctx, key, val); err != nil {
			b.Fatalf("Set: %v", err)
		}
	}

	got, err := s.Get(ctx, key)
	if err != nil || string(got) != string(val) {
		b.Fatalf("post-loop Get: got=%q err=%v, want %q", got, err, val)
	}
}

func BenchmarkMemStore_Get(b *testing.B) {
	s := NewMemStore()
	defer func() { _ = s.Close() }()

	ctx := context.Background()
	key := []byte("bench-key")
	val := []byte("bench-value")

	if err := s.Set(ctx, key, val); err != nil {
		b.Fatalf("seed Set: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		got, err := s.Get(ctx, key)
		if err != nil {
			b.Fatalf("Get: %v", err)
		}
		if string(got) != string(val) {
			b.Fatalf("Get: got %q, want %q", got, val)
		}
	}
}

func BenchmarkMemStore_Has(b *testing.B) {
	s := NewMemStore()
	defer func() { _ = s.Close() }()

	ctx := context.Background()
	key := []byte("bench-key")

	if err := s.Set(ctx, key, []byte("bench-value")); err != nil {
		b.Fatalf("seed Set: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		found, err := s.Has(ctx, key)
		if err != nil {
			b.Fatalf("Has: %v", err)
		}
		if !found {
			b.Fatal("Has: expected found=true for existing key")
		}
	}
}

func BenchmarkMemStore_Delete(b *testing.B) {
	s := NewMemStore()
	defer func() { _ = s.Close() }()

	ctx := context.Background()
	key := []byte("k")
	val := []byte("v")

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		if err := s.Set(ctx, key, val); err != nil {
			b.Fatalf("Set: %v", err)
		}
		if err := s.Delete(ctx, key); err != nil {
			b.Fatalf("Delete: %v", err)
		}
	}

	found, err := s.Has(ctx, key)
	if err != nil {
		b.Fatalf("post-loop Has: %v", err)
	}
	if found {
		b.Fatal("post-loop Has: key still exists after Delete")
	}
}

func BenchmarkMemStore_BatchCommit(b *testing.B) {
	s := NewMemStore()
	defer func() { _ = s.Close() }()

	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		batch, err := s.Batch(ctx)
		if err != nil {
			b.Fatalf("Batch: %v", err)
		}
		for i := range 10 {
			if err := batch.Set(ctx, fmt.Appendf(nil, "key-%d", i), []byte("val")); err != nil {
				b.Fatalf("batch.Set %d: %v", i, err)
			}
		}
		if err := batch.Commit(ctx); err != nil {
			b.Fatalf("Commit: %v", err)
		}
	}

	got, err := s.Get(ctx, []byte("key-0"))
	if err != nil {
		b.Fatalf("post-loop Get key-0: %v", err)
	}
	if string(got) != "val" {
		b.Fatalf("post-loop Get key-0: got %q, want %q", got, "val")
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
