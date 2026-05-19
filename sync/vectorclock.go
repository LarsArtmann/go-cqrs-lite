package sync

// VectorClock tracks logical time across nodes for causal ordering.
// It maps node identifiers to monotonically increasing counters.
//
// Vector clocks enable detection of concurrent operations and causal relationships:
//   - If A < B, then A "happened before" B (causal order)
//   - If A || B, then A and B are concurrent (potential conflict)
type VectorClock map[NodeID]int64

// NewVectorClock creates a new empty vector clock.
func NewVectorClock() VectorClock {
	return make(VectorClock)
}

// NewVectorClockFromMap creates a VectorClock from a map of node IDs to counters.
func NewVectorClockFromMap(entries map[NodeID]int64) VectorClock {
	vc := make(VectorClock, len(entries))

	for node, t := range entries {
		vc[node] = t
	}

	return vc
}

// Increment increments the clock counter for a node.
func (vc VectorClock) Increment(nodeID NodeID) {
	vc[nodeID]++
}

// Get returns the counter value for a node, or 0 if not present.
func (vc VectorClock) Get(nodeID NodeID) int64 {
	return vc[nodeID]
}

// Merge merges another vector clock into this one, taking the maximum value
// for each node. This establishes causality: after Merge, this clock
// "knows about" everything the other clock knows about.
func (vc VectorClock) Merge(other VectorClock) {
	for node, t := range other {
		if current, exists := vc[node]; !exists || t > current {
			vc[node] = t
		}
	}
}

// Compare compares two vector clocks for causal ordering.
//
// Returns:
//   - -1 if vc "happened before" other (vc < other)
//   - 1 if vc "happened after" other (vc > other)
//   - 0 if they are concurrent or equal (vc || other)
func (vc VectorClock) Compare(other VectorClock) int {
	allNodes := make(map[NodeID]bool)
	for node := range vc {
		allNodes[node] = true
	}

	for node := range other {
		allNodes[node] = true
	}

	less, greater := false, false

	for node := range allNodes {
		v1, v2 := vc[node], other[node]

		if v1 < v2 {
			less = true
		} else if v1 > v2 {
			greater = true
		}
	}

	if less && !greater {
		return -1
	}

	if greater && !less {
		return 1
	}

	return 0
}

// Clone creates a deep copy of the vector clock.
func (vc VectorClock) Clone() VectorClock {
	clone := NewVectorClock()
	clone.Merge(vc)

	return clone
}

// Equal returns true if two vector clocks have identical values for all nodes.
func (vc VectorClock) Equal(other VectorClock) bool {
	if len(vc) != len(other) {
		return false
	}

	for node, val := range vc {
		if other.Get(node) != val {
			return false
		}
	}

	return true
}
