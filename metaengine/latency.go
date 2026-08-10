package metaengine

import (
	"slices"
	"sync"
	"time"
)

// Live-latency measurement for the cost model.
//
// Design report: docs/planning/METAENGINE-LIVE-LATENCY-MODEL.md.
//
// NetworkRTT and per-op latency are RUNTIME observations, not compile-time
// facts. LatencyTracker holds a sliding window of samples with an incremental
// EWMA so estimates respond to recent conditions while dampening jitter. Stats
// (mean/percentiles) are computed on demand from the window. The tracker is
// pure Go with zero external dependencies, so it stays in the budget-free core.

const (
	defaultTrackerWindow = 512
	defaultStaleAfter    = 30 * time.Second
	// defaultTrackerAlpha controls EWMA responsiveness: the weight given to each
	// new sample. 0.1 reaches ~90% of a step change in ~22 samples, which at a
	// 1s probe interval means a routing-relevant shift shows up in ~20s while a
	// single jitter spike moves the estimate by at most 10%. See risk #4 in the
	// design report (EWMA can oscillate; P50/P95 are exposed for diagnostics).
	defaultTrackerAlpha = 0.1
)

// TrackerOption configures a LatencyTracker.
type TrackerOption func(*trackerSettings)

type trackerSettings struct {
	window     int
	alpha      float64
	staleAfter time.Duration
	sink       StatSink
	name       string
}

// WithTrackerWindow sets the sample window size (default 512).
func WithTrackerWindow(n int) TrackerOption {
	return func(s *trackerSettings) {
		if n > 0 {
			s.window = n
		}
	}
}

// WithTrackerAlpha sets the EWMA smoothing factor in (0,1]. Higher means more
// responsive to recent samples. Default 0.1.
func WithTrackerAlpha(a float64) TrackerOption {
	return func(s *trackerSettings) {
		if a > 0 && a <= 1 {
			s.alpha = a
		}
	}
}

// WithStaleAfter sets how long after the last sample the tracker is considered
// stale. Zero disables staleness (always fresh once it has a sample). A tracker
// with no samples is always stale.
func WithStaleAfter(d time.Duration) TrackerOption {
	return func(s *trackerSettings) { s.staleAfter = d }
}

// WithTrackerSink wires a StatSink that receives every recorded sample. This is
// the open ingress path (ADR-0093 follow-up): external engines and probes push
// measurements through the sink without depending on probe internals.
func WithTrackerSink(name string, sink StatSink) TrackerOption {
	return func(s *trackerSettings) {
		s.name = name
		s.sink = sink
	}
}

// LatencyTracker maintains a sliding window of latency samples with an
// exponential-decay (EWMA) estimate. Record is O(1); Snapshot copies and sorts
// the window (O(N log N)) — call it at plan time or from GetEngineStats, not on
// the hot read path.
type LatencyTracker struct {
	mu         sync.Mutex
	samples    []time.Duration // ring buffer of capacity window
	head       int             // next write position
	count      int             // valid samples (<= window)
	ewma       float64         // nanoseconds, incremental
	hasEWMA    bool
	lastAt     time.Time
	alpha      float64
	staleAfter time.Duration
	sink       StatSink
	name       string
}

// NewLatencyTracker creates a tracker with the given options applied over
// sensible defaults.
func NewLatencyTracker(opts ...TrackerOption) *LatencyTracker {
	s := trackerSettings{
		window:     defaultTrackerWindow,
		alpha:      defaultTrackerAlpha,
		staleAfter: defaultStaleAfter,
	}
	for _, opt := range opts {
		opt(&s)
	}

	return &LatencyTracker{
		samples:    make([]time.Duration, s.window),
		alpha:      s.alpha,
		staleAfter: s.staleAfter,
		sink:       s.sink,
		name:       s.name,
	}
}

// Record appends a latency sample and updates the EWMA incrementally. Safe for
// concurrent use. If a StatSink is configured, the sample is forwarded outside
// the lock.
func (t *LatencyTracker) Record(d time.Duration) {
	now := time.Now()

	t.mu.Lock()
	t.samples[t.head] = d
	t.head = (t.head + 1) % len(t.samples)
	if t.count < len(t.samples) {
		t.count++
	}

	ns := float64(d.Nanoseconds())
	if !t.hasEWMA {
		t.ewma = ns
		t.hasEWMA = true
	} else {
		t.ewma = t.alpha*ns + (1-t.alpha)*t.ewma
	}

	t.lastAt = now
	sink := t.sink
	name := t.name
	t.mu.Unlock()

	if sink != nil {
		sink.ReportSample(name, LatencySample{Kind: SampleRTT, Value: d, At: now})
	}
}

// LatencyStats is a point-in-time view of a tracker's window.
type LatencyStats struct {
	Samples    int
	Mean       time.Duration
	P50        time.Duration
	P95        time.Duration
	P99        time.Duration
	Max        time.Duration
	EWMA       time.Duration
	LastSample time.Time
}

// Snapshot computes stats from the current window. An empty tracker returns a
// zero-value LatencyStats (Samples == 0).
func (t *LatencyTracker) Snapshot() LatencyStats {
	t.mu.Lock()
	window := t.orderedSamples()
	ewma := t.ewma
	hasEWMA := t.hasEWMA
	lastAt := t.lastAt
	t.mu.Unlock()

	n := len(window)
	if n == 0 {
		return LatencyStats{LastSample: lastAt}
	}

	sorted := slices.Clone(window)
	slices.Sort(sorted)

	var sum time.Duration
	for _, d := range sorted {
		sum += d
	}

	stats := LatencyStats{
		Samples:    n,
		Mean:       sum / time.Duration(n),
		P50:        percentileDur(sorted, 0.50),
		P95:        percentileDur(sorted, 0.95),
		P99:        percentileDur(sorted, 0.99),
		Max:        sorted[n-1],
		LastSample: lastAt,
	}
	if hasEWMA {
		stats.EWMA = time.Duration(ewma)
	}

	return stats
}

// Fresh reports whether the tracker has a usable current estimate: at least one
// sample AND the last sample is within the stale-after window. A zero
// staleAfter means "never stale once sampled."
func (t *LatencyTracker) Fresh() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.count == 0 {
		return false
	}

	if t.staleAfter <= 0 {
		return true
	}

	return time.Since(t.lastAt) <= t.staleAfter
}

// Live returns the current snapshot and whether it is fresh enough to use for
// routing. When the second return is false the caller MUST fall back to the
// declared prior and label the estimate stale — never silently route on a stale
// number.
func (t *LatencyTracker) Live() (LatencyStats, bool) {
	stats := t.Snapshot()
	if stats.Samples == 0 {
		return stats, false
	}

	t.mu.Lock()
	stale := t.staleAfter > 0 && time.Since(t.lastAt) > t.staleAfter
	t.mu.Unlock()

	return stats, !stale
}

// orderedSamples returns the valid window in chronological order. Caller holds
// the lock.
func (t *LatencyTracker) orderedSamples() []time.Duration {
	if t.count == 0 {
		return nil
	}

	if t.count < len(t.samples) {
		// Not yet full: samples [0, count) are in order, head == count.
		return slices.Clone(t.samples[:t.count])
	}

	// Full ring: oldest is at head, newest is at head-1.
	out := make([]time.Duration, 0, t.count)
	out = append(out, t.samples[t.head:]...)
	out = append(out, t.samples[:t.head]...)

	return out
}

func percentileDur(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}

	idx := int(float64(len(sorted)-1) * p)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}

	if idx < 0 {
		idx = 0
	}

	return sorted[idx]
}
