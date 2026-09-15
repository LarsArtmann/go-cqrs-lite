package benchkit

import (
	"bytes"
	"context"
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/stack/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// syntheticRun builds a Result carrying only the metrics the dispersion tests
// care about, so computeVariation can be exercised without running a backend.
func syntheticRun(throughput float64, p50 time.Duration) *Result {
	return &Result{
		WriteThroughput: throughput,
		WriteLatency:    LatencyStats{Count: 10, P50: p50, P99: 4 * p50},
	}
}

func TestRunRepeated_ReturnsEveryRunAndMedian(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(),
		loadScaledCeiling(soakTestScale(90*time.Second)))
	defer cancel()

	repeated, err := RunRepeated(ctx, Config{
		Profile: ProfileDev,
		Repeat:  3,
	}, func() (*stack.Bundle, error) { return memory.New() })
	if err != nil {
		t.Fatalf("RunRepeated: %v", err)
	}

	if len(repeated.Runs) != 3 {
		t.Fatalf("Runs len = %d, want 3", len(repeated.Runs))
	}

	if repeated.Median == nil {
		t.Fatal("Median is nil")
	}

	if !slices.Contains(repeated.Runs, repeated.Median) {
		t.Error("Median must be one of the recorded runs")
	}

	if repeated.Median.RepeatCount != 3 {
		t.Errorf("RepeatCount = %d, want 3 on the median", repeated.Median.RepeatCount)
	}

	if len(repeated.Median.MetricVariation) == 0 {
		t.Error("MetricVariation is empty — every-metric dispersion missing")
	}

	if !slices.ContainsFunc(repeated.Median.MetricVariation, func(v MetricVariation) bool {
		return v.Name == "write_throughput" && len(v.Samples) == 3
	}) {
		t.Error("write_throughput variation must carry one sample per run")
	}
}

func TestRunRepeated_SingleRunHasNoVariation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(),
		loadScaledCeiling(soakTestScale(90*time.Second)))
	defer cancel()

	repeated, err := RunRepeated(ctx, Config{
		Profile: ProfileDev,
		Repeat:  1,
	}, func() (*stack.Bundle, error) { return memory.New() })
	if err != nil {
		t.Fatalf("RunRepeated: %v", err)
	}

	if len(repeated.Runs) != 1 {
		t.Fatalf("Runs len = %d, want 1", len(repeated.Runs))
	}

	if repeated.Median.RepeatCount != 0 {
		t.Errorf("RepeatCount = %d, want 0 for a single run", repeated.Median.RepeatCount)
	}

	if len(repeated.Median.MetricVariation) != 0 {
		t.Errorf("MetricVariation len = %d, want 0 for a single run",
			len(repeated.Median.MetricVariation))
	}
}

func TestComputeVariation_MathAndOrdering(t *testing.T) {
	t.Parallel()

	variation := computeVariation([]*Result{
		syntheticRun(100, 1000),
		syntheticRun(110, 1000),
		syntheticRun(120, 1000),
	})

	if len(variation) < 2 {
		t.Fatalf("variation len = %d, want at least throughput + write_p50", len(variation))
	}

	if variation[0].Name != "write_throughput" {
		t.Errorf("first metric = %q, want write_throughput (report order)", variation[0].Name)
	}

	thr := variation[0]

	if len(thr.Samples) != 3 {
		t.Fatalf("samples len = %d, want 3", len(thr.Samples))
	}

	if want := 110.0; !nearlyEqual(thr.Mean, want) {
		t.Errorf("Mean = %v, want %v", thr.Mean, want)
	}

	wantStdDev := math.Sqrt((100 + 0 + 100) / 3.0)
	if !nearlyEqual(thr.StdDev, wantStdDev) {
		t.Errorf("StdDev = %v, want %v", thr.StdDev, wantStdDev)
	}

	if !nearlyEqual(thr.CoV, wantStdDev/110) {
		t.Errorf("CoV = %v, want %v", thr.CoV, wantStdDev/110)
	}

	if !thr.Reliable {
		t.Errorf("CoV %.2f%% should be under the 10%% reliability threshold", thr.CoV*100)
	}
}

func TestComputeVariation_NoisyMetricNotReliable(t *testing.T) {
	t.Parallel()

	variation := computeVariation([]*Result{
		syntheticRun(100, 1000),
		syntheticRun(150, 1000),
		syntheticRun(200, 1000),
	})

	idx := slices.IndexFunc(variation, func(v MetricVariation) bool {
		return v.Name == "write_throughput"
	})
	if idx < 0 {
		t.Fatal("write_throughput variation missing")
	}

	if variation[idx].Reliable {
		t.Errorf("CoV %.1f%% must not be Reliable above the threshold", variation[idx].CoV*100)
	}

	rr := &RepeatedResult{Median: &Result{MetricVariation: variation}}
	if rr.Reliable() {
		t.Error("RepeatedResult.Reliable must be false with one noisy metric")
	}

	if !slices.Contains(rr.NoisyMetrics(), "write_throughput") {
		t.Errorf("NoisyMetrics = %v, want write_throughput listed", rr.NoisyMetrics())
	}
}

func TestComputeVariation_SingleSampleMetricDropped(t *testing.T) {
	t.Parallel()

	// write_p50_ns is set in every run, but allocs_per_op only in one:
	// a single sample cannot express dispersion and must not surface as
	// a fake "perfectly stable" CoV of 0.
	runs := []*Result{syntheticRun(100, 1000), syntheticRun(120, 1000)}
	runs[1].AllocsPerOp = 9

	if slices.ContainsFunc(computeVariation(runs), func(v MetricVariation) bool {
		return v.Name == "allocs_per_op"
	}) {
		t.Error("metric recorded by a single run must be dropped, not reported as stable")
	}
}

func TestDispersion_EmptyAndConstant(t *testing.T) {
	t.Parallel()

	if mean, stdDev, cov := dispersion(nil); mean != 0 || stdDev != 0 || cov != 0 {
		t.Errorf("dispersion(nil) = %v, %v, %v; want zeros", mean, stdDev, cov)
	}

	mean, stdDev, cov := dispersion([]float64{7, 7, 7})
	if mean != 7 || stdDev != 0 || cov != 0 {
		t.Errorf("constant samples: mean=%v stdDev=%v cov=%v; want 7, 0, 0", mean, stdDev, cov)
	}
}

func TestWriteBenchstatRepeated_OneLinePerRun(t *testing.T) {
	t.Parallel()

	rr := &RepeatedResult{
		Runs: []*Result{
			syntheticRun(100, 1000),
			syntheticRun(110, 1100),
			syntheticRun(120, 1200),
		},
	}
	rr.Median = medianByWriteThroughput(rr.Runs)

	var buf bytes.Buffer
	WriteBenchstatRepeated(&buf, rr)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) == 0 || lines[0] == "" {
		t.Fatal("no benchstat output")
	}

	counts := make(map[string]int)
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) != 4 {
			t.Fatalf("line %q does not match 'BenchmarkX-N_m\t1\t<value> <unit>'", line)
		}

		if parts[1] != "1" {
			t.Errorf("line %q: iteration column = %q, want 1", line, parts[1])
		}

		counts[parts[0]]++
	}

	if len(counts) == 0 {
		t.Fatal("no metrics emitted")
	}

	for name, count := range counts {
		if count != 3 {
			t.Errorf("metric %q emitted %d lines, want one per run (3)", name, count)
		}
	}

	if !strings.Contains(buf.String(), "write_throughput") {
		t.Error("write_throughput missing from repeated benchstat output")
	}
}

func TestWriteBenchstatRepeated_NilAndSingleRun(t *testing.T) {
	t.Parallel()

	var nilBuf bytes.Buffer
	WriteBenchstatRepeated(&nilBuf, nil)
	if nilBuf.Len() != 0 {
		t.Error("nil RepeatedResult must emit nothing")
	}

	result := syntheticRun(100, 1000)
	single := &RepeatedResult{Median: result, Runs: []*Result{result}}

	var singleBuf bytes.Buffer
	WriteBenchstatRepeated(&singleBuf, single)

	var direct bytes.Buffer
	WriteBenchstat(&direct, result)

	if singleBuf.String() != direct.String() {
		t.Error("single-run repeated output must match WriteBenchstat exactly")
	}
}

func nearlyEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}
