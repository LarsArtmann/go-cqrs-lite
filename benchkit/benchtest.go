package benchkit

import (
	"context"
	"testing"
	"time"
)

// RunSuite runs a single benchkit benchmark and reports key metrics via
// b.ReportMetric. Use from a testing.B function to integrate benchkit
// benchmarks into the standard Go benchmark suite (go test -bench=).
//
// The function runs the benchmark once (not b.N iterations — each run is
// a full write+read+project workload). For multiple samples, use
// Config.Repeat or go test -count=N.
//
// Example:
//
//	func BenchmarkBenchkitSuite_Memory(b *testing.B) {
//	    benchkit.RunSuite(b, benchkit.Config{
//	        Profile: benchkit.ProfileSmall,
//	    }, func() (*stack.Bundle, error) {
//	        return memory.New()
//	    })
//	}
//
// Run with: go test -bench=BenchmarkBenchkitSuite -benchtime=1x ./stack/bench/...
func RunSuite(b *testing.B, config Config, factory Factory) {
	b.Helper()

	ctx, cancel := context.WithTimeout(b.Context(), 30*time.Minute)
	defer cancel()

	result, err := Run(ctx, config, factory)
	if err != nil {
		b.Fatalf("benchkit.Run: %v", err)
	}

	b.ReportMetric(result.WriteThroughput, "events/sec")
	b.ReportMetric(float64(result.WriteLatency.P50.Nanoseconds()), "ns/write-p50")
	b.ReportMetric(float64(result.WriteLatency.P99.Nanoseconds()), "ns/write-p99")

	if result.RawSinkThroughput > 0 {
		b.ReportMetric(result.RawSinkThroughput, "raw_sink_events/sec")
		b.ReportMetric(float64(result.RawSinkLatency.P50.Nanoseconds()), "ns/raw-sink-p50")
		b.ReportMetric(float64(result.RawSinkLatency.P99.Nanoseconds()), "ns/raw-sink-p99")
	}

	b.ReportMetric(float64(result.LoadLatency.P50.Nanoseconds()), "ns/load-p50")
	b.ReportMetric(float64(result.LoadLatency.P99.Nanoseconds()), "ns/load-p99")

	if result.ReadAllTime > 0 {
		b.ReportMetric(float64(result.ReadAllTime.Nanoseconds()), "ns/readall")
	}

	if result.ReadFromTime > 0 {
		b.ReportMetric(float64(result.ReadFromTime.Nanoseconds()), "ns/readfrom")
	}

	if result.ProjectionEvents > 0 {
		b.ReportMetric(float64(result.ProjectionEvents), "projection-events")
	}

	if result.RecoveryTime > 0 {
		b.ReportMetric(float64(result.RecoveryTime.Nanoseconds()), "ns/recovery")
	}
}
