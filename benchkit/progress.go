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

	heart *time.Ticker
	done  chan struct{}
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

	go p.heartbeat()
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

	fmt.Fprintf(p.w, "  %s | %d/%d %s | %s elapsed\n", p.backend, num, p.total, phase, elapsed)
}

// beginPhase records the current phase and prints a start line.
func (p *progressReporter) beginPhase(num int, phase string) {
	if p == nil {
		return
	}

	p.mu.Lock()
	p.num = num
	p.phase = phase
	p.start = time.Now()
	p.mu.Unlock()

	if p.w != nil {
		fmt.Fprintf(p.w, "  %s | %d/%d %s | started\n", p.backend, num, p.total, phase)
	}
}

// endPhase prints a completion line for the phase that just finished.
func (p *progressReporter) endPhase(phase string, d time.Duration) {
	if p == nil || p.w == nil {
		return
	}

	p.mu.RLock()
	num := p.num
	p.mu.RUnlock()

	fmt.Fprintf(p.w, "  %s | %d/%d %s | done (%s)\n",
		p.backend, num, p.total, phase, d.Round(time.Millisecond))
}

// stop terminates the heartbeat goroutine. Idempotent.
func (p *progressReporter) stop() {
	if p == nil || p.done == nil {
		return
	}

	close(p.done)
	p.done = nil
}
