package pebbleengine

import (
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func (e *pebbleEngine) Profile() metaengine.EngineProfile {
	p := metaengine.EngineProfile{
		Name:        "pebble",
		NsPerOp:     PebbleNsPerOp,
		NsPerWrite:  PebbleNsPerWrite,
		Persistence: e.persistence,
		// Per-read-pattern calibrated costs (see calibration_bench_test.go;
		// measured 2026-08-30, medians of 3 runs on 32 threads, load ~3.8).
		// Pebble is a KV engine: filtered scans and aggregations have no SQL
		// pushdown, so they degrade to ScanRawValues with Go-side work — hence
		// per-row scan costs (~700-830ns) far above the point-lookup cost.
		ReadCosts: metaengine.ReadCosts{
			// Indexed LSM point lookup (BenchmarkCalibration_PebbleGet).
			NsPerPointLookup: PebbleNsPerRead,
			// ~833 ns/row (BenchmarkCalibration_Pebble_FilteredScan):
			// ScanRawValues + Go-side PassesFilterSpecs over 10K rows, ~50% match.
			NsPerFilteredScan: 830,
			// ~125 ns/row (BenchmarkCalibration_Pebble_CounterScan): CounterGet
			// prefix scan over 1K counters — the ReadAggregate path.
			NsPerAggregate: 125,
			// ~695 ns/row (BenchmarkCalibration_Pebble_FullScan): full
			// ScanRawValues, JSON decode of every row.
			NsPerScan: 700,
		},
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap:       metaengine.ComplexityO1, // LSM point read
			metaengine.ADTSet:       metaengine.ComplexityO1,
			metaengine.ADTCounter:   metaengine.ComplexityON, // CounterGet = prefix scan
			metaengine.ADTSortedMap: metaengine.ComplexityON, // O(limit) with sort index, O(N) fallback
			metaengine.ADTLog:       metaengine.ComplexityOLogN,
			metaengine.ADTMultimap:  metaengine.ComplexityOLogN,
			metaengine.ADTVector:    metaengine.ComplexityON,
			metaengine.ADTSearch:    metaengine.ComplexityON,
			metaengine.ADTSpatial:   metaengine.ComplexityON,
			// ADR-0142: Map-runtime claims scan the collection (O(N) candidate
			// scan, correct under single-writer RMW) — degraded, declared.
			metaengine.ADTDueClaim: metaengine.ComplexityON,
			metaengine.ADTDedup:    metaengine.ComplexityO1,
		},
		DegradedADTs: map[metaengine.ADT]bool{
			metaengine.ADTVector:   true,
			metaengine.ADTSearch:   true,
			metaengine.ADTSpatial:  true,
			metaengine.ADTDueClaim: true,
		},
		Layouts: map[metaengine.ADT]metaengine.StorageLayout{
			metaengine.ADTMap:       metaengine.LayoutLSM,
			metaengine.ADTSet:       metaengine.LayoutLSM,
			metaengine.ADTCounter:   metaengine.LayoutLSM,
			metaengine.ADTSortedMap: metaengine.LayoutLSM,
		},
	}
	e.ApplyCalibration(&p)

	return p
}
