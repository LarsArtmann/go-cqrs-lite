package watermill

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
	r.Add("d") // evicts a

	if r.Has("a") {
		t.Error("a should have been evicted")
	}

	if !r.Has("d") {
		t.Error("d should be present")
	}

	if r.Len() != 3 {
		t.Errorf("Len after eviction: got %d, want 3", r.Len())
	}
}

func TestDedupRing_DuplicateAdd(t *testing.T) {
	t.Parallel()

	r := newDedupRing(4)

	r.Add("x")
	r.Add("x")
	r.Add("x")

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

func TestDedupRing_FallbackCapacity(t *testing.T) {
	t.Parallel()

	r := newDedupRing(0) // should fall back to dedupRingCapacity

	if r.Len() != 0 {
		t.Errorf("Len: got %d, want 0", r.Len())
	}

	if cap(r.buf) != dedupRingCapacity {
		t.Errorf("buffer capacity: got %d, want %d", cap(r.buf), dedupRingCapacity)
	}
}
