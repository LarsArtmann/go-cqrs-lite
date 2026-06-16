package kv

import (
	"errors"
	"testing"
)

func TestMemStore_BatchCommit(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer func() { _ = s.Close() }()

	_ = s.Set([]byte("existing"), []byte("old"))

	batch, err := s.Batch()
	if err != nil {
		t.Fatalf("Batch: %v", err)
	}

	_ = batch.Set([]byte("a"), []byte("1"))
	_ = batch.Set([]byte("b"), []byte("2"))
	_ = batch.Delete([]byte("existing"))

	err = batch.Commit()
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}

	val, _ := s.Get([]byte("a"))
	if string(val) != "1" {
		t.Errorf("batch Set a: got %q, want %q", val, "1")
	}

	val, _ = s.Get([]byte("b"))
	if string(val) != "2" {
		t.Errorf("batch Set b: got %q, want %q", val, "2")
	}

	_, err = s.Get([]byte("existing"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("batch Delete existing: err = %v, want ErrNotFound", err)
	}
}

func TestMemStore_BatchCloseDiscards(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer func() { _ = s.Close() }()

	batch, _ := s.Batch()
	_ = batch.Set([]byte("a"), []byte("1"))
	_ = batch.Close()

	err := batch.Commit()
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("Commit after Close: %v, want ErrClosed", err)
	}

	_, err = s.Get([]byte("a"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after discarded batch: err = %v, want ErrNotFound", err)
	}
}

func TestMemStore_BatchAfterCommit(t *testing.T) {
	t.Parallel()

	s := NewMemStore()
	defer func() { _ = s.Close() }()

	batch, _ := s.Batch()
	_ = batch.Set([]byte("a"), []byte("1"))
	_ = batch.Commit()

	err := batch.Set([]byte("b"), []byte("2"))
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("Set after Commit: %v, want ErrClosed", err)
	}
}
