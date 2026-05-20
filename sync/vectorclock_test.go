package sync

import (
	"testing"
)

func TestNewVectorClock(t *testing.T) {
	t.Parallel()

	clock := NewVectorClock()
	if clock == nil {
		t.Fatal("NewVectorClock returned nil")
	}

	if len(clock) != 0 {
		t.Fatalf("expected empty vector clock, got %d entries", len(clock))
	}
}

func TestNewVectorClockFromMap(t *testing.T) {
	t.Parallel()

	entries := map[NodeID]int64{
		"node-a": 3,
		"node-b": 1,
	}

	clock := NewVectorClockFromMap(entries)

	if len(clock) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(clock))
	}

	if clock.Get("node-a") != 3 {
		t.Errorf("expected node-a=3, got %d", clock.Get("node-a"))
	}

	if clock.Get("node-b") != 1 {
		t.Errorf("expected node-b=1, got %d", clock.Get("node-b"))
	}
}

func TestNewVectorClockFromMap_Empty(t *testing.T) {
	t.Parallel()

	clock := NewVectorClockFromMap(nil)
	if len(clock) != 0 {
		t.Fatalf("expected empty clock, got %d entries", len(clock))
	}
}

func TestVectorClock_Increment(t *testing.T) {
	t.Parallel()

	clock := NewVectorClock()

	clock.Increment(NodeID("node-a"))
	if got := clock.Get(NodeID("node-a")); got != 1 {
		t.Errorf("first increment: got %d, want 1", got)
	}

	clock.Increment(NodeID("node-a"))
	if got := clock.Get(NodeID("node-a")); got != 2 {
		t.Errorf("second increment: got %d, want 2", got)
	}

	clock.Increment(NodeID("node-b"))
	if got := clock.Get(NodeID("node-b")); got != 1 {
		t.Errorf("new node: got %d, want 1", got)
	}

	if got := clock.Get(NodeID("node-a")); got != 2 {
		t.Errorf("original unchanged: got %d, want 2", got)
	}
}

func TestVectorClock_Get_MissingNode(t *testing.T) {
	t.Parallel()

	clock := NewVectorClock()
	if got := clock.Get(NodeID("nonexistent")); got != 0 {
		t.Errorf("missing node: got %d, want 0", got)
	}
}

func TestVectorClock_Merge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		base     VectorClock
		other    VectorClock
		expected VectorClock
	}{
		{
			name:     "empty into empty",
			base:     NewVectorClock(),
			other:    NewVectorClock(),
			expected: NewVectorClock(),
		},
		{
			name:     "non-empty into empty",
			base:     NewVectorClock(),
			other:    VectorClock{NodeID("a"): 3, NodeID("b"): 5},
			expected: VectorClock{NodeID("a"): 3, NodeID("b"): 5},
		},
		{
			name:     "empty into non-empty",
			base:     VectorClock{NodeID("a"): 3, NodeID("b"): 5},
			other:    NewVectorClock(),
			expected: VectorClock{NodeID("a"): 3, NodeID("b"): 5},
		},
		{
			name:     "merge takes max per node",
			base:     VectorClock{NodeID("a"): 3, NodeID("b"): 2},
			other:    VectorClock{NodeID("a"): 1, NodeID("b"): 5, NodeID("c"): 4},
			expected: VectorClock{NodeID("a"): 3, NodeID("b"): 5, NodeID("c"): 4},
		},
		{
			name:     "disjoint nodes merged",
			base:     VectorClock{NodeID("a"): 2},
			other:    VectorClock{NodeID("b"): 3},
			expected: VectorClock{NodeID("a"): 2, NodeID("b"): 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.base.Merge(tt.other)

			for node, expected := range tt.expected {
				if got := tt.base.Get(node); got != expected {
					t.Errorf("node %q: got %d, want %d", node, got, expected)
				}
			}

			if len(tt.base) != len(tt.expected) {
				t.Errorf("expected %d nodes, got %d", len(tt.expected), len(tt.base))
			}
		})
	}
}

func TestVectorClock_Compare(t *testing.T) {
	t.Parallel()

	identicalAB := VectorClock{NodeID("a"): 1, NodeID("b"): 2}
	greaterA := VectorClock{NodeID("a"): 5, NodeID("b"): 2}
	lessB := VectorClock{NodeID("a"): 3, NodeID("b"): 2}
	superset := VectorClock{NodeID("a"): 3, NodeID("b"): 2}
	subset := VectorClock{NodeID("a"): 3}

	tests := []struct {
		name     string
		a        VectorClock
		b        VectorClock
		expected int
	}{
		{"empty clocks are equal", NewVectorClock(), NewVectorClock(), 0},
		{"identical clocks are equal", superset, superset, 0},
		{"a < b (happened before)", VectorClock{NodeID("a"): 1, NodeID("b"): 2}, lessB, -1},
		{"a > b (happened after)", greaterA, lessB, 1},
		{
			"concurrent clocks",
			VectorClock{NodeID("a"): 3, NodeID("b"): 1},
			VectorClock{NodeID("a"): 1, NodeID("b"): 3},
			0,
		},
		{"one node vs empty", VectorClock{NodeID("a"): 1}, NewVectorClock(), 1},
		{"empty vs one node", NewVectorClock(), VectorClock{NodeID("a"): 1}, -1},
		{"superset clock is greater", superset, subset, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.a.Compare(tt.b)
			if got != tt.expected {
				t.Errorf("Compare() = %d, want %d", got, tt.expected)
			}
		})
	}

	if got := identicalAB.Get(NodeID("a")); got != 1 {
		t.Errorf("identicalAB sanity: got %d", got)
	}
}

func TestVectorClock_Compare_Symmetric(t *testing.T) {
	t.Parallel()

	clockA := VectorClock{NodeID("a"): 1}
	clockB := VectorClock{NodeID("a"): 3}

	if clockA.Compare(clockB) != -1 {
		t.Error("a < b expected")
	}

	if clockB.Compare(clockA) != 1 {
		t.Error("b > a expected")
	}
}

func TestVectorClock_Clone(t *testing.T) {
	t.Parallel()

	original := VectorClock{NodeID("a"): 3, NodeID("b"): 5}
	cloned := original.Clone()

	if !original.Equal(cloned) {
		t.Fatal("clone should be equal to original")
	}

	cloned.Increment(NodeID("a"))

	if original.Get(NodeID("a")) != 3 {
		t.Fatalf("modifying clone should not affect original, got %d", original.Get(NodeID("a")))
	}
}

func TestVectorClock_Clone_Empty(t *testing.T) {
	t.Parallel()

	original := NewVectorClock()
	cloned := original.Clone()

	if len(cloned) != 0 {
		t.Fatalf("clone of empty should be empty, got %d entries", len(cloned))
	}
}

func TestVectorClock_Equal(t *testing.T) {
	t.Parallel()

	identicalAB := VectorClock{NodeID("a"): 1, NodeID("b"): 2}
	concurrent1 := VectorClock{NodeID("a"): 3, NodeID("b"): 1}
	concurrent2 := VectorClock{NodeID("a"): 1, NodeID("b"): 3}
	disjointA := VectorClock{NodeID("a"): 2}
	disjointB := VectorClock{NodeID("b"): 3}

	tests := []struct {
		name     string
		a        VectorClock
		b        VectorClock
		expected bool
	}{
		{"empty clocks equal", NewVectorClock(), NewVectorClock(), true},
		{"identical clocks equal", identicalAB, identicalAB, true},
		{"concurrent clocks not equal", concurrent1, concurrent2, false},
		{"different sizes not equal", disjointA, identicalAB, false},
		{"same compare result but different nodes", disjointA, disjointB, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.a.Equal(tt.b); got != tt.expected {
				t.Errorf("Equal() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestVectorClock_Equal_Symmetric(t *testing.T) {
	t.Parallel()

	a := VectorClock{NodeID("a"): 1, NodeID("b"): 2}
	b := VectorClock{NodeID("a"): 1, NodeID("b"): 2}

	if !a.Equal(b) || !b.Equal(a) {
		t.Error("Equal should be symmetric")
	}
}

func TestVectorClock_Cmp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        VectorClock
		b        VectorClock
		expected ClockOrder
	}{
		{"empty clocks are equal", NewVectorClock(), NewVectorClock(), OrderEqual},
		{
			"identical clocks are equal",
			VectorClock{NodeID("a"): 1},
			VectorClock{NodeID("a"): 1},
			OrderEqual,
		},
		{"before", VectorClock{NodeID("a"): 1}, VectorClock{NodeID("a"): 3}, OrderBefore},
		{"after", VectorClock{NodeID("a"): 3}, VectorClock{NodeID("a"): 1}, OrderAfter},
		{
			"concurrent",
			VectorClock{NodeID("a"): 3, NodeID("b"): 1},
			VectorClock{NodeID("a"): 1, NodeID("b"): 3},
			OrderConcurrent,
		},
		{"empty vs non-empty", NewVectorClock(), VectorClock{NodeID("a"): 1}, OrderBefore},
		{
			"superset is after",
			VectorClock{NodeID("a"): 3, NodeID("b"): 2},
			VectorClock{NodeID("a"): 3},
			OrderAfter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.a.Cmp(tt.b)
			if got != tt.expected {
				t.Errorf("Cmp() = %v (%s), want %v (%s)", got, got, tt.expected, tt.expected)
			}
		})
	}
}

func TestClockOrder_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		order    ClockOrder
		expected string
	}{
		{OrderBefore, "before"},
		{OrderAfter, "after"},
		{OrderConcurrent, "concurrent"},
		{OrderEqual, "equal"},
		{ClockOrder(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()

			if got := tt.order.String(); got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}
