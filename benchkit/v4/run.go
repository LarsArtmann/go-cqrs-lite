package benchkit

import (
	"context"
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
// The returned Result holds the median run's full metrics, annotated with
// min/max throughput across all N runs plus cross-run dispersion for every
// measured metric (Result.MetricVariation). Callers that need the individual
// runs — for example to emit per-run benchstat samples — should call
// [RunRepeated] instead.
func Run(ctx context.Context, config Config, factory Factory) (*Result, error) {
	if config.Repeat > 1 {
		repeated, err := RunRepeated(ctx, config, factory)
		if err != nil {
			return nil, err
		}

		return repeated.Median, nil
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	return newRunner(config, factory).run(ctx)
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
