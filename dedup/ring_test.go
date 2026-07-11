package dedup_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/dedup/v3"
)

func TestRing_Basic(t *testing.T) {
	t.Parallel()

	r := dedup.NewRing(4)

	r.Add("a")
	r.Add("b")
	r.Add("c")

	for _, id := range []string{"a", "b", "c"} {
		if !r.Has(id) {
			t.Errorf("expected %q in ring", id)
		}
	}

	if r.Has("d") {
		t.Error("d should not be present")
	}

	if r.Len() != 3 {
		t.Errorf("Len: got %d, want 3", r.Len())
	}

	if r.Capacity() != 4 {
		t.Errorf("Capacity: got %d, want 4", r.Capacity())
	}
}

func TestRing_Eviction(t *testing.T) {
	t.Parallel()

	r := dedup.NewRing(3)

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

func TestRing_DuplicateAdd(t *testing.T) {
	t.Parallel()

	r := dedup.NewRing(4)

	r.Add("x")
	r.Add("x") // no-op
	r.Add("x") // no-op

	if r.Len() != 1 {
		t.Errorf("Len after duplicate adds: got %d, want 1", r.Len())
	}
}

func TestRing_NilSafe(t *testing.T) {
	t.Parallel()

	var r *dedup.Ring

	if r.Has("anything") {
		t.Error("nil ring should not contain anything")
	}

	if r.Len() != 0 {
		t.Error("nil ring Len should be 0")
	}
}

func TestRing_LargeCapacity_Wraparound(t *testing.T) {
	t.Parallel()

	const capacity = 1024
	r := dedup.NewRing(capacity)

	// Fill beyond capacity to exercise wraparound.
	for i := range capacity * 3 {
		r.Add(string(rune(i)))
	}

	if r.Len() != capacity {
		t.Errorf("Len: got %d, want %d", r.Len(), capacity)
	}

	// The oldest entries should be evicted.
	if r.Has(string(rune(0))) {
		t.Error("first entry should have been evicted")
	}

	// The newest entries should be present.
	if !r.Has(string(rune(capacity*3 - 1))) {
		t.Error("last entry should be present")
	}
}

func TestRing_DefaultCapacityFallback(t *testing.T) {
	t.Parallel()

	r := dedup.NewRing(0) // should fall back to DefaultCapacity
	if r.Capacity() != dedup.DefaultCapacity {
		t.Errorf("Capacity: got %d, want default %d", r.Capacity(), dedup.DefaultCapacity)
	}

	rNegative := dedup.NewRing(-5)
	if rNegative.Capacity() != dedup.DefaultCapacity {
		t.Errorf("Capacity: got %d, want default %d", rNegative.Capacity(), dedup.DefaultCapacity)
	}
}

// TestRing_RingShapeInvariants is a property-based test verifying that:
//  1. Len never exceeds Capacity
//  2. Evicted IDs are no longer Has-able
//  3. A full ring's Len == Capacity, and stays there as more IDs are added
func TestRing_RingShapeInvariants(t *testing.T) {
	t.Parallel()

	const capacity = 8
	r := dedup.NewRing(capacity)
	seen := make(map[string]bool)
	idGen := func(i int) string {
		// Use a sparse string space to avoid collisions within the test run.
		return string(rune('a'+i%26)) + "-" + itoa(i)
	}

	for i := range capacity * 5 {
		id := idGen(i)
		r.Add(id)
		seen[id] = true

		if r.Len() > capacity {
			t.Fatalf("iteration %d: Len %d exceeded capacity %d", i, r.Len(), capacity)
		}

		// The last `capacity` unique IDs should be present; older ones may have been evicted.
		if i >= capacity {
			oldest := idGen(i - capacity)
			if r.Has(oldest) {
				t.Fatalf("iteration %d: oldest-in-window %q should have been evicted", i, oldest)
			}
		}
	}

	if r.Len() != capacity {
		t.Errorf("after saturation: Len %d, want %d", r.Len(), capacity)
	}
}

// itoa is a tiny strconv.Itoa-free helper to avoid the import for one use.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	out := make([]byte, 0, 8)

	for n > 0 {
		out = append([]byte{byte('0' + n%10)}, out...)
		n /= 10
	}

	return string(out)
}
