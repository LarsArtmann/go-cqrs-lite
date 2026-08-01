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

	// Evidence-grade metrics — surface GC pressure, allocation cost, and tail risk
	if result.GCCount > 0 {
		b.ReportMetric(float64(result.GCCount), "gc-cycles")
		b.ReportMetric(float64(result.GCMaxPause.Nanoseconds()), "ns/gc-max-pause")
		b.ReportMetric(result.GCPercent, "gc-percent")
	}

	if result.AllocsPerOp > 0 {
		b.ReportMetric(result.AllocsPerOp, "allocs/op")
		b.ReportMetric(result.BytesPerOp, "B/op")
	}

	if result.TailRatio > 0 {
		b.ReportMetric(result.TailRatio, "tail-ratio")
	}

	if result.WriteTailRatio > 0 {
		b.ReportMetric(result.WriteTailRatio, "write-tail-ratio")
	}

	if result.ColdReadLatency.Count > 0 {
		b.ReportMetric(float64(result.ColdReadLatency.P99.Nanoseconds()), "ns/cold-read-p99")
	}

	if result.Disk.WriteAmplification > 0 {
		b.ReportMetric(result.Disk.WriteAmplification, "write-amp")
	}

	if result.IntegrityErrors > 0 {
		b.ReportMetric(float64(result.IntegrityErrors), "integrity-errors")
	}

	if result.MetaEngineApplyLatency.Count > 0 {
		b.ReportMetric(float64(result.MetaEngineScanLatency.P99.Nanoseconds()), "ns/me-scan-p99")
		b.ReportMetric(float64(result.MetaEnginePointReadLatency.P99.Nanoseconds()), "ns/me-point-p99")
		b.ReportMetric(result.MetaEngineApplyConcurrent, "me-events/sec-concurrent")

		if result.MetaEngineSQLiteScanLatency.Count > 0 {
			b.ReportMetric(float64(result.MetaEngineSQLiteScanLatency.P99.Nanoseconds()), "ns/me-sqlite-scan-p99")
			b.ReportMetric(float64(result.MetaEngineSQLitePointReadLatency.P99.Nanoseconds()), "ns/me-sqlite-point-p99")
			b.ReportMetric(result.MetaEngineSQLiteApplyThroughput, "me-sqlite-events/sec")
		}
	}
}
