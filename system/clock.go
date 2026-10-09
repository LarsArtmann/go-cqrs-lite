package system

import (
	"sync"
	"time"
)

// Clock is the System's time source seam (ADR-0153 D4): everything that
// needs "now" reads it from the composition root instead of the wall clock,
// so tests can freeze and advance time deterministically.
type Clock interface {
	Now() time.Time
}

// RealClock reads the wall clock. It is the default Clock.
type RealClock struct{}

// Now returns the current wall-clock time.
func (RealClock) Now() time.Time { return time.Now() }

// ManualClock is a controllable Clock for tests: start it at a fixed
// instant, then Advance it as the scenario demands. Safe for concurrent
// use. The go-cqrs-lite testing harness (systemscenario) boots every
// scenario on one.
type ManualClock struct {
	mu  sync.Mutex
	now time.Time
}

// NewManualClock creates a ManualClock frozen at start.
func NewManualClock(start time.Time) *ManualClock {
	return &ManualClock{now: start}
}

// Now returns the clock's current (frozen) instant.
func (c *ManualClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.now
}

// Advance moves the clock forward by d. Negative durations move it back.
func (c *ManualClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.now = c.now.Add(d)
}

// Set moves the clock to the given instant.
func (c *ManualClock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.now = t
}

// systemOptions collects New's variadic options.
type systemOptions struct {
	clock Clock
}

// Option configures [New]. Options are additive (the two-config model D11
// is unchanged); system is experimental, so new options do not break the
// v4 stability surface.
type Option func(*systemOptions)

// WithClock sets the System's time source (ADR-0153 D4). Default:
// [RealClock]. Consumers read it via [System.Clock] — e.g. a Timers closure
// computing deterministic FireAt values and wiring
// scheduling.WithClock(sys.Clock().Now).
func WithClock(c Clock) Option {
	return func(o *systemOptions) {
		if c != nil {
			o.clock = c
		}
	}
}

// Clock returns the System's time source. Defaults to [RealClock]; tests
// inject a [ManualClock] via [WithClock].
func (s *System) Clock() Clock {
	if s.clock == nil {
		return RealClock{}
	}

	return s.clock
}
