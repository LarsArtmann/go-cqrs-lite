package http

import (
	"testing"
)

func TestDedupRing_Basic(t *testing.T) {
	t.Parallel()

	r := newDedupRing(4)

	r.Add("a")
	r.Add("b")
	r.Add("c")

	if !r.Has("a") {
		t.Error("expected a")
	}

	if !r.Has("b") {
		t.Error("expected b")
	}

	if !r.Has("c") {
		t.Error("expected c")
	}

	if r.Has("d") {
		t.Error("d should not be present")
	}

	if r.Len() != 3 {
		t.Errorf("Len: got %d, want 3", r.Len())
	}
}

func TestDedupRing_Eviction(t *testing.T) {
	t.Parallel()

	r := newDedupRing(3)

	r.Add("a")
	r.Add("b")
	r.Add("c")
	// Ring is full (cap 3). Adding "d" evicts "a".
	r.Add("d")

	if r.Has("a") {
		t.Error("a should have been evicted")
	}

	if !r.Has("d") {
		t.Error("d should be present")
	}

	if r.Len() != 3 {
		t.Errorf("Len after eviction: got %d, want 3", r.Len())
	}

	// Add more — evicts b, then c.
	r.Add("e")
	r.Add("f")

	if r.Has("b") {
		t.Error("b should have been evicted")
	}

	if !r.Has("f") {
		t.Error("f should be present")
	}
}

func TestDedupRing_DuplicateAdd(t *testing.T) {
	t.Parallel()

	r := newDedupRing(4)

	r.Add("x")
	r.Add("x") // no-op
	r.Add("x") // no-op

	if r.Len() != 1 {
		t.Errorf("Len after duplicate adds: got %d, want 1", r.Len())
	}
}

func TestDedupRing_NilSafe(t *testing.T) {
	t.Parallel()

	var r *dedupRing

	if r.Has("anything") {
		t.Error("nil ring should not contain anything")
	}
}

func TestDedupRing_LargeCapacity(t *testing.T) {
	t.Parallel()

	const cap = 1024
	r := newDedupRing(cap)

	// Fill beyond capacity to exercise wraparound.
	for i := range cap * 3 {
		r.Add(string(rune(i)))
	}

	// Only the last `cap` entries should be present.
	if r.Len() != cap {
		t.Errorf("Len: got %d, want %d", r.Len(), cap)
	}

	// The oldest entries should be evicted.
	if r.Has(string(rune(0))) {
		t.Error("first entry should have been evicted")
	}

	// The newest entries should be present.
	if !r.Has(string(rune(cap*3 - 1))) {
		t.Error("last entry should be present")
	}
}
