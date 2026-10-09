package kv

import (
	"context"
	"errors"
	"testing"
)

func TestMemStore_BatchCommit(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer func() { _ = s.Close() }()

	_ = s.Set(context.Background(), []byte("existing"), []byte("old"))

	batch, err := s.Batch(context.Background())
	if err != nil {
		t.Fatalf("Batch: %v", err)
	}

	_ = batch.Set(context.Background(), []byte("a"), []byte("1"))
	_ = batch.Set(context.Background(), []byte("b"), []byte("2"))
	_ = batch.Delete(context.Background(), []byte("existing"))

	err = batch.Commit(context.Background())
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}

	val, _ := s.Get(context.Background(), []byte("a"))
	if string(val) != "1" {
		t.Errorf("batch Set a: got %q, want %q", val, "1")
	}

	val, _ = s.Get(context.Background(), []byte("b"))
	if string(val) != "2" {
		t.Errorf("batch Set b: got %q, want %q", val, "2")
	}

	_, err = s.Get(context.Background(), []byte("existing"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("batch Delete existing: err = %v, want ErrNotFound", err)
	}
}

func TestMemStore_BatchCloseDiscards(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer func() { _ = s.Close() }()

	batch, _ := s.Batch(context.Background())
	_ = batch.Set(context.Background(), []byte("a"), []byte("1"))
	_ = batch.Close()

	err := batch.Commit(context.Background())
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("Commit after Close: %v, want ErrClosed", err)
	}

	_, err = s.Get(context.Background(), []byte("a"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after discarded batch: err = %v, want ErrNotFound", err)
	}
}

func TestMemStore_BatchAfterCommit(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer func() { _ = s.Close() }()

	batch, _ := s.Batch(context.Background())
	_ = batch.Set(context.Background(), []byte("a"), []byte("1"))
	_ = batch.Commit(context.Background())

	err := batch.Set(context.Background(), []byte("b"), []byte("2"))
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("Set after Commit: %v, want ErrClosed", err)
	}
}
