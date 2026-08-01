package metaengine

import (
	"fmt"
	"math"
)

// WorkloadStats holds observed or estimated workload statistics for a
// collection. When provided to Plan via WithWorkloadStats, the planner
// can recommend whether a projection should be materialized (persisted)
// or replayed on-demand.
//
// All rates are per-second. Zero values mean "no data" — the planner
// skips materialization analysis when WorkloadStats are absent.
type WorkloadStats struct {
	// WriteRatePerSec is the average number of events written to the
	// stream backing this collection per second.
	WriteRatePerSec float64

	// ReadRatePerSec is the average number of queries executed against
	// this collection per second.
	ReadRatePerSec float64

	// AvgStreamLength is the average number of events in each stream.
	// Used to estimate replay cost: replaying a stream of N events
	// costs N * fold_cost.
	AvgStreamLength float64
}

// foldCostPerEvent is a normalized cost unit for folding one event.
// It is deliberately abstract — the formula only needs relative comparison.
const foldCostPerEvent = 1.0

// queryCostPerLookup is the cost of a single point query against a
// materialized projection. Cheaper than replay because it is O(1).
const queryCostPerLookup = 0.1

// ReplayCost estimates the cost of replaying a stream to answer a query.
//
//	replay_cost = read_rate * avg_stream_length * fold_cost_per_event
func ReplayCost(stats WorkloadStats) float64 {
	return replayCost(stats)
}

// MaterializeCost estimates the cost of maintaining a materialized projection.
//
//	materialize_cost = write_rate * fold_cost_per_event + read_rate * query_cost_per_lookup
func MaterializeCost(stats WorkloadStats) float64 {
	return materializeCost(stats)
}

// ShouldMaterialize returns true when the materialization cost is lower
// than the replay cost. When costs are equal, it defaults to materialize
// (materialized projections offer better read latency).
func ShouldMaterialize(stats WorkloadStats) bool {
	return shouldMaterialize(stats)
}

// ObservedWorkloadStats returns observed workload statistics derived from
// the Store's internal counters. WriteRatePerSec is computed from the total
// Apply() calls divided by uptime. ReadRatePerSec is computed from the total
// ExecuteCtx/ExecuteTyped calls divided by uptime. AvgStreamLength is left
// as 0 (requires domain knowledge the Store doesn't have).
//
// This is a convenience for consumers who want to feed stats back into
// Plan via WithWorkloadStats. For more accurate stats, track per-collection
// rates externally (e.g., from projectionhost processed counts).
func (s *Store) ObservedWorkloadStats() WorkloadStats {
	return s.meter.Stats()
}

// replayCost estimates the cost of replaying a stream to answer a query.
//
//	replay_cost = read_rate * avg_stream_length * fold_cost_per_event
func replayCost(stats WorkloadStats) float64 {
	return stats.ReadRatePerSec * stats.AvgStreamLength * foldCostPerEvent
}

// materializeCost estimates the cost of maintaining a materialized projection.
//
//	materialize_cost = write_rate * fold_cost_per_event + read_rate * query_cost_per_lookup
func materializeCost(stats WorkloadStats) float64 {
	return stats.WriteRatePerSec*foldCostPerEvent + stats.ReadRatePerSec*queryCostPerLookup
}

// shouldMaterialize returns true when the materialization cost is lower
// than the replay cost. When costs are equal, it defaults to materialize
// (materialized projections offer better read latency).
func shouldMaterialize(stats WorkloadStats) bool {
	return materializeCost(stats) <= replayCost(stats)
}

// materializeRule recommends whether each query's projection should be
// materialized or replayed on demand, based on observed workload stats.
// This is THE event-sourcing-specific killer feature: in ES, projections
// are planning decisions, not deployment facts.
type materializeRule struct {
	stats map[string]WorkloadStats
}

func (*materializeRule) Name() string { return "materialize-vs-replay" }

func (r *materializeRule) Apply(result *PlanResult, _ PlanContext) error {
	if len(r.stats) == 0 {
		return nil
	}

	for _, q := range result.Queries {
		stats, ok := r.stats[q.QueryName]
		if !ok {
			continue
		}

		if stats.WriteRatePerSec == 0 && stats.ReadRatePerSec == 0 {
			continue
		}

		rc := replayCost(stats)
		mc := materializeCost(stats)

		if shouldMaterialize(stats) {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Level: DiagLevelInfo,
				Query: q.QueryName,
				Message: fmt.Sprintf(
					"materialize recommended: replay_cost=%.2f > materialize_cost=%.2f "+
						"(writes=%.0f/s reads=%.0f/s avg_stream=%.0f)",
					rc, mc,
					stats.WriteRatePerSec, stats.ReadRatePerSec, stats.AvgStreamLength,
				),
			})
		} else {
			ratio := rc / math.Max(mc, 0.001)
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Level: DiagLevelWarn,
				Query: q.QueryName,
				Message: fmt.Sprintf(
					"replay may be cheaper: materialize_cost=%.2f is %.1fx replay_cost=%.2f "+
						"(writes=%.0f/s reads=%.0f/s avg_stream=%.0f) — "+
						"consider on-demand replay instead of materialization",
					mc, ratio, rc,
					stats.WriteRatePerSec, stats.ReadRatePerSec, stats.AvgStreamLength,
				),
			})
		}
	}

	return nil
}
