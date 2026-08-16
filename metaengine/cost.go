// Package metaengine cost model for the planner.
//
// HONESTY NOTE: This is a rough first-order model, not a calibrated query
// optimizer. Per-engine cost constants come from calibration benchmarks
// (see calibration_bench_test.go in each engine module). Engines with
// ReadCosts express different costs per read pattern (point lookup vs scan
// vs aggregate) — critical for engines like DuckDB where these span 4000x.
// Engines without ReadCosts fall back to a single NsPerRead scalar.
//
// The model's purpose is to pick the obviously-right engine (O(1) memory vs
// O(N) scan, or point-lookup-optimized vs aggregation-optimized) and to flag
// when no engine meets a latency budget — not to make fine-grained decisions
// between two engines with similar profiles. Re-run calibration benchmarks on
// the target hardware before trusting absolute latency estimates; relative
// rankings are more stable.
package metaengine

import (
	"fmt"
	"math"
	"time"
)

// Graph-traversal defaults for ComplexityODegree cost estimation. These are
// rough averages for social-graph-like data; real graphs vary widely.
const (
	defaultGraphBranchingFactor = 10
	defaultGraphTraversalDepth  = 2
)

// CostEstimate is the estimated cost of serving a query on a given engine.
// The planner uses this to select the optimal engine and to detect when
// a query's latency requirements cannot be met.
type CostEstimate struct {
	Complexity         Complexity
	Volume             int64
	EstimatedOps       float64
	EstimatedLatencyMs float64
}

func (ce CostEstimate) String() string {
	return fmt.Sprintf(
		"%s @ vol=%d → ops=%.0f latency=%.3fms",
		ce.Complexity, ce.Volume, ce.EstimatedOps, ce.EstimatedLatencyMs,
	)
}

// WithinBudget returns true if the estimated latency is within the given budget (ms).
// A budget of 0 or less means unlimited (always within budget).
func (ce CostEstimate) WithinBudget(budgetMs int64) bool {
	if budgetMs <= 0 {
		return true
	}

	return ce.EstimatedLatencyMs <= float64(budgetMs)
}

// defaultNsPerOp is the fallback nanoseconds-per-operation cost used when an
// engine profile does not provide a calibrated value.
const defaultNsPerOp = 100.0

// estimateCost computes a cost estimate for a query with given complexity and volume.
// The volume represents the expected number of items in the projection.
// If volume is zero or negative, a default of 1000 is assumed — the planner
// also emits an INFO diagnostic so the assumption is visible, not silent.
// filterCount is the number of declarative filters on the query; each filter
// reduces the estimated rows touched via a selectivity discount.
// nsPerOp is the calibrated per-operation cost for the engine being evaluated.
// networkRTT is the fixed per-query network overhead (0 for in-process engines).
// It is additive: total_latency = (ops × nsPerOp / 1e6) + networkRTT.
func estimateCost(
	complexity Complexity,
	volume int64,
	filterCount int,
	nsPerOp float64,
	networkRTT time.Duration,
) CostEstimate {
	effectiveVolume := volume
	if effectiveVolume <= 0 {
		effectiveVolume = 1000
	}

	n := float64(effectiveVolume)

	var ops float64

	switch complexity {
	case ComplexityO1:
		ops = 1
	case ComplexityOLogN:
		ops = math.Log2(n)
	case ComplexityON:
		ops = n
	case ComplexityONLogN:
		ops = n * math.Log2(n)
	case ComplexityODegree:
		// Graph traversal: cost grows as branching^depth (nodes visited).
		// E.g. branching=10, depth=2 → 100 nodes. See constants above.
		ops = math.Pow(defaultGraphBranchingFactor, defaultGraphTraversalDepth)
	default:
		ops = n
	}

	if nsPerOp <= 0 {
		nsPerOp = defaultNsPerOp
	}

	latencyMs := (ops*nsPerOp)/1e6 + float64(networkRTT.Microseconds())/1e3

	return CostEstimate{
		Complexity:         complexity,
		Volume:             effectiveVolume,
		EstimatedOps:       ops,
		EstimatedLatencyMs: latencyMs,
	}
}

// filterSelectivity estimates the fraction of rows that survive all filters.
// Each equality filter is assumed to match ~10% of rows (selectivity 0.1).
// Multiple filters compose multiplicatively: 2 filters → 0.1 × 0.1 = 0.01.
// The result is clamped to a minimum of 0.001 (always at least 1 scan unit).
//
// This is a rough first-order heuristic, not a calibrated statistics engine.
// Real selectivity depends on data distribution and index availability.
// Used only for diagnostics — not applied to the routing cost, since
// selectivity discounts can flip engine ranking in unexpected ways.
func filterSelectivity(filterCount int) float64 {
	const perFilterSelectivity = 0.1
	const minSelectivity = 0.001

	s := math.Pow(perFilterSelectivity, float64(filterCount))
	if s < minSelectivity {
		return minSelectivity
	}

	return s
}

// ScaleThreshold describes the optimal cardinality range for a data structure.
// Outside this range, the planner emits a diagnostic warning.
type ScaleThreshold struct {
	Structure string
	MinItems  int64
	MaxItems  int64
}

// scaleThresholds provides empirical guidance for structure selection.
// These are conservative defaults derived from database engineering practice.
// Returned by a function to avoid a package-level mutable global (gochecknoglobals);
// the map is small and only consulted at plan time.
func scaleThresholds() map[ADT]ScaleThreshold {
	return map[ADT]ScaleThreshold{
		ADTMap: {
			Structure: "hash map",
			MinItems:  1,
			MaxItems:  10_000_000,
		},
		ADTSet: {
			Structure: "hash set",
			MinItems:  1,
			MaxItems:  10_000_000,
		},
		ADTCounter: {
			Structure: "counter map",
			MinItems:  1,
			MaxItems:  1_000, // few distinct counter keys expected
		},
		ADTGraph: {
			Structure: "adjacency list",
			MinItems:  1,
			MaxItems:  1_000_000,
		},
		ADTSortedMap: {
			Structure: "sorted map",
			MinItems:  1,
			MaxItems:  1_000_000,
		},
		ADTLog: {
			Structure: "append-only log",
			MinItems:  1,
			MaxItems:  math.MaxInt64, // logs are unbounded by design
		},
		ADTMultimap: {
			Structure: "multimap (map of slices)",
			MinItems:  1,
			MaxItems:  5_000_000,
		},
	}
}

// checkScaleThreshold returns a diagnostic if the volume falls outside
// the optimal range for the given ADT's default structure.
func checkScaleThreshold(adt ADT, volume int64) *Diagnostic {
	threshold, ok := scaleThresholds()[adt]
	if !ok {
		return nil
	}

	if volume <= 0 {
		return nil // no volume hint, skip check
	}

	if volume > threshold.MaxItems {
		return &Diagnostic{
			Level: DiagLevelWarn,
			Message: fmt.Sprintf(
				"volume %d exceeds optimal range for %s (max %d) — consider a disk-backed engine",
				volume, threshold.Structure, threshold.MaxItems,
			),
		}
	}

	return nil
}

// effectiveReadComplexity adjusts the ADT-level complexity for the actual read pattern.
// A hash map (O(1) for point lookup) still scans all items (O(N)) for filtered scans.
// This ensures cost estimates reflect real query costs, not just data structure costs.
func effectiveReadComplexity(readPattern ReadPattern, adtComplexity Complexity) Complexity {
	switch readPattern {
	case ReadFilteredScan, ReadScan:
		if adtComplexity == ComplexityO1 {
			return ComplexityON
		}
	case ReadPointLookup,
		ReadMembership,
		ReadAggregate,
		ReadTraversal,
		ReadMultiLookup,
		ReadLogTail:
		// Point/membership/aggregate/traversal/multi/log reads do not degrade the
		// ADT-level complexity — only scans force a full collection walk.
	}

	return adtComplexity
}
