package pgengine

import (
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Profile returns the cost profile for this Postgres engine.
func (e *pgEngine) Profile() metaengine.EngineProfile {
	p := metaengine.EngineProfile{
		Name:        "postgres",
		NsPerOp:     PG_NsPerOp,
		Persistence: metaengine.PersistencePersistent, // remote server — always survives
		// Postgres is a networked service. RequiresNetwork declares the structural
		// fact; NetworkRTT is a same-datacenter PRIOR replaced by a live probe
		// (SELECT 1) once ProbeEngine runs. See METAENGINE-LIVE-LATENCY-MODEL.md.
		RequiresNetwork: true,
		NetworkRTT:      PG_NetworkRTT,
		// Per-read-pattern calibrated costs (see calibration_bench_test.go).
		// Postgres has a real B-tree index on meta_map PK, so point lookups
		// are genuinely fast (unlike DuckDB's columnar scan). Scan/aggregation
		// per-row costs are lower than NsPerRead because a single query
		// amortizes connection setup across all rows.
		ReadCosts: metaengine.ReadCosts{
			// Indexed B-tree point lookup. Measured ~28K via Docker (inflated);
			// production (same-datacenter) ~5K. Matches PG_NsPerRead.
			NsPerPointLookup: 5_000,
			// Measured ~402 ns/row via Docker (BenchmarkCalibration_Postgres_PushdownScan).
			// JSONB WHERE pushdown + row decode.
			NsPerFilteredScan: 400,
			// Measured ~246 ns/row (BenchmarkCalibration_Postgres_CounterGet,
			// ephemeral local PG 2026-09-01). ADR-0133: ReadAggregate executes
			// CounterGet over meta_counter — SQL SUM (AggregateSum bench) documents
			// the typed AggregateReader path that bypasses the planner.
			NsPerAggregate: 250,
			// Measured ~805 ns/row via Docker (BenchmarkCalibration_Postgres_FullScan).
			// Full scan + Go-side JSON decode.
			NsPerScan: 800,
		},
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap:       metaengine.ComplexityOLogN,
			metaengine.ADTCounter:   metaengine.ComplexityO1,
			metaengine.ADTSortedMap: metaengine.ComplexityOLogN,
			metaengine.ADTSet:       metaengine.ComplexityON,
			metaengine.ADTGraph:     metaengine.ComplexityODegree, // native WITH RECURSIVE on meta_graph_edges
			metaengine.ADTLog:       metaengine.ComplexityON,
			metaengine.ADTMultimap:  metaengine.ComplexityON,
			metaengine.ADTVector:    metaengine.ComplexityON,
			metaengine.ADTSearch:    metaengine.ComplexityON,
			metaengine.ADTSpatial:   metaengine.ComplexityON,
			// ADR-0142: claimkit SQL claims (CTE SKIP LOCKED) and PK dedup
			// upserts are native indexed operations.
			metaengine.ADTDueClaim: metaengine.ComplexityOLogN,
			metaengine.ADTDedup:    metaengine.ComplexityOLogN,
		},
		DegradedADTs: map[metaengine.ADT]bool{
			metaengine.ADTSet:      true,
			metaengine.ADTLog:      true,
			metaengine.ADTMultimap: true,
			metaengine.ADTVector:   true,
			metaengine.ADTSearch:   true,
			metaengine.ADTSpatial:  true,
		},
		Layouts: map[metaengine.ADT]metaengine.StorageLayout{
			metaengine.ADTMap:       metaengine.LayoutRow,
			metaengine.ADTCounter:   metaengine.LayoutRow,
			metaengine.ADTSortedMap: metaengine.LayoutRow,
		},
	}
	e.ApplyCalibration(&p)

	return p
}
