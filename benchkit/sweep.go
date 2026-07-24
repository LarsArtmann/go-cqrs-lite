package benchkit

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"sort"
	"strings"
)

// SweepResult holds the result of a single scaling-sweep data point.
type SweepResult struct {
	Parameter string  `json:"parameter"`
	Value     int     `json:"value"`
	Result    *Result `json:"result"`
}

// ScalingSweep runs a benchmark for each value in values, applying modifier
// to a fresh copy of base before each run. Returns one SweepResult per value,
// in the same order as values. Failed runs have a non-empty Result.Error.
//
// For GOMAXPROCS sweeps, the modifier must call runtime.GOMAXPROCS and the
// caller is responsible for restoring the original value (or use
// GOMAXPROCSSweep which handles this automatically).
func ScalingSweep(
	ctx context.Context,
	base Config,
	factory Factory,
	parameter string,
	values []int,
	modifier func(cfg *Config, v int),
) []SweepResult {
	results := make([]SweepResult, len(values))

	for i, v := range values {
		cfg := base
		modifier(&cfg, v)

		r, _ := Run(ctx, cfg, factory)
		results[i] = SweepResult{
			Parameter: parameter,
			Value:     v,
			Result:    r,
		}
	}

	return results
}

// GOMAXPROCSSweep runs a benchmark at each GOMAXPROCS setting.
// The original GOMAXPROCS is restored after the sweep.
// NOT safe for parallel execution — GOMAXPROCS is a global setting.
func GOMAXPROCSSweep(
	ctx context.Context,
	base Config,
	factory Factory,
	procs []int,
) []SweepResult {
	prev := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(prev)

	return ScalingSweep(ctx, base, factory, "gomaxprocs", procs,
		func(cfg *Config, v int) {
			runtime.GOMAXPROCS(v)
		})
}

// WorkerSweep runs a benchmark at each worker-count (Concurrency) setting.
func WorkerSweep(
	ctx context.Context,
	base Config,
	factory Factory,
	workers []int,
) []SweepResult {
	return ScalingSweep(ctx, base, factory, "workers", workers,
		func(cfg *Config, v int) {
			cfg.Concurrency = v
		})
}

// BatchSizeSweep runs a benchmark at each batch-size setting.
func BatchSizeSweep(
	ctx context.Context,
	base Config,
	factory Factory,
	sizes []int,
) []SweepResult {
	return ScalingSweep(ctx, base, factory, "batchSize", sizes,
		func(cfg *Config, v int) {
			cfg.Profile.BatchSize = v
		})
}

// StreamLengthSweep runs a benchmark at each events-per-stream setting.
// The total event count changes with each value, so throughput numbers
// are NOT directly comparable across data points — use latency percentiles
// and per-event overhead instead.
func StreamLengthSweep(
	ctx context.Context,
	base Config,
	factory Factory,
	lengths []int,
) []SweepResult {
	return ScalingSweep(ctx, base, factory, "streamLength", lengths,
		func(cfg *Config, v int) {
			cfg.Profile.EventsPerStream = v
		})
}

// PrintSweep writes a scaling-sweep comparison table.
func PrintSweep(w io.Writer, results []SweepResult) {
	if len(results) == 0 {
		return
	}

	param := results[0].Parameter

	fmt.Fprintf(w, "\n%s Sweep\n", titleCase(param))
	fmt.Fprintln(w, strings.Repeat("=", 80))

	header := fmt.Sprintf(
		"%-12s %12s %14s %14s %10s %10s",
		param, "Write ops/s", "RawSink ops/s", "Load P50", "Heap MB", "Disk MB",
	)
	fmt.Fprintln(w, header)
	fmt.Fprintln(w, strings.Repeat("-", len(header)))

	for _, sr := range results {
		r := sr.Result
		if r.Error != "" {
			fmt.Fprintf(w, "%-12d %s\n", sr.Value, "FAILED: "+truncate(r.Error, 50))

			continue
		}

		fmt.Fprintf(
			w, "%-12d %12s %14s %14s %10s %10s\n",
			sr.Value,
			formatFloat(r.WriteThroughput),
			formatFloatOrDash(r.RawSinkThroughput),
			roundDuration(r.LoadLatency.P50),
			formatBytes(r.Memory.After),
			formatBytes(uint64(r.Disk.DatabaseBytes)),
		)
	}

	fmt.Fprintln(w)
}

// WriteSweepJSON serializes sweep results as a JSON array.
func WriteSweepJSON(w io.Writer, results []SweepResult) error {
	type export struct {
		Parameter string  `json:"parameter"`
		Value     int     `json:"value"`
		Result    *Result `json:"result"`
	}

	items := make([]export, len(results))
	for i, sr := range results {
		items[i] = export(sr)
	}

	return writeJSONAny(w, items)
}

// SortedSweepResults returns sweep results sorted by parameter value.
func SortedSweepResults(results []SweepResult) []SweepResult {
	sorted := make([]SweepResult, len(results))
	copy(sorted, results)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value < sorted[j].Value
	})

	return sorted
}

func titleCase(s string) string {
	if s == "" {
		return s
	}

	return strings.ToUpper(s[:1]) + s[1:]
}

func formatFloatOrDash(v float64) string {
	if v == 0 {
		return "-"
	}

	return formatFloat(v)
}
