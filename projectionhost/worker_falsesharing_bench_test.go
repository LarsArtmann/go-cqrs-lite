package projectionhost

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Measure-then-pad protocol (docs/BENCHMARKS.md): pad ONLY if the padded
// variant wins by more than 10% under contended load. These benches model the
// production worker-counter access pattern: ONE writer goroutine (the worker
// event loop touches processed + lastProcessedNs per event) while background
// readers spin snapshot() — the pessimistic bound of a metrics scrape, which
// in production is rare.
//
// Worker counters are single-writer: within one worker the same core writes
// all four counters, so they cannot false-share with each other. The only
// cross-core traffic is the reader, so the question is whether isolating the
// counters from the read path pays.

// paddedWorkerCounters mirrors worker's state/counter block with a 64-byte
// pad between the mutex-guarded fields and the counters. Its snapshot method
// does the same work as worker.snapshot so the reader load is isomorphic.
type paddedWorkerCounters struct {
	mu    sync.RWMutex
	state WorkerState

	_ [64]byte

	processed       atomic.Int64
	errors          atomic.Int64
	restarts        atomic.Int64
	lastProcessedNs atomic.Int64
}

func (p *paddedWorkerCounters) snapshot() WorkerState {
	p.mu.RLock()
	defer p.mu.RUnlock()

	s := p.state
	s.Processed = p.processed.Load()
	s.Errors = p.errors.Load()
	s.Restarts = int(p.restarts.Load())

	if nanos := p.lastProcessedNs.Load(); nanos != 0 {
		s.Lag = time.Since(time.Unix(0, nanos))
	}

	return s
}

// spinWhile runs read in GOMAXPROCS-1 background goroutines until the benched
// write loop finishes, then waits for them.
func spinWhile(b *testing.B, read func()) {
	var stop atomic.Bool

	var wg sync.WaitGroup

	for range max(runtime.GOMAXPROCS(0)-1, 1) {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for !stop.Load() {
				read()
			}
		}()
	}

	b.Cleanup(func() {
		stop.Store(true)
		wg.Wait()
	})
}

func BenchmarkWorkerCountersAdjacent(b *testing.B) {
	w := &worker{}
	spinWhile(b, func() { _ = w.snapshot() })

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		w.processed.Add(1)
		w.lastProcessedNs.Store(time.Now().UnixNano())
	}
}

func BenchmarkWorkerCountersPadded(b *testing.B) {
	p := &paddedWorkerCounters{}
	spinWhile(b, func() { _ = p.snapshot() })

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		p.processed.Add(1)
		p.lastProcessedNs.Store(time.Now().UnixNano())
	}
}
