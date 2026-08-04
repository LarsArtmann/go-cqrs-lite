package irohengine

import (
	"slices"
	"sync"
	"time"
)

const latencyWindowSize = 512

// SortDurations returns a sorted copy of d (ascending). Shared by transports
// that compute latency percentiles from RTT samples.
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
// traffic. All stats are computed from actual measurements — no hardcoded values.
type LatencyCollector struct {
	mu           sync.Mutex
	deliveries   []time.Duration
	convergences []time.Duration
}

func newLatencyCollector() *LatencyCollector {
	return &LatencyCollector{
		deliveries:   make([]time.Duration, 0, latencyWindowSize),
		convergences: make([]time.Duration, 0, latencyWindowSize),
	}
}

func (lc *LatencyCollector) recordDelivery(d time.Duration) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.deliveries = append(lc.deliveries, d)
	if len(lc.deliveries) > latencyWindowSize {
		lc.deliveries = lc.deliveries[len(lc.deliveries)-latencyWindowSize:]
	}
}

func (lc *LatencyCollector) recordConvergence(d time.Duration) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.convergences = append(lc.convergences, d)
	if len(lc.convergences) > latencyWindowSize {
		lc.convergences = lc.convergences[len(lc.convergences)-latencyWindowSize:]
	}
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
	lc.mu.Lock()
	s := append([]time.Duration(nil), lc.deliveries...)
	lc.mu.Unlock()
	return computeStats(s)
}

func (lc *LatencyCollector) ConvergenceStats() LatencyStats {
	lc.mu.Lock()
	s := append([]time.Duration(nil), lc.convergences...)
	lc.mu.Unlock()
	return computeStats(s)
}

// LatencySnapshot is a compact view for the EngineProfile cost model.
type LatencySnapshot struct {
	DeliveryP50    time.Duration
	DeliveryP99    time.Duration
	ConvergenceP99 time.Duration
}

func (lc *LatencyCollector) Snapshot() LatencySnapshot {
	d := lc.DeliveryStats()
	c := lc.ConvergenceStats()
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

func computeStats(samples []time.Duration) LatencyStats {
	n := len(samples)
	if n == 0 {
		return LatencyStats{}
	}
	sorted := SortDurations(samples)

	var sum time.Duration
	for _, d := range sorted {
		sum += d
	}

	return LatencyStats{
		Samples: n,
		Mean:    sum / time.Duration(n),
		P50:     percentile(sorted, 0.50),
		P95:     percentile(sorted, 0.95),
		P99:     percentile(sorted, 0.99),
		Max:     sorted[n-1],
	}
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}

	return sorted[PercentileIdx(len(sorted), p)]
}
