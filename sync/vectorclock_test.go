package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var sharedVC = struct {
	IdenticalAB VectorClock
	Concurrent1 VectorClock
	Concurrent2 VectorClock
	GreaterA    VectorClock
	LessB       VectorClock
	Superset    VectorClock
	Subset      VectorClock
	DisjointA   VectorClock
	DisjointB   VectorClock
}{
	IdenticalAB: VectorClock{NodeID("a"): 1, NodeID("b"): 2},
	Concurrent1: VectorClock{NodeID("a"): 3, NodeID("b"): 1},
	Concurrent2: VectorClock{NodeID("a"): 1, NodeID("b"): 3},
	GreaterA:    VectorClock{NodeID("a"): 5, NodeID("b"): 2},
	LessB:       VectorClock{NodeID("a"): 3, NodeID("b"): 2},
	Superset:    VectorClock{NodeID("a"): 3, NodeID("b"): 2},
	Subset:      VectorClock{NodeID("a"): 3},
	DisjointA:   VectorClock{NodeID("a"): 2},
	DisjointB:   VectorClock{NodeID("b"): 3},
}

func TestNewVectorClock(t *testing.T) {
	vc := NewVectorClock()
	if vc == nil {
		t.Fatal("NewVectorClock returned nil")
	}

	if len(vc) != 0 {
		t.Fatalf("expected empty vector clock, got %d entries", len(vc))
	}
}

func TestVectorClock_Increment(t *testing.T) {
	vc := NewVectorClock()

	vc.Increment(NodeID("node-a"))
	assert.Equal(t, int64(1), vc.Get(NodeID("node-a")), "first increment")

	vc.Increment(NodeID("node-a"))
	assert.Equal(t, int64(2), vc.Get(NodeID("node-a")), "second increment")

	vc.Increment(NodeID("node-b"))
	assert.Equal(t, int64(1), vc.Get(NodeID("node-b")), "new node")
	assert.Equal(t, int64(2), vc.Get(NodeID("node-a")), "original unchanged")
}

func TestVectorClock_Get_MissingNode(t *testing.T) {
	vc := NewVectorClock()
	assert.Equal(t, int64(0), vc.Get(NodeID("nonexistent")), "missing node returns 0")
}

func TestVectorClock_Merge(t *testing.T) {
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
			tt.base.Merge(tt.other)

			for node, expected := range tt.expected {
				if got := tt.base.Get(node); got != expected {
					t.Errorf("node %q: expected %d, got %d", node, expected, got)
				}
			}

			if len(tt.base) != len(tt.expected) {
				t.Errorf("expected %d nodes, got %d", len(tt.expected), len(tt.base))
			}
		})
	}
}

func TestVectorClock_Compare(t *testing.T) {
	tests := []struct {
		name     string
		a        VectorClock
		b        VectorClock
		expected int
	}{
		{
			name:     "empty clocks are equal",
			a:        NewVectorClock(),
			b:        NewVectorClock(),
			expected: 0,
		},
		{
			name:     "identical clocks are equal",
			a:        sharedVC.Superset,
			b:        sharedVC.Superset,
			expected: 0,
		},
		{
			name:     "a < b (happened before)",
			a:        VectorClock{NodeID("a"): 1, NodeID("b"): 2},
			b:        sharedVC.LessB,
			expected: -1,
		},
		{
			name:     "a > b (happened after)",
			a:        sharedVC.GreaterA,
			b:        sharedVC.LessB,
			expected: 1,
		},
		{
			name:     "concurrent clocks",
			a:        sharedVC.Concurrent1,
			b:        sharedVC.Concurrent2,
			expected: 0,
		},
		{
			name:     "one node vs empty",
			a:        VectorClock{NodeID("a"): 1},
			b:        NewVectorClock(),
			expected: 1,
		},
		{
			name:     "empty vs one node",
			a:        NewVectorClock(),
			b:        VectorClock{NodeID("a"): 1},
			expected: -1,
		},
		{
			name:     "superset clock is greater",
			a:        sharedVC.Superset,
			b:        sharedVC.Subset,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Compare(tt.b)
			if got != tt.expected {
				t.Errorf("Compare() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestVectorClock_Compare_Symmetric(t *testing.T) {
	a := VectorClock{NodeID("a"): 1}
	b := VectorClock{NodeID("a"): 3}

	if a.Compare(b) != -1 {
		t.Error("a < b expected")
	}

	if b.Compare(a) != 1 {
		t.Error("b > a expected")
	}
}

func TestVectorClock_Clone(t *testing.T) {
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
	original := NewVectorClock()
	cloned := original.Clone()

	if len(cloned) != 0 {
		t.Fatalf("clone of empty should be empty, got %d entries", len(cloned))
	}
}

func TestVectorClock_Equal(t *testing.T) {
	tests := []struct {
		name     string
		a        VectorClock
		b        VectorClock
		expected bool
	}{
		{
			name:     "empty clocks equal",
			a:        NewVectorClock(),
			b:        NewVectorClock(),
			expected: true,
		},
		{
			name:     "identical clocks equal",
			a:        sharedVC.IdenticalAB,
			b:        sharedVC.IdenticalAB,
			expected: true,
		},
		{
			name:     "concurrent clocks not equal",
			a:        sharedVC.Concurrent1,
			b:        sharedVC.Concurrent2,
			expected: false,
		},
		{
			name:     "different sizes not equal",
			a:        sharedVC.DisjointA,
			b:        sharedVC.IdenticalAB,
			expected: false,
		},
		{
			name:     "same compare result but different nodes",
			a:        sharedVC.DisjointA,
			b:        sharedVC.DisjointB,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Equal(tt.b); got != tt.expected {
				t.Errorf("Equal() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestVectorClock_Equal_Symmetric(t *testing.T) {
	a := VectorClock{NodeID("a"): 1, NodeID("b"): 2}
	b := VectorClock{NodeID("a"): 1, NodeID("b"): 2}

	if !a.Equal(b) || !b.Equal(a) {
		t.Error("Equal should be symmetric")
	}
}
