package metaengine

import (
	"sync"
	"sync/atomic"
	"testing"
)

// Measure-then-pad protocol (docs/BENCHMARKS.md): pad ONLY if the padded
// variant wins by more than 10% under contended load.
//
// SSEReplay.record touches BOTH seq (atomic add, before the lock) and the
// mutex-guarded ring fields on every call, so concurrent recorders pull both
// cache lines regardless. This bench checks whether separating seq from the
// guarded fields pays at all.

// paddedReplay mirrors SSEReplay[int] with seq isolated from the
// mutex-guarded fields by a 64-byte pad.
type paddedReplay struct {
	mu      sync.Mutex
	entries []seqEntry[int]
	cap     int
	head    int
	count   int

	_ [64]byte

	seq atomic.Uint64
}

func newPaddedReplay(capacity int) *paddedReplay {
	return &paddedReplay{entries: make([]seqEntry[int], capacity), cap: capacity}
}

func (r *paddedReplay) record(value int) uint64 {
	seq := r.seq.Add(1)

	r.mu.Lock()
	//art-dupl:accept intentional isomorphic mirror of SSEReplay.record — false-sharing A/B bench
	r.entries[r.head] = seqEntry[int]{seq: seq, value: value}
	r.head = (r.head + 1) % r.cap
	if r.count < r.cap {
		r.count++
	}
	r.mu.Unlock()

	return seq
}

func benchReplay(b *testing.B, record func(int) uint64) {
	b.ReportAllocs()
	b.ResetTimer()

	var n atomic.Int64

	b.RunParallel(func(pb *testing.PB) {
		i := int(n.Add(1))

		for pb.Next() {
			record(i)
		}
	})
}

func BenchmarkSSEReplaySeqAdjacent(b *testing.B) {
	replay := NewSSEReplay[int](64)
	benchReplay(b, replay.record)
}

func BenchmarkSSEReplaySeqPadded(b *testing.B) {
	replay := newPaddedReplay(64)
	benchReplay(b, replay.record)
}
