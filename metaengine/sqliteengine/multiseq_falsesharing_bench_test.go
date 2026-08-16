package sqliteengine

import (
	"sync"
	"sync/atomic"
	"testing"
)

// Measure-then-pad protocol (docs/BENCHMARKS.md): pad ONLY if the padded
// variant wins by more than 10% under contended load. Measured 2026-08-16:
// padded won 2.2-2.8x, so multiSeqCounter shipped with the pad.
//
// multiSeqCounter is allocated per multimap collection (sync.Map
// LoadOrStore). Go's small-object allocator packs 32-byte objects 16-per-512B
// span, so two hot collections' counters can share a cache line. The [2]T
// arrays below guarantee adjacency — the worst-case packing — for both the
// unpadded control and the padded production layout.

// unpaddedMultiSeq preserves the PRE-PAD layout (32 bytes) as the negative
// control: if the trailing pad is ever removed from multiSeqCounter, the
// Padded bench degrades to these numbers.
type unpaddedMultiSeq struct {
	once    sync.Once
	counter atomic.Int64
	initErr error
}

func noop() {}

// benchTwoCounters splits the parallel goroutines evenly between incA and
// incB so both counters burn a full core set at high -cpu values.
func benchTwoCounters(b *testing.B, incA, incB func()) {
	var role atomic.Int64

	b.RunParallel(func(pb *testing.PB) {
		inc := incA
		if role.Add(1)%2 == 0 {
			inc = incB
		}

		for pb.Next() {
			inc()
		}
	})
}

func BenchmarkMultiSeqCounterUnpadded(b *testing.B) {
	counters := &[2]unpaddedMultiSeq{}
	counters[0].once.Do(noop)
	counters[1].once.Do(noop)

	benchTwoCounters(b,
		func() { counters[0].once.Do(noop); counters[0].counter.Add(1) },
		func() { counters[1].once.Do(noop); counters[1].counter.Add(1) },
	)
}

// BenchmarkMultiSeqCounterPadded uses the production multiSeqCounter (padded
// to the 128-byte size class since 2026-08-16).
func BenchmarkMultiSeqCounterPadded(b *testing.B) {
	counters := &[2]multiSeqCounter{}
	counters[0].once.Do(noop)
	counters[1].once.Do(noop)

	benchTwoCounters(b,
		func() { counters[0].once.Do(noop); counters[0].counter.Add(1) },
		func() { counters[1].once.Do(noop); counters[1].counter.Add(1) },
	)
}
