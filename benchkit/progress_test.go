package benchkit

import (
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// parkWriter lets exactly one Write call park inside the writer and records
// any overlap. It makes both contracts observable deterministically, without
// relying on the race detector:
//
//   - stop() must JOIN the heartbeat: with a write parked, a stop() that only
//     closes the done channel returns while a line is still in flight.
//   - every write must be serialized: a second goroutine reaching Write while
//     the parked call has not returned is an interleaving inside fmt.Fprintf.
type parkWriter struct {
	started chan struct{}
	release chan struct{}

	parkNext atomic.Bool
	once     sync.Once

	inside     atomic.Int32
	overlapped atomic.Bool
	writes     atomic.Int64
}

func newParkWriter() *parkWriter {
	w := &parkWriter{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	w.parkNext.Store(true)

	return w
}

func (w *parkWriter) Write(p []byte) (int, error) {
	if w.inside.Add(1) > 1 {
		w.overlapped.Store(true)
	}

	defer w.inside.Add(-1)

	if w.parkNext.CompareAndSwap(true, false) {
		w.once.Do(func() { close(w.started) })
		<-w.release
	}

	w.writes.Add(1)

	return len(p), nil
}

// TestProgressReporter_StopWaitsForInFlightBeat pins the defect found by the
// 2026-10-23 full-workspace -race sweep: progressReporter.stop() only closed
// the done channel, so a beat() already inside Write could still be running
// after stop() returned. RunRepeated then printed its next "repeat N/M" header
// into the same writer, which the race detector reported as a race inside
// bytes.Buffer (benchkit/repeat.go:83 vs benchkit/progress.go:77).
func TestProgressReporter_StopWaitsForInFlightBeat(t *testing.T) {
	w := newParkWriter()

	r := newProgressReporter(w, time.Microsecond, "test", 1)
	r.beginPhase(1, "write")
	r.startHeartbeat()

	waitParked(t, w)

	stopped := make(chan struct{})

	go func() {
		r.stop()
		close(stopped)
	}()

	// stop() must block on the in-flight beat. Without the join it returns
	// immediately, releasing the writer behind the caller's back.
	select {
	case <-stopped:
		t.Fatal("stop returned while a progress write was still in flight")
	case <-time.After(200 * time.Millisecond):
	}

	before := w.writes.Load()

	close(w.release)

	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("stop did not return after the in-flight write completed")
	}

	if after := w.writes.Load(); after != before+1 {
		t.Fatalf("writes after stop = %d, want %d", after, before+1)
	}

	// No further writes may appear once stop returns.
	time.Sleep(20 * time.Millisecond)

	if after := w.writes.Load(); after != before+1 {
		t.Fatalf("progress wrote after stop returned: writes = %d, want %d", after, before+1)
	}
}

// TestProgressReporter_ConcurrentBeatAndEndPhaseSerialized proves the writer is
// the serialized resource: with a heartbeat parked inside Write, endPhase must
// wait for it rather than interleaving inside the same fmt.Fprintf.
func TestProgressReporter_ConcurrentBeatAndEndPhaseSerialized(t *testing.T) {
	w := newParkWriter()

	r := newProgressReporter(w, time.Microsecond, "test", 4)
	r.beginPhase(1, "write")
	r.startHeartbeat()
	t.Cleanup(r.stop)

	waitParked(t, w)

	done := make(chan struct{})

	go func() {
		r.endPhase("write", time.Millisecond)
		close(done)
	}()

	// Give endPhase time to reach the writer. With the write lock it parks on
	// the mutex instead, so it cannot overlap.
	time.Sleep(200 * time.Millisecond)

	close(w.release)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("endPhase did not return")
	}

	if w.overlapped.Load() {
		t.Fatal("endPhase entered the writer while a heartbeat write was in flight")
	}
}

func waitParked(t *testing.T, w *parkWriter) {
	t.Helper()

	select {
	case <-w.started:
	case <-time.After(5 * time.Second):
		t.Fatal("heartbeat never entered the writer")
	}
}

var _ io.Writer = (*parkWriter)(nil)
