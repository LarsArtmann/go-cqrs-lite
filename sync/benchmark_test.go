package sync

import (
	"testing"
	"time"
)

func BenchmarkNewVectorClock(b *testing.B) {
	for b.Loop() {
		_ = NewVectorClock()
	}
}

func BenchmarkVectorClock_Increment(b *testing.B) {
	clock := NewVectorClock()
	nodeID := NodeID("bench-node")

	b.ResetTimer()

	for b.Loop() {
		clock.Increment(nodeID)
	}
}

func BenchmarkVectorClock_Merge(b *testing.B) {
	clock1 := NewVectorClockFromMap(map[NodeID]int64{
		"node-a": 5,
		"node-b": 3,
	})
	clock2 := NewVectorClockFromMap(map[NodeID]int64{
		"node-a": 2,
		"node-c": 7,
	})

	b.ResetTimer()

	for b.Loop() {
		clone := clock1.Clone()
		clone.Merge(clock2)
	}
}

func BenchmarkVectorClock_Compare(b *testing.B) {
	clock1 := NewVectorClockFromMap(map[NodeID]int64{
		"node-a": 5,
		"node-b": 3,
	})
	clock2 := NewVectorClockFromMap(map[NodeID]int64{
		"node-a": 5,
		"node-b": 4,
	})

	b.ResetTimer()

	for b.Loop() {
		clock1.Cmp(clock2)
	}
}

func BenchmarkNewLWWResolver(b *testing.B) {
	ts := func(_ struct{}) time.Time { return time.Now() }

	b.ResetTimer()

	for b.Loop() {
		_ = NewLWWResolver(ts)
	}
}
