package benchkit

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"strings"
	"time"
)

// SoakConfig configures a soak test — a sustained benchmark that runs the
// workload repeatedly for a fixed duration to detect memory leaks and
// performance degradation across iterations.
type SoakConfig struct {
	// Duration is the total wall-clock time to run the soak test. The loop
	// stops as soon as the deadline is reached, even mid-iteration.
	Duration time.Duration

	// ReportInterval controls how often a one-line progress summary is written
	// to ProgressWriter. Zero means no intermediate reports (final summary only).
	ReportInterval time.Duration

	// ProgressWriter receives intermediate progress lines. nil means no
	// intermediate output.
	ProgressWriter io.Writer

	// Config is the base benchmark config for each iteration. Each iteration
	// calls factory() fresh, so use a small profile (e.g. ProfileDev) for fast
	// iterations and more data points.
	Config Config
}

// SoakSample captures one iteration's headline metrics.
type SoakSample struct {
	Iteration   int           `json:"iteration"`
	Duration    time.Duration `json:"duration"`
	Throughput  float64       `json:"throughput"` // events/sec
	WriteP50    time.Duration `json:"writeP50"`   // write latency p50
	WriteP99    time.Duration `json:"writeP99"`   // write latency p99
	LoadP50     time.Duration `json:"loadP50"`    // load latency p50
	HeapBytes   uint64        `json:"heapBytes"`  // heap after GC
	TotalEvents int           `json:"totalEvents"`

	// New-phase P99 latencies (zero-valued when the corresponding phase is
	// skipped or the bundle lacks the required capabilities).
	JourneyP99   time.Duration `json:"journeyP99,omitempty"`   // journey round-trip
	QueryHitP99  time.Duration `json:"queryHitP99,omitempty"`  // query dispatch hit
	CacheHitP99  time.Duration `json:"cacheHitP99,omitempty"`  // cache hit (decider)
}

// SoakResult holds the outcome of a soak test, including per-iteration samples
// and trend analysis for leak and degradation detection.
type SoakResult struct {
	Backend    string        `json:"backend"`
	Duration   time.Duration `json:"duration"`
	Iterations int           `json:"iterations"`
	Samples    []SoakSample  `json:"samples"`

	// HeapGrowthBytes = last sample heap − first sample heap. Positive values
	// indicate a possible memory leak.
	HeapGrowthBytes uint64 `json:"heapGrowthBytes"`

	// HeapLeakRate is HeapGrowthBytes divided by the number of iterations
	// (bytes/iteration). A near-zero rate means no cyclic leak.
	HeapLeakRate float64 `json:"heapLeakRate"`

	// ThroughputDriftPct is the percentage change in throughput from the first
	// sample to the last. Negative values indicate degradation.
	ThroughputDriftPct float64 `json:"throughputDriftPct"`

	// WriteP99DriftPct is the percentage change in write P99 latency from the
	// first sample to the last. Positive values indicate latency degradation.
	WriteP99DriftPct float64 `json:"writeP99DriftPct"`

	// JourneyP99DriftPct is the percentage change in journey round-trip P99
	// from the first sample to the last. Zero when the journey phase is skipped.
	JourneyP99DriftPct float64 `json:"journeyP99DriftPct,omitempty"`

	// QueryHitP99DriftPct is the percentage change in query dispatch hit P99
	// from the first sample to the last. Zero when the query phase is skipped.
	QueryHitP99DriftPct float64 `json:"queryHitP99DriftPct,omitempty"`

	// CacheHitP99DriftPct is the percentage change in cache hit P99 from the
	// first sample to the last. Zero when the snapshot phase is skipped.
	CacheHitP99DriftPct float64 `json:"cacheHitP99DriftPct,omitempty"`

	Config SoakConfig `json:"config"`
}

// RunSoak runs the benchmark repeatedly for the configured duration, detecting
// memory leaks and performance degradation across iterations. Each iteration
// creates a fresh Bundle via the factory and runs the full benchmark suite,
// then forces a GC and records heap usage.
//
// Use a small profile (ProfileDev) for fast iterations — more data points give
// better trend detection. The returned SoakResult includes per-iteration
// samples and computed drift metrics.
func RunSoak(ctx context.Context, config SoakConfig, factory Factory) (*SoakResult, error) {
	soakCtx, cancel := context.WithTimeout(ctx, config.Duration)
	defer cancel()

	result := &SoakResult{
		Backend:  config.Config.Backend,
		Duration: config.Duration,
		Samples:  make([]SoakSample, 0, 64),
		Config:   config,
	}

	startTime := time.Now()

	var lastReport time.Time

	for soakCtx.Err() == nil {
		// Each iteration runs a full benchmark with a fresh factory call.
		iterCtx, iterCancel := context.WithTimeout(soakCtx, 5*time.Minute)

		res, err := newRunner(config.Config, factory).run(iterCtx)

		iterCancel()

		if err != nil {
			// A single iteration failure is not fatal — record what we have.
			if len(result.Samples) > 0 {
				break
			}

			return nil, fmt.Errorf("soak iteration %d: %w", len(result.Samples), err)
		}

		// Skip partial iterations where the context deadline cut the run
		// short (zero events written). These produce meaningless samples.
		if res.TotalEvents == 0 {
			break
		}

		// Force GC for a stable heap baseline between iterations.
		runtime.GC()

		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		sample := SoakSample{
			Iteration:   len(result.Samples),
			Duration:    res.Duration,
			Throughput:  res.WriteThroughput,
			WriteP50:    res.WriteLatency.P50,
			WriteP99:    res.WriteLatency.P99,
			LoadP50:     res.LoadLatency.P50,
			HeapBytes:   m.HeapAlloc,
			TotalEvents: res.TotalEvents,
			JourneyP99:  res.JourneyLatency.P99,
			QueryHitP99: res.QueryHitLatency.P99,
			CacheHitP99: res.CacheHitLatency.P99,
		}

		result.Samples = append(result.Samples, sample)

		// Intermediate progress report.
		if config.ProgressWriter != nil && config.ReportInterval > 0 {
			now := time.Now()

			if now.Sub(lastReport) >= config.ReportInterval {
				fmt.Fprintf(
					config.ProgressWriter,
					"soak: iter %d | %s/s | heap %s | p99 %s\n",
					sample.Iteration+1,
					formatFloat(sample.Throughput),
					formatBytes(sample.HeapBytes),
					roundDuration(sample.WriteP99),
				)

				lastReport = now
			}
		}
	}

	result.Iterations = len(result.Samples)
	result.Duration = time.Since(startTime)
	computeSoakTrends(result)

	return result, nil
}

// computeSoakTrends fills in the drift and leak-rate fields from the samples.
func computeSoakTrends(r *SoakResult) {
	if len(r.Samples) < 2 {
		return
	}

	first := r.Samples[0]
	last := r.Samples[len(r.Samples)-1]

	if last.HeapBytes >= first.HeapBytes {
		r.HeapGrowthBytes = last.HeapBytes - first.HeapBytes
	}

	if r.Iterations > 1 {
		r.HeapLeakRate = float64(r.HeapGrowthBytes) / float64(r.Iterations)
	}

	if first.Throughput > 0 {
		r.ThroughputDriftPct = (last.Throughput - first.Throughput) /
			first.Throughput * 100
	}

	if first.WriteP99 > 0 {
		r.WriteP99DriftPct = float64(last.WriteP99-first.WriteP99) /
			float64(first.WriteP99) * 100
	}

	if first.JourneyP99 > 0 {
		r.JourneyP99DriftPct = float64(last.JourneyP99-first.JourneyP99) /
			float64(first.JourneyP99) * 100
	}

	if first.QueryHitP99 > 0 {
		r.QueryHitP99DriftPct = float64(last.QueryHitP99-first.QueryHitP99) /
			float64(first.QueryHitP99) * 100
	}

	if first.CacheHitP99 > 0 {
		r.CacheHitP99DriftPct = float64(last.CacheHitP99-first.CacheHitP99) /
			float64(first.CacheHitP99) * 100
	}
}

// PrintSoakReport writes a human-readable soak test report.
func PrintSoakReport(w io.Writer, r *SoakResult) {
	fmt.Fprintf(w, "Soak Test: %s | %d iterations over %s\n",
		r.Backend, r.Iterations, roundDuration(r.Duration))
	fmt.Fprintln(w, strings.Repeat("=", 60))

	if len(r.Samples) == 0 {
		fmt.Fprintln(w, "No samples collected.")

		return
	}

	first := r.Samples[0]
	last := r.Samples[len(r.Samples)-1]

	fmt.Fprintf(w, "Throughput: %s/s → %s/s (%.1f%%)\n",
		formatFloat(first.Throughput), formatFloat(last.Throughput),
		r.ThroughputDriftPct)

	fmt.Fprintf(w, "Write P99:  %s → %s (%+.1f%%)\n",
		roundDuration(first.WriteP99), roundDuration(last.WriteP99),
		r.WriteP99DriftPct)

	if first.JourneyP99 > 0 {
		fmt.Fprintf(w, "Journey P99:%s → %s (%+.1f%%)\n",
			roundDuration(first.JourneyP99), roundDuration(last.JourneyP99),
			r.JourneyP99DriftPct)
	}

	if first.QueryHitP99 > 0 {
		fmt.Fprintf(w, "Query P99:  %s → %s (%+.1f%%)\n",
			roundDuration(first.QueryHitP99), roundDuration(last.QueryHitP99),
			r.QueryHitP99DriftPct)
	}

	if first.CacheHitP99 > 0 {
		fmt.Fprintf(w, "Cache P99:  %s → %s (%+.1f%%)\n",
			roundDuration(first.CacheHitP99), roundDuration(last.CacheHitP99),
			r.CacheHitP99DriftPct)
	}

	fmt.Fprintf(w, "Heap:       %s → %s (growth: %s, %s/iter)\n",
		formatBytes(first.HeapBytes), formatBytes(last.HeapBytes),
		formatBytes(r.HeapGrowthBytes),
		formatBytes(uint64(r.HeapLeakRate)))

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Per-iteration samples:")
	fmt.Fprintln(w, "  iter   throughput   heap      writeP50   writeP99")

	for _, s := range r.Samples {
		fmt.Fprintf(
			w, "  %-5d  %7s/s   %8s  %9s  %9s\n",
			s.Iteration+1,
			formatFloat(s.Throughput),
			formatBytes(s.HeapBytes),
			roundDuration(s.WriteP50),
			roundDuration(s.WriteP99),
		)
	}

	// New-phase per-iteration table (only when phases ran).
	if first.JourneyP99 > 0 || first.QueryHitP99 > 0 || first.CacheHitP99 > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Phase P99 per iteration:")
		fmt.Fprintln(w, "  iter   journey      query       cache")

		for _, s := range r.Samples {
			fmt.Fprintf(w, "  %-5d  %9s  %9s  %9s\n",
				s.Iteration+1,
				dashIfZero(s.JourneyP99),
				dashIfZero(s.QueryHitP99),
				dashIfZero(s.CacheHitP99),
			)
		}
	}
}

// dashIfZero returns "-" for zero durations (phase skipped), so the table
// doesn't show misleading "0s" values.
func dashIfZero(d time.Duration) string {
	if d == 0 {
		return "-"
	}

	return fmt.Sprint(roundDuration(d))
}

// WriteSoakJSON serializes a soak result as indented JSON.
func WriteSoakJSON(w io.Writer, r *SoakResult) error {
	return writeJSONAny(w, r)
}
