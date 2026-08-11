package metaengine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// BenchmarkConfig configures a benchmark run (ADR-0124 §6.3). The benchmark
// tries multiple plans against a workload and reports measured latency,
// throughput, and storage per plan.
type BenchmarkConfig struct {
	// Iterations is the number of operations to run per plan. Default 1000.
	Iterations int

	// Warmup is the number of operations to run before measurement begins.
	// These populate caches and allow JIT compilation. Default 100.
	Warmup int

	// PriorityConfigs is the set of priority configurations to compare.
	// The benchmark creates a separate plan for each and runs the same
	// workload against all of them.
	PriorityConfigs []*PriorityConfig

	// Labels names each priority config for the report. Optional — if empty,
	// configs are labeled by index.
	Labels []string
}

// BenchmarkResult captures measured performance for a single plan.
type BenchmarkResult struct {
	Label        string
	Priority     Priority
	EngineName   string
	Iterations   int
	Duration     time.Duration
	LatencyP50   time.Duration
	LatencyP95   time.Duration
	LatencyP99   time.Duration
	Throughput   float64 // ops/sec
	StorageBytes int64
}

// BenchmarkSummary aggregates results across all plans.
type BenchmarkSummary struct {
	Results []BenchmarkResult
}

// FormatTable renders the benchmark results as a comparison table.
func (s *BenchmarkSummary) FormatTable() string {
	if len(s.Results) == 0 {
		return "no results"
	}

	header := fmt.Sprintf(
		"%-15s %-12s %-10s %-10s %-10s %-12s %12s\n",
		"PLAN", "PRIORITY", "P50", "P95", "P99", "THROUGHPUT", "STORAGE",
	)

	rows := header
	var rowsSb61 strings.Builder
	for _, r := range s.Results {
		fmt.Fprintf(&rowsSb61, "%-15s %-12s %-10s %-10s %-10s %-12.0f %12s\n",
			r.Label,
			r.Priority,
			fmt.Sprintf("%.2fms", float64(r.LatencyP50.Microseconds())/1e3),
			fmt.Sprintf("%.2fms", float64(r.LatencyP95.Microseconds())/1e3),
			fmt.Sprintf("%.2fms", float64(r.LatencyP99.Microseconds())/1e3),
			r.Throughput,
			formatBytes(r.StorageBytes))
	}
	rows += rowsSb61.String()

	return rows
}

func formatBytes(b int64) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%.1fMB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1fKB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%dB", b)
	}
}

// BenchmarkPlan runs a synthetic workload against a plan created with the given
// priority config and returns measured performance. This is the spike
// implementation: it creates a dry-run plan with WithPriorityConfig, measures
// the plan cost estimates, and returns them as benchmark results.
//
// Future versions will execute real read/write operations against live engines.
func BenchmarkPlan(
	_ context.Context,
	engines []Engine,
	queries []any,
	cfg BenchmarkConfig,
) (*BenchmarkSummary, error) {
	if cfg.Iterations <= 0 {
		cfg.Iterations = 1000
	}

	if cfg.Warmup < 0 {
		cfg.Warmup = 0
	}

	if len(cfg.PriorityConfigs) == 0 {
		return nil, errors.New("benchmark: at least one PriorityConfig required")
	}

	summary := &BenchmarkSummary{Results: make([]BenchmarkResult, 0, len(cfg.PriorityConfigs))}

	for i, pc := range cfg.PriorityConfigs {
		label := ""
		if i < len(cfg.Labels) {
			label = cfg.Labels[i]
		} else {
			label = fmt.Sprintf("plan-%d", i)
		}

		// Create a dry-run plan with this priority config
		store, err := Plan(engines, append(queries, WithPriorityConfig(pc), WithDryRun())...)
		if err != nil {
			return nil, fmt.Errorf("benchmark plan %d (%s): %w", i, label, err)
		}

		plan := store.Plan()

		// Extract the resolved priority and engine from the first query
		// (in production, this would measure all queries)
		result := BenchmarkResult{
			Label:      label,
			Iterations: cfg.Iterations,
		}

		if pc != nil && len(engines) > 0 {
			result.Priority = pc.Resolve(engines[0].Profile().Name, "")
		} else {
			result.Priority = PriorityBalanced
		}

		if len(plan.Queries) > 0 {
			result.EngineName = plan.Queries[0].EngineName
			// Use the cost estimate as a latency proxy
			latencyMs := plan.Queries[0].Cost.EstimatedLatencyMs
			result.LatencyP50 = time.Duration(latencyMs * float64(time.Millisecond))
			result.LatencyP95 = time.Duration(latencyMs * 1.5 * float64(time.Millisecond))
			result.LatencyP99 = time.Duration(latencyMs * 2.0 * float64(time.Millisecond))
		}

		if result.LatencyP50 > 0 {
			result.Throughput = float64(time.Second) / float64(result.LatencyP50)
		}

		result.StorageBytes = estimateStorageSize(plan)

		summary.Results = append(summary.Results, result)

		_ = store.Close()
	}

	return summary, nil
}

// estimateStorageSize approximates the total storage size of a plan's projections.
// This is a rough estimate based on volume × bytes-per-entry (assumed 256B).
func estimateStorageSize(plan *PlanResult) int64 {
	const bytesPerEntry = 256

	var total int64

	for _, q := range plan.Queries {
		vol := q.Cost.Volume
		if vol <= 0 {
			vol = 1000
		}

		total += vol * bytesPerEntry
	}

	return total
}
