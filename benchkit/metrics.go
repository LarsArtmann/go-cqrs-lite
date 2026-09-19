package benchkit

import (
	"math/rand/v2"
	"runtime"
	"slices"
	"sync"
	"time"
)

// defaultReservoirSize is the maximum number of latency samples kept in memory
// when the total count exceeds the reservoir threshold. This bounds memory to
// ~80KB per collector regardless of workload size.
const defaultReservoirSize = 10_000

// LatencyCollector records latency samples and computes percentile statistics.
//
// For small workloads (< reservoirSize samples), all samples are retained.
// For large workloads, Algorithm R reservoir sampling keeps a fixed-size
// uniform sample so percentile estimates remain accurate with O(1) memory.
//
// Two statistics are exact even after the reservoir saturates: the mean
// (accumulated as a running sum), the minimum, and the maximum (both tracked
// on every Record). The remaining percentiles are estimates computed over the
// retained sample, and a tail spike that never entered the reservoir cannot
// inflate them.
//
// LatencyCollector is safe for concurrent use.
type LatencyCollector struct {
	mu          sync.Mutex
	samples     []time.Duration
	count       int64
	sumNs       int64 // running sum for mean (nanoseconds)
	min         time.Duration
	max         time.Duration
	maxLen      int
	interpolate bool
	rng         *rand.Rand
}

// CollectorOption adjusts how a [LatencyCollector] computes its statistics.
type CollectorOption func(*LatencyCollector)

// WithInterpolatedPercentiles switches P50–P99 from the default nearest-rank
// method to linear interpolation between neighboring samples. Nearest-rank on
// a small sample is coarse — with 5 samples the P50 is the 3rd-smallest and
// the P99 collapses onto the maximum — while interpolation reads the value
// between the surrounding samples. Turn it on for small-n runs (dev profiles,
// smoke tests); keep the default for large runs where the reservoir already
// smooths the estimates.
func WithInterpolatedPercentiles() CollectorOption {
	return func(lc *LatencyCollector) { lc.interpolate = true }
}

// NewLatencyCollector creates a collector that retains at most maxLen samples.
// If maxLen <= 0, defaults to [defaultReservoirSize].
func NewLatencyCollector(maxLen int) *LatencyCollector {
	return NewLatencyCollectorWithOptions(maxLen)
}

// NewLatencyCollectorWithOptions creates a collector with a sample limit and
// statistic behavior options (see [WithInterpolatedPercentiles]).
func NewLatencyCollectorWithOptions(maxLen int, opts ...CollectorOption) *LatencyCollector {
	if maxLen <= 0 {
		maxLen = defaultReservoirSize
	}

	lc := &LatencyCollector{
		samples: make([]time.Duration, 0, min(maxLen, 1024)),
		maxLen:  maxLen,
		rng:     rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0)),
	}

	for _, opt := range opts {
		opt(lc)
	}

	return lc
}

// Record adds a latency sample. If the reservoir is full, the sample replaces
// a random existing entry with probability maxLen/count.
func (lc *LatencyCollector) Record(d time.Duration) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	lc.count++
	lc.sumNs += int64(d)

	if lc.min == 0 || d < lc.min {
		lc.min = d
	}

	if d > lc.max {
		lc.max = d
	}

	if len(lc.samples) < lc.maxLen {
		lc.samples = append(lc.samples, d)

		return
	}

	idx := lc.rng.Int64N(lc.count)
	if idx < int64(lc.maxLen) {
		lc.samples[idx] = d
	}
}

// Stats computes percentile statistics from the collected samples.
// Returns a zero-valued [LatencyStats] if no samples were recorded.
//
// P100 is the exact maximum observed, Min the exact minimum; both are tracked
// per Record rather than sampled, so reservoir eviction cannot lose them.
// P50–P99 are estimates over the retained sample — nearest-rank by default,
// linearly interpolated when the collector was built with
// [WithInterpolatedPercentiles].
func (lc *LatencyCollector) Stats() LatencyStats {
	lc.mu.Lock()
	count := lc.count
	sumNs := lc.sumNs
	minLatency := lc.min
	maxLatency := lc.max
	interpolate := lc.interpolate
	samples := make([]time.Duration, len(lc.samples))
	copy(samples, lc.samples)
	lc.mu.Unlock()

	if count == 0 {
		return LatencyStats{}
	}

	slices.Sort(samples)

	pct := percentile
	if interpolate {
		pct = percentileInterpolated
	}

	return LatencyStats{
		Count: count,
		P50:   pct(samples, 50),
		P75:   pct(samples, 75),
		P90:   pct(samples, 90),
		P95:   pct(samples, 95),
		P99:   pct(samples, 99),
		P100:  maxLatency,
		Mean:  time.Duration(sumNs / count),
		Min:   minLatency,
	}
}

// percentile returns the p-th percentile from a sorted slice.
// Uses nearest-rank method: index = ceil(p/100 * n) - 1.
func percentile(sorted []time.Duration, p int) time.Duration {
	n := len(sorted)
	if n == 0 {
		return 0
	}

	idx := (n*p + 99) / 100 // ceil(p/100 * n) with integer math
	if idx >= n {
		idx = n - 1
	}

	return sorted[idx]
}

// percentileInterpolated returns the p-th percentile from a sorted slice by
// linear interpolation between the two surrounding samples (rank
// (p/100)*(n-1)). For small n this is far smoother than nearest-rank, whose
// P50 sits exactly on a sample and whose P99 collapses onto the maximum.
func percentileInterpolated(sorted []time.Duration, p int) time.Duration {
	n := len(sorted)
	if n == 0 {
		return 0
	}

	if n == 1 {
		return sorted[0]
	}

	rank := float64(p) / 100 * float64(n-1)
	lower := int(rank)

	if lower >= n-1 {
		return sorted[n-1]
	}

	frac := rank - float64(lower)
	span := float64(sorted[lower+1] - sorted[lower])

	return sorted[lower] + time.Duration(frac*span)
}

// ── Resource sampling ──

// memSnapshot captures Go heap metrics at a point in time.
type memSnapshot struct {
	heapAlloc  uint64
	totalAlloc uint64
	numGC      uint32
	pauseNs    uint64
}

func readMemStats() memSnapshot {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return memSnapshot{
		heapAlloc:  m.HeapAlloc,
		totalAlloc: m.TotalAlloc,
		numGC:      m.NumGC,
		pauseNs:    m.PauseTotalNs,
	}
}

// resourceSampler polls memory usage in a background goroutine and tracks
// the peak heap allocation. Used during the write phase to detect memory
// pressure under load.
type resourceSampler struct {
	mu       sync.Mutex
	peak     uint64
	baseline memSnapshot
	stop     chan struct{}
}

func newResourceSampler() *resourceSampler {
	rs := &resourceSampler{
		baseline: readMemStats(),
		stop:     make(chan struct{}),
	}
	rs.peak = rs.baseline.heapAlloc

	return rs
}

func (rs *resourceSampler) start() {
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-rs.stop:
				return
			case <-ticker.C:
				m := readMemStats()

				rs.mu.Lock()
				if m.heapAlloc > rs.peak {
					rs.peak = m.heapAlloc
				}
				rs.mu.Unlock()
			}
		}
	}()
}

func (rs *resourceSampler) stopAndSnapshot() (peak uint64, baseline memSnapshot) {
	rs.mu.Lock()
	peak = rs.peak
	baseline = rs.baseline
	rs.mu.Unlock()

	close(rs.stop)

	return peak, baseline
}

// cpuTime returns the process CPU time (user + sys) in nanoseconds.
// The platform-specific implementation lives in cpu_unix.go / cpu_other.go.
func cpuTime() uint64 {
	return cpuTimeProc()
}

// gcMetrics computes GC pause statistics between two runtime.MemStats snapshots.
// Returns count, total pause, max pause, and mean pause for all GC cycles
// that occurred between the baseline and final snapshots.
type gcMetricsResult struct {
	Count      int
	TotalPause time.Duration
	MaxPause   time.Duration
	MeanPause  time.Duration
}

func computeGCMetrics(baseline, final runtime.MemStats) gcMetricsResult {
	gcCount := int(final.NumGC - baseline.NumGC)
	if gcCount <= 0 {
		return gcMetricsResult{}
	}

	totalPauseNs := final.PauseTotalNs - baseline.PauseTotalNs

	// Scan the PauseNs circular buffer for entries between baseline.NumGC
	// and final.NumGC. The buffer has 256 entries indexed by (NumGC-1-i) % 256.
	var maxPauseNs uint64

	for i := range uint32(gcCount) {
		idx := (final.NumGC - 1 - i) % 256
		p := final.PauseNs[idx]

		if p > maxPauseNs {
			maxPauseNs = p
		}
	}

	meanPauseNs := uint64(0)
	if gcCount > 0 {
		meanPauseNs = totalPauseNs / uint64(gcCount)
	}

	return gcMetricsResult{
		Count:      gcCount,
		TotalPause: time.Duration(totalPauseNs),
		MaxPause:   time.Duration(maxPauseNs),
		MeanPause:  time.Duration(meanPauseNs),
	}
}
