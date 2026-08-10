package irohengine

import (
	"slices"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

const latencyWindowSize = 512

// SortDurations returns a sorted copy of d (ascending). Shared by transports
// that compute latency percentiles from RTT samples outside the LatencyCollector
// (e.g. loopback, quic transports that maintain their own sample slices).
func SortDurations(d []time.Duration) []time.Duration {
	cp := append([]time.Duration(nil), d...)
	slices.Sort(cp)
	return cp
}

// PercentileIdx returns the array index for the p-th percentile of n elements.
// Clamped to [0, n-1]. Shared by transports that index into sorted samples.
func PercentileIdx(n int, p float64) int {
	idx := int(float64(n-1) * p)
	if idx >= n {
		idx = n - 1
	}

	if idx < 0 {
		idx = 0
	}

	return idx
}

// LatencyCollector records real delivery and convergence times from replication
// traffic. It delegates to two core [metaengine.LatencyTracker] instances
// (ring buffer + incremental EWMA), consolidating the percentile machinery so
// there is one source of truth for latency statistics across the codebase.
type LatencyCollector struct {
	delivery    *metaengine.LatencyTracker
	convergence *metaengine.LatencyTracker
}

func newLatencyCollector() *LatencyCollector {
	return &LatencyCollector{
		delivery:    metaengine.NewLatencyTracker(metaengine.WithTrackerWindow(latencyWindowSize)),
		convergence: metaengine.NewLatencyTracker(metaengine.WithTrackerWindow(latencyWindowSize)),
	}
}

func (lc *LatencyCollector) recordDelivery(d time.Duration) {
	lc.delivery.Record(d)
}

func (lc *LatencyCollector) recordConvergence(d time.Duration) {
	lc.convergence.Record(d)
}

// LatencyStats is a snapshot of measured latency at a point in time.
type LatencyStats struct {
	Samples int
	Mean    time.Duration
	P50     time.Duration
	P95     time.Duration
	P99     time.Duration
	Max     time.Duration
}

func (lc *LatencyCollector) DeliveryStats() LatencyStats {
	return fromCoreStats(lc.delivery.Snapshot())
}

func (lc *LatencyCollector) ConvergenceStats() LatencyStats {
	return fromCoreStats(lc.convergence.Snapshot())
}

// LatencySnapshot is a compact view for the EngineProfile cost model.
type LatencySnapshot struct {
	DeliveryP50    time.Duration
	DeliveryP99    time.Duration
	ConvergenceP99 time.Duration
}

func (lc *LatencyCollector) Snapshot() LatencySnapshot {
	d := lc.delivery.Snapshot()
	c := lc.convergence.Snapshot()

	return LatencySnapshot{
		DeliveryP50:    d.P50,
		DeliveryP99:    d.P99,
		ConvergenceP99: c.P99,
	}
}

// LatencyProvider is an optional interface that transports can implement
// to expose real latency measurements to the engine for Profile() reporting.
type LatencyProvider interface {
	LatencySnapshot() LatencySnapshot
}

// fromCoreStats converts a [metaengine.LatencyStats] (which includes EWMA and
// LastSample) to the local LatencyStats type that the iroh transports expect.
func fromCoreStats(s metaengine.LatencyStats) LatencyStats {
	return LatencyStats{
		Samples: s.Samples,
		Mean:    s.Mean,
		P50:     s.P50,
		P95:     s.P95,
		P99:     s.P99,
		Max:     s.Max,
	}
}
