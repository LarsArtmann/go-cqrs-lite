package benchkit

import (
	"context"
	"fmt"
	"math"
	"slices"
	"sort"
)

// VariationThreshold is the cross-run coefficient of variation above which a
// metric is too noisy to draw conclusions from. A metric above it must be
// re-measured — more repeats, a quieter machine, or a longer benchtime —
// before anyone treats a difference as real.
const VariationThreshold = 0.10

// MetricVariation describes how one metric varied across repeat runs.
type MetricVariation struct {
	// Name is the benchstat-compatible metric name (e.g. "write_p50_ns").
	Name string `json:"name"`
	// Unit is the benchstat unit token for the metric ("ns/op", "ops/s", ...).
	Unit string `json:"unit"`
	// Samples holds the value recorded by every run that produced this metric,
	// in execution order. A run that skipped the metric contributes nothing.
	Samples []float64 `json:"samples"`
	// Mean is the arithmetic mean of Samples.
	Mean float64 `json:"mean"`
	// StdDev is the population standard deviation of Samples.
	StdDev float64 `json:"stdDev"`
	// CoV is StdDev/Mean — the reliability indicator. Under
	// [VariationThreshold] the metric is comparable across runs.
	CoV float64 `json:"cov"`
	// Reliable reports whether CoV stayed under [VariationThreshold].
	Reliable bool `json:"reliable"`
}

// RepeatedResult is the complete output of a multi-run benchmark
// (Config.Repeat > 1): every run, the median run, and per-metric cross-run
// dispersion.
//
// Keeping the individual runs is what allows [WriteBenchstatRepeated] to emit
// one sample per run — the sample count benchstat needs before it will report
// a confidence interval. Reporting only the median run made every
// `--format benchstat` file a single-sample file, so benchstat could compare
// point estimates but never say whether a difference was real.
type RepeatedResult struct {
	// Median is the run whose write throughput sits at the median of all runs.
	// It carries the full metric set plus the Repeat* / MetricVariation
	// annotations, so existing consumers that only read a *Result keep working.
	Median *Result
	// Runs holds every run in execution order.
	Runs []*Result
}

// RunRepeated runs the benchmark Config.Repeat times and returns every run
// plus the median run annotated with per-metric cross-run dispersion.
//
// Each repeat calls factory() fresh, so isolated factories (unique temp dirs)
// give independent measurements. A factory that reopens the same path makes
// later repeats inherit earlier runs' data — see [Run].
//
// Config.Repeat below 2 is honoured as a single run; consumers wanting the
// single-run fast path should call [Run] directly.
func RunRepeated(ctx context.Context, config Config, factory Factory) (*RepeatedResult, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}

	repeat := max(config.Repeat, 1)

	single := config
	single.Repeat = 0

	runs := make([]*Result, 0, repeat)

	for i := range repeat {
		result, err := newRunner(single, factory).run(ctx)
		if err != nil {
			return nil, fmt.Errorf("repeat run %d/%d: %w", i+1, repeat, err)
		}

		runs = append(runs, result)
	}

	median := medianByWriteThroughput(runs)
	if repeat > 1 {
		annotateRepeatVariation(median, runs)
	}

	return &RepeatedResult{Median: median, Runs: runs}, nil
}

// Reliable reports whether every measured metric stayed under
// [VariationThreshold]. A false result means at least one metric moved more
// than 10% between runs and the median alone is not decision-grade.
func (rr *RepeatedResult) Reliable() bool {
	if rr == nil || rr.Median == nil {
		return false
	}

	for _, v := range rr.Median.MetricVariation {
		if !v.Reliable {
			return false
		}
	}

	return true
}

// NoisyMetrics returns the names of the metrics whose cross-run coefficient of
// variation exceeded [VariationThreshold], worst first. Empty when every metric
// was stable.
func (rr *RepeatedResult) NoisyMetrics() []string {
	if rr == nil || rr.Median == nil {
		return nil
	}

	var noisy []MetricVariation

	for _, v := range rr.Median.MetricVariation {
		if !v.Reliable {
			noisy = append(noisy, v)
		}
	}

	sort.SliceStable(noisy, func(i, j int) bool { return noisy[i].CoV > noisy[j].CoV })

	names := make([]string, len(noisy))
	for i, v := range noisy {
		names[i] = v.Name
	}

	return names
}

// medianByWriteThroughput picks the run whose write throughput is the median
// of all runs. Sorting by throughput (rather than trusting execution order)
// keeps the reported run representative: the median run's full metric set is
// what a single-run consumer sees.
func medianByWriteThroughput(runs []*Result) *Result {
	if len(runs) == 0 {
		return nil
	}

	order := make([]int, len(runs))
	for i := range order {
		order[i] = i
	}

	sort.SliceStable(order, func(a, b int) bool {
		return runs[order[a]].WriteThroughput < runs[order[b]].WriteThroughput
	})

	return runs[order[len(order)/2]]
}

// annotateRepeatVariation attaches cross-run dispersion to the median run.
// It keeps the legacy Repeat* fields (throughput only) working for consumers
// that predate MetricVariation, and records every metric in MetricVariation.
func annotateRepeatVariation(median *Result, runs []*Result) {
	if median == nil || len(runs) < 2 {
		return
	}

	median.MetricVariation = computeVariation(runs)

	throughput := make([]float64, len(runs))
	for i, run := range runs {
		throughput[i] = run.WriteThroughput
	}

	mean, stdDev, cov := dispersion(throughput)

	sorted := slices.Clone(throughput)
	slices.Sort(sorted)

	median.RepeatCount = len(runs)
	median.RepeatSamples = sorted
	median.RepeatMin = sorted[0]
	median.RepeatMax = sorted[len(sorted)-1]
	median.RepeatMean = mean
	median.RepeatStdDev = stdDev
	median.RepeatCoV = cov
	median.RepeatIsReliable = cov < VariationThreshold
}

// computeVariation builds per-metric cross-run dispersion, ordered as the
// metrics appear in a report (phase-grouped, deterministic). Metrics recorded
// by fewer than two runs are dropped: dispersion of one sample is not a
// measurement, and reporting CoV 0 for it would read as "perfectly stable".
func computeVariation(runs []*Result) []MetricVariation {
	units := make(map[string]string, 40)

	var order []string

	values := make(map[string][]float64, 40)

	for _, run := range runs {
		for _, m := range resultMetrics(run) {
			if _, seen := units[m.suffix]; !seen {
				order = append(order, m.suffix)
				units[m.suffix] = m.unit
			}

			values[m.suffix] = append(values[m.suffix], m.value)
		}
	}

	out := make([]MetricVariation, 0, len(order))

	for _, name := range order {
		samples := values[name]
		if len(samples) < 2 {
			continue
		}

		mean, stdDev, cov := dispersion(samples)

		out = append(out, MetricVariation{
			Name:     name,
			Unit:     units[name],
			Samples:  samples,
			Mean:     mean,
			StdDev:   stdDev,
			CoV:      cov,
			Reliable: cov < VariationThreshold,
		})
	}

	return out
}

// dispersion returns the arithmetic mean, the population standard deviation,
// and the coefficient of variation of samples. CoV is 0 when the mean is 0 —
// a metric with no measurable level has no relative spread.
func dispersion(samples []float64) (mean, stdDev, cov float64) {
	if len(samples) == 0 {
		return 0, 0, 0
	}

	var sum float64

	for _, s := range samples {
		sum += s
	}

	mean = sum / float64(len(samples))

	var sqDiffSum float64

	for _, s := range samples {
		diff := s - mean
		sqDiffSum += diff * diff
	}

	stdDev = math.Sqrt(sqDiffSum / float64(len(samples)))

	if mean > 0 {
		cov = stdDev / mean
	}

	return mean, stdDev, cov
}
