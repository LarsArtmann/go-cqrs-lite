package metaengine

// SQLiteNsPerOp is the calibrated per-operation cost for the SQLite engine.
// Calibrated via BenchmarkCalibration_SQLiteSet/Get on 2026-07-25 using
// in-memory modernc.org/sqlite (file::memory:):
//   - MapSet (INSERT): ~6,548 ns/op
//   - MapGet (SELECT): ~4,960 ns/op
//
// The value 7,000 ns is conservative for in-memory planning. Disk-backed
// SQLite adds I/O latency (10-50µs per op), but the planner is designed to
// prefer memory engines when they can serve the query at lower cost.
const SQLiteNsPerOp = 7000.0

// SQLiteEngineProfile returns the cost profile for a SQLite engine.
func SQLiteEngineProfile() EngineProfile {
	return EngineProfile{
		Name:    "sqlite",
		NsPerOp: SQLiteNsPerOp,
		// Per-read-pattern calibrated costs (measured 2026-08-30, medians of
		// 3 runs; benches live in metaengine/sqliteengine — the engine
		// constructor is module-side, the profile factory is core-side).
		// SQLite is the only embedded SQL engine: point lookups hit the PK
		// index; filtered scans push json_extract() WHERE down (still ~3.5x
		// the point-lookup cost because a 10K-row pushdown scan pays row
		// materialization per row); the aggregate price is CounterGet
		// (ADR-0133), NOT SQL SUM — typed Sum/Avg bypasses the planner.
		ReadCosts: ReadCosts{
			// ~3,102 ns (BenchmarkCalibration_SQLite_PointLookup): PK index
			// lookup + JSON decode via database/sql.
			NsPerPointLookup: 3_100,
			// ~1,075 ns/row (BenchmarkCalibration_SQLite_FilteredScan):
			// json_extract WHERE pushdown over 10K rows, ~50% match.
			NsPerFilteredScan: 1_080,
			// ~531 ns/row (BenchmarkCalibration_SQLite_CounterGet):
			// CounterGet over 1K counters — the ReadAggregate path.
			NsPerAggregate: 530,
			// ~1,238 ns/row (BenchmarkCalibration_SQLite_FullScan): full
			// MapScan, JSON decode of every row.
			NsPerScan: 1_240,
		},
		Supports: map[ADT]Complexity{
			ADTMap:     ComplexityOLogN,
			ADTSet:     ComplexityOLogN,
			ADTCounter: ComplexityO1,
			// ADTSortedMap: SQLite now supports PushdownMapScan with json_extract()
			// WHERE/ORDER BY/LIMIT, giving O(logN + k) instead of O(NlogN).
			// Without pushdown (closure-based FilterOn/SortOn), the fallback
			// MapScan path remains O(NlogN). The cost model uses O(logN) as the
			// best-case estimate — queries using FilterOnField/SortOnField achieve it.
			ADTSortedMap: ComplexityOLogN,
			ADTLog:       ComplexityOLogN,
			ADTMultimap:  ComplexityOLogN,
			ADTGraph:     ComplexityODegree, // native recursive CTE on meta_graph_edges
			ADTVector:    ComplexityON,
			ADTSearch:    ComplexityON,
			ADTSpatial:   ComplexityON,
			// ADR-0142 write-side capabilities (claimkit SQL runtime): claims
			// scan the (collection, due_at) index; dedup is a PK upsert.
			ADTDueClaim: ComplexityOLogN,
			ADTDedup:    ComplexityOLogN,
		},
		DegradedADTs: map[ADT]bool{
			ADTVector:  true,
			ADTSearch:  true,
			ADTSpatial: true,
		},
		Persistence: PersistencePersistent, // SQLite is file-backed by default
	}
}
