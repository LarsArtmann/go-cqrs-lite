package benchkit

import (
	"fmt"
	"io"
	"strings"
	"time"
)

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

	if first.JourneyP99 > 0 && last.JourneyP99 > 0 {
		fmt.Fprintf(w, "Journey P99:%s → %s (%+.1f%%)\n",
			roundDuration(first.JourneyP99), roundDuration(last.JourneyP99),
			r.JourneyP99DriftPct)
	}

	if first.QueryHitP99 > 0 && last.QueryHitP99 > 0 {
		fmt.Fprintf(w, "Query P99:  %s → %s (%+.1f%%)\n",
			roundDuration(first.QueryHitP99), roundDuration(last.QueryHitP99),
			r.QueryHitP99DriftPct)
	}

	if first.CacheHitP99 > 0 && last.CacheHitP99 > 0 {
		fmt.Fprintf(w, "Cache P99:  %s → %s (%+.1f%%)\n",
			roundDuration(first.CacheHitP99), roundDuration(last.CacheHitP99),
			r.CacheHitP99DriftPct)
	}

	if first.GCMaxPause > 0 && last.GCMaxPause > 0 {
		fmt.Fprintf(w, "GC Pause:   %s → %s (%+.1f%%)\n",
			roundDuration(first.GCMaxPause), roundDuration(last.GCMaxPause),
			r.GCMaxPauseDriftPct)
	}

	if first.AllocBytes > 0 && last.AllocBytes > 0 {
		fmt.Fprintf(w, "Allocs:     %s → %s (%+.1f%%)\n",
			formatBytes(first.AllocBytes), formatBytes(last.AllocBytes),
			r.AllocGrowthPct)
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
			fmt.Fprintf(
				w, "  %-5d  %9s  %9s  %9s\n",
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
