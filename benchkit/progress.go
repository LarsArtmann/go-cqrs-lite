package benchkit

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// progressReporter writes debounced progress updates during a benchmark run.
// Phase transitions are always emitted; in-phase heartbeat updates fire on a
// ticker so the user can see that a long phase (e.g. write with 5M events) is
// still running and how long it has taken.
type progressReporter struct {
	w        io.Writer
	interval time.Duration
	backend  string
	total    int

	mu    sync.RWMutex
	num   int
	phase string
	start time.Time

	// writeMu serializes writes to w. beat() runs on the heartbeat goroutine
	// while endPhase() runs on the caller's goroutine, so the writer itself
	// needs its own lock: the state mutex only guards num/phase/start.
	writeMu sync.Mutex

	heart    *time.Ticker
	done     chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

func newProgressReporter(
	w io.Writer, interval time.Duration, backend string, totalPhases int,
) *progressReporter {
	if interval <= 0 {
		interval = 5 * time.Second
	}

	return &progressReporter{
		w:        w,
		interval: interval,
		backend:  backend,
		total:    totalPhases,
	}
}

// start launches the heartbeat goroutine. Safe to call on a nil receiver or
// when no writer is configured — both are no-ops.
func (p *progressReporter) startHeartbeat() {
	if p == nil || p.w == nil {
		return
	}

	p.heart = time.NewTicker(p.interval)
	p.done = make(chan struct{})

	p.wg.Add(1)

	go func() {
		defer p.wg.Done()

		p.heartbeat()
	}()
}

func (p *progressReporter) heartbeat() {
	for {
		select {
		case <-p.heart.C:
			p.beat()
		case <-p.done:
			p.heart.Stop()

			return
		}
	}
}

func (p *progressReporter) beat() {
	p.mu.RLock()
	num, phase := p.num, p.phase
	elapsed := time.Since(p.start).Round(time.Second)
	p.mu.RUnlock()

	p.printf("  %s | %d/%d %s | %s elapsed\n", p.backend, num, p.total, phase, elapsed)
}

// printf writes one line to the progress writer under the write lock. Every
// path that writes to p.w must go through it, otherwise the heartbeat
// goroutine and the caller's goroutine interleave inside a single
// fmt.Fprintf (surfaced by a full-workspace -race sweep, 2026-10-23:
// bytes.Buffer growth raced between beat() and RunRepeated's repeat header).
func (p *progressReporter) printf(format string, args ...any) {
	p.writeMu.Lock()
	defer p.writeMu.Unlock()

	fmt.Fprintf(p.w, format, args...)
}

// beginPhase records the current phase and starts the timer.
// The completion line is printed by endPhase — emitting both "started" and
// "done" doubles the progress noise (2 lines per phase) without adding value.
func (p *progressReporter) beginPhase(num int, phase string) {
	if p == nil {
		return
	}

	p.mu.Lock()
	p.num = num
	p.phase = phase
	p.start = time.Now()
	p.mu.Unlock()
}

// endPhase prints a completion line for the phase that just finished.
func (p *progressReporter) endPhase(phase string, d time.Duration) {
	if p == nil || p.w == nil {
		return
	}

	p.mu.RLock()
	num := p.num
	p.mu.RUnlock()

	p.printf("  %s | %d/%d %s | done (%s)\n",
		p.backend, num, p.total, phase, d.Round(time.Millisecond))
}

// stop terminates the heartbeat goroutine and WAITS for it to exit. Idempotent.
// The done channel is closed exactly once and never reassigned — heartbeat()
// reads it in a select, so mutating it here would race with that read.
//
// The join is load-bearing: without it a heartbeat write could still be in
// flight after the run returned, racing with whatever the caller prints next
// (RunRepeated's per-repeat header) and with the following run's progress
// lines. Closing done only signals; wg.Wait makes "stopped" true.
func (p *progressReporter) stop() {
	if p == nil || p.done == nil {
		return
	}

	p.stopOnce.Do(func() {
		close(p.done)
		p.wg.Wait()
	})
}
