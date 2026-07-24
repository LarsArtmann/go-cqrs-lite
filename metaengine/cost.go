package metaengine

import (
	"fmt"
	"math"
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

// estimateCost computes a cost estimate for a query with given complexity and volume.
// The volume represents the expected number of items in the projection.
// If volume is zero or negative, a default of 1000 is assumed.
func estimateCost(complexity Complexity, volume int64) CostEstimate {
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
		// Graph traversal: assume average branching factor 10, depth 2.
		ops = float64(10 * 10)
	default:
		ops = n
	}

	// Rough baseline: 100 nanoseconds per in-memory operation.
	const nsPerOp = 100.0

	latencyMs := (ops * nsPerOp) / 1e6

	return CostEstimate{
		Complexity:         complexity,
		Volume:             effectiveVolume,
		EstimatedOps:       ops,
		EstimatedLatencyMs: latencyMs,
	}
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
var scaleThresholds = map[ADT]ScaleThreshold{
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

// checkScaleThreshold returns a diagnostic if the volume falls outside
// the optimal range for the given ADT's default structure.
func checkScaleThreshold(adt ADT, volume int64) *Diagnostic {
	threshold, ok := scaleThresholds[adt]
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
	}

	return adtComplexity
}
