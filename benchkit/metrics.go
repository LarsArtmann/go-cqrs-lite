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
// LatencyCollector is safe for concurrent use.
type LatencyCollector struct {
	mu      sync.Mutex
	samples []time.Duration
	count   int64
	sumNs   int64 // running sum for mean (nanoseconds)
	maxLen  int
	rng     *rand.Rand
}

// NewLatencyCollector creates a collector that retains at most maxLen samples.
// If maxLen <= 0, defaults to [defaultReservoirSize].
func NewLatencyCollector(maxLen int) *LatencyCollector {
	if maxLen <= 0 {
		maxLen = defaultReservoirSize
	}

	return &LatencyCollector{
		samples: make([]time.Duration, 0, min(maxLen, 1024)),
		maxLen:  maxLen,
		rng:     rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0)),
	}
}

// Record adds a latency sample. If the reservoir is full, the sample replaces
// a random existing entry with probability maxLen/count.
func (lc *LatencyCollector) Record(d time.Duration) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	lc.count++
	lc.sumNs += int64(d)

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
func (lc *LatencyCollector) Stats() LatencyStats {
	lc.mu.Lock()
	count := lc.count
	sumNs := lc.sumNs
	samples := make([]time.Duration, len(lc.samples))
	copy(samples, lc.samples)
	lc.mu.Unlock()

	if count == 0 {
		return LatencyStats{}
	}

	slices.Sort(samples)

	return LatencyStats{
		Count: count,
		P50:   percentile(samples, 50),
		P75:   percentile(samples, 75),
		P90:   percentile(samples, 90),
		P95:   percentile(samples, 95),
		P99:   percentile(samples, 99),
		P100:  samples[len(samples)-1],
		Mean:  time.Duration(sumNs / count),
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
