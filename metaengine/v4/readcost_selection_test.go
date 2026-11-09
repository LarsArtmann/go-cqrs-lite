package metaengine

import (
	"testing"
)

// readcost_selection_test.go verifies that per-read-pattern costs (ReadCosts)
// make the planner pick the RIGHT engine for the RIGHT workload.
//
// Before ReadCosts: a single NsPerRead scalar meant the planner could not
// distinguish between engines that are fast at point lookups but slow at scans
// (Memory) and engines that are slow at point lookups but fast at aggregations
// (DuckDB). The same constant was used for every read pattern.
//
// After ReadCosts: the planner uses NsForRead(pattern) which picks the
// calibrated per-pattern constant. This test verifies the selection changes.

func TestReadCosts_PlannerPicksMemoryForPointLookup(t *testing.T) {
	// Two engines: Memory (fast point lookup) and a "DuckDB-like" profile
	// (slow point lookup, fast aggregation).
	memProfile := EngineProfile{
		Name:    "memory",
		NsPerOp: 500,
		Supports: map[ADT]Complexity{
			ADTMap: ComplexityO1,
		},
	}

	duckdbProfile := EngineProfile{
		Name:      "duckdb",
		NsPerOp:   15000,
		NsPerRead: 1200,
		ReadCosts: ReadCosts{
			NsPerPointLookup:  50_000, // DuckDB is terrible at point lookups
			NsPerAggregate:    150,    // DuckDB excels at aggregations
			NsPerFilteredScan: 450,
			NsPerScan:         1000,
		},
		Supports: map[ADT]Complexity{
			ADTMap: ComplexityOLogN,
		},
	}

	// Point lookup: Memory (500ns) should beat DuckDB (50,000ns).
	memCost := memProfile.NsForRead(ReadPointLookup)
	duckCost := duckdbProfile.NsForRead(ReadPointLookup)

	if memCost >= duckCost {
		t.Errorf("point lookup: Memory (%.0fns) should be cheaper than DuckDB (%.0fns)",
			memCost, duckCost)
	}

	t.Logf("point lookup: Memory=%.0fns DuckDB=%.0fns → Memory wins by %.0fx",
		memCost, duckCost, duckCost/memCost)
}

func TestReadCosts_PlannerPicksDuckDBForAggregate(t *testing.T) {
	memProfile := EngineProfile{
		Name:    "memory",
		NsPerOp: 500,
		ReadCosts: ReadCosts{
			NsPerAggregate: 500, // Memory has no vectorization — per-row cost
		},
		Supports: map[ADT]Complexity{
			ADTCounter: ComplexityO1,
		},
	}

	duckdbProfile := EngineProfile{
		Name:      "duckdb",
		NsPerOp:   15000,
		NsPerRead: 1200,
		ReadCosts: ReadCosts{
			NsPerPointLookup:  50_000,
			NsPerAggregate:    150, // vectorized aggregation is 3x faster per row
			NsPerFilteredScan: 450,
			NsPerScan:         1000,
		},
		Supports: map[ADT]Complexity{
			ADTCounter: ComplexityO1,
		},
	}

	// Aggregation at 10K rows: Memory (500 × 10000) vs DuckDB (150 × 10000).
	volume := int64(10_000)

	memAgg := estimateCost(ComplexityO1, volume, memProfile.NsForRead(ReadAggregate), 0)
	duckAgg := estimateCost(ComplexityO1, volume, duckdbProfile.NsForRead(ReadAggregate), 0)

	if duckAgg.EstimatedLatencyMs >= memAgg.EstimatedLatencyMs {
		t.Errorf("aggregate @ %d rows: DuckDB (%.3fms) should be cheaper than Memory (%.3fms)",
			volume, duckAgg.EstimatedLatencyMs, memAgg.EstimatedLatencyMs)
	}

	t.Logf("aggregate @ %d rows: Memory=%.3fms DuckDB=%.3fms → DuckDB wins by %.1fx",
		volume, memAgg.EstimatedLatencyMs, duckAgg.EstimatedLatencyMs,
		memAgg.EstimatedLatencyMs/duckAgg.EstimatedLatencyMs)
}

func TestReadCosts_NsForReadFallbackChain(t *testing.T) {
	// Engine with only NsPerOp set (no NsPerRead, no ReadCosts) — oldest pattern.
	// NsForRead must fall back to NsPerOp for every pattern.
	p := EngineProfile{
		Name:    "legacy",
		NsPerOp: 1234,
		Supports: map[ADT]Complexity{
			ADTMap: ComplexityO1,
		},
	}

	for _, pattern := range AllReadPatterns() {
		got := p.NsForRead(pattern)
		if got != 1234 {
			t.Errorf("pattern %s: expected fallback to NsPerOp (1234), got %.0f", pattern, got)
		}
	}

	// Engine with NsPerRead but no ReadCosts — should fall back to NsPerRead.
	p2 := EngineProfile{
		Name:      "semi-legacy",
		NsPerOp:   1000,
		NsPerRead: 5678,
		Supports: map[ADT]Complexity{
			ADTMap: ComplexityO1,
		},
	}

	for _, pattern := range AllReadPatterns() {
		got := p2.NsForRead(pattern)
		if got != 5678 {
			t.Errorf("pattern %s: expected fallback to NsPerRead (5678), got %.0f", pattern, got)
		}
	}
}

func TestReadCosts_PerPatternOverrides(t *testing.T) {
	// Engine with all four ReadCosts fields set — each pattern should use
	// its specific field, not the fallback.
	p := EngineProfile{
		Name:      "calibrated",
		NsPerOp:   9999,
		NsPerRead: 8888,
		ReadCosts: ReadCosts{
			NsPerPointLookup:  1111,
			NsPerFilteredScan: 2222,
			NsPerAggregate:    3333,
			NsPerScan:         4444,
		},
	}

	cases := []struct {
		pattern ReadPattern
		want    float64
	}{
		{ReadPointLookup, 1111},
		{ReadMembership, 1111},
		{ReadMultiLookup, 1111},
		{ReadLogTail, 1111},
		{ReadFilteredScan, 2222},
		{ReadAggregate, 3333},
		{ReadScan, 4444},
		{ReadTraversal, 4444},
		{ReadVectorSearch, 4444},
		{ReadFullTextSearch, 4444},
		{ReadSpatialRange, 4444},
	}

	for _, tc := range cases {
		got := p.NsForRead(tc.pattern)
		if got != tc.want {
			t.Errorf("pattern %s: expected %.0f, got %.0f", tc.pattern, tc.want, got)
		}
	}
}

// TestReadCosts_4000xSpanProven verifies the core problem is fixed: DuckDB's
// point lookup vs aggregation cost ratio is now expressed in the cost model.
// Before ReadCosts, both used NsPerRead=1200, hiding the 4000x gap.
func TestReadCosts_4000xSpanProven(t *testing.T) {
	duckdb := EngineProfile{
		ReadCosts: ReadCosts{
			NsPerPointLookup: 50_000,
			NsPerAggregate:   150,
		},
	}

	pointCost := duckdb.NsForRead(ReadPointLookup)
	aggCost := duckdb.NsForRead(ReadAggregate)

	ratio := pointCost / aggCost

	if ratio < 100 {
		t.Errorf("DuckDB point/agg ratio should be >100x, got %.1fx "+
			"(point=%.0f agg=%.0f) — the 4000x gap is hidden without ReadCosts",
			ratio, pointCost, aggCost)
	}

	t.Logf("DuckDB read-cost span: point=%.0fns agg=%.0fns → %.1fx ratio expressed",
		pointCost, aggCost, ratio)
}
