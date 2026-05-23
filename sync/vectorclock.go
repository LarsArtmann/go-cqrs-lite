package sync

import (
	"maps"
)

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
// Returns an error if any counter is negative.
func NewVectorClockFromMap(entries map[NodeID]int64) (VectorClock, error) {
	for node, counter := range entries {
		if counter < 0 {
			return nil, NegativeCounterError{Node: node, Counter: counter}
		}
	}

	if entries == nil {
		return NewVectorClock(), nil
	}

	return maps.Clone(entries), nil
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

// ClockOrder represents the result of comparing two vector clocks.
type ClockOrder int

const (
	OrderBefore     ClockOrder = iota - 1 // this happened before other
	OrderConcurrent                       // this and other are concurrent
	OrderAfter                            // this happened after other
	OrderEqual                            // this and other are identical
)

// Cmp compares two vector clocks and returns a typed result.
//
// Unlike Compare (which conflates concurrent and equal), Cmp distinguishes all four cases.
func (vc VectorClock) Cmp(other VectorClock) ClockOrder {
	if vc.Equal(other) {
		return OrderEqual
	}

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
		return OrderBefore
	}

	if greater && !less {
		return OrderAfter
	}

	return OrderConcurrent
}

func (o ClockOrder) String() string {
	switch o {
	case OrderBefore:
		return clockOrderBefore
	case OrderAfter:
		return clockOrderAfter
	case OrderConcurrent:
		return clockOrderConcurrent
	case OrderEqual:
		return "equal"
	default:
		return "unknown"
	}
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
