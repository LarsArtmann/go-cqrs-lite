package benchkit

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"
)

// Run executes a benchmark against one backend and returns the result.
//
// The factory is called once to create the Bundle. All phases (write, read,
// read-model, projection, durability) run against that same Bundle. The Bundle
// is closed automatically after the run. When Warmup > 0, the factory is called
// a second time for a throwaway warmup Bundle that never pollutes measurement.
//
// When Config.Repeat > 1, the benchmark runs N times. Each repeat calls
// factory() fresh, so for in-memory backends each run is fully isolated.
// For persistent backends (SQLite file, Pebble directory), the factory opens
// the same path — meaning later repeats inherit earlier runs' data. To ensure
// isolation with persistent backends, provide a factory that creates a unique
// path per call (e.g., using a temp dir with a unique suffix).
// The returned Result holds the median run's full metrics, annotated
// with min/max throughput across all N runs plus statistical reliability
// metrics (StdDev, CoV, IsReliable).
func Run(ctx context.Context, config Config, factory Factory) (*Result, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}

	if config.Repeat > 1 {
		return runRepeated(ctx, config, factory)
	}

	return newRunner(config, factory).run(ctx)
}

// runRepeated executes the benchmark N times, returning the median result
// annotated with min/max throughput spread and statistical reliability metrics.
func runRepeated(ctx context.Context, config Config, factory Factory) (*Result, error) {
	single := config
	single.Repeat = 0

	results := make([]*Result, 0, config.Repeat)

	for i := range config.Repeat {
		r, err := newRunner(single, factory).run(ctx)
		if err != nil {
			return nil, fmt.Errorf("repeat run %d/%d: %w", i+1, config.Repeat, err)
		}

		results = append(results, r)
	}

	// Sort results by throughput so the median index actually corresponds to
	// the median throughput, not insertion order. The previous code sorted a
	// separate samples slice but picked from the unsorted results array.
	sort.Slice(results, func(i, j int) bool {
		return results[i].WriteThroughput < results[j].WriteThroughput
	})

	medianIdx := len(results) / 2
	median := results[medianIdx]

	samples := make([]float64, len(results))
	for i, r := range results {
		samples[i] = r.WriteThroughput
	}

	// Compute statistical reliability: mean, stddev, coefficient of variation.
	// CoV is the key metric: CoV < 0.10 means results are trustworthy.
	var sum float64

	for _, s := range samples {
		sum += s
	}

	mean := sum / float64(len(samples))

	var sqDiffSum float64

	for _, s := range samples {
		diff := s - mean
		sqDiffSum += diff * diff
	}

	stdDev := math.Sqrt(sqDiffSum / float64(len(samples)))

	var cov float64

	if mean > 0 {
		cov = stdDev / mean
	}

	median.RepeatCount = config.Repeat
	median.RepeatMin = samples[0]
	median.RepeatMax = samples[len(samples)-1]
	median.RepeatSamples = samples
	median.RepeatMean = mean
	median.RepeatStdDev = stdDev
	median.RepeatCoV = cov
	median.RepeatIsReliable = cov < 0.10

	return median, nil
}

// Compare executes the same benchmark against multiple backends and returns
// a map of backend name to Result. Each backend gets a fresh Bundle.
//
// Backends whose factory returns an error are included in the result map with
// a zero-valued Result containing the error message — they do not abort the
// comparison.
func Compare(
	ctx context.Context,
	config Config,
	factories map[string]Factory,
) (map[string]*Result, error) {
	results := make(map[string]*Result, len(factories))

	for name, factory := range factories {
		cfg := config
		cfg.Backend = name

		result, err := Run(ctx, cfg, factory)
		if err != nil {
			results[name] = &Result{
				Backend:   name,
				Profile:   cfg.Profile.Name,
				Timestamp: time.Now(),
				Error:     err.Error(),
			}

			continue
		}

		results[name] = result
	}

	return results, nil
}
