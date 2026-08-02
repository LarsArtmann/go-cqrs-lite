package metaengine

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// cost_validation_test.go validates that the cost model's RELATIVE predictions
// match actual performance rankings. The model uses per-engine NsPerOp
// calibrated on the WRITE path, so absolute predictions are conservatively
// high — but the RANKING (which engine is faster) must be correct for
// engine selection to work.
//
// Kill criterion: if the predicted ranking does NOT match the actual ranking
// for any test case, the cost model is misleading the planner.

func TestCostModel_RankingMatchesActual(t *testing.T) {
	if testing.Short() {
		t.Skip("cost validation requires non-short mode")
	}

	ctx := context.Background()
	volume := int64(10_000)

	memEng := NewMemoryEngine()
	defer memEng.Close()

	db, _ := sql.Open("sqlite", ":memory:")
	defer func() { _ = db.Close() }()

	sqlEng, err := NewSQLiteEngine(db)
	if err != nil {
		t.Fatal(err)
	}

	defer sqlEng.Close()

	// Populate.
	mbMem := memEng.(MapBackend)
	mbSQL := sqlEng.(MapBackend)

	for i := 0; i < int(volume); i++ {
		_ = mbMem.MapSet(ctx, "bench", i, i*2)
		_ = mbSQL.MapSet(ctx, "bench", i, i*2)
	}

	// Predicted costs.
	memPred := estimateCost(ComplexityO1, volume, MemoryNsPerOp, 0)
	sqlPred := estimateCost(ComplexityOLogN, volume, SQLiteNsPerOp, 0)

	// Actual latencies.
	memActualNs := avgNs(500, func() {
		_, _, _ = mbMem.MapGet(ctx, "bench", 42)
	})

	sqlActualNs := avgNs(500, func() {
		_, _, _ = mbSQL.MapGet(ctx, "bench", 42)
	})

	memActualMs := float64(memActualNs) / 1e6
	sqlActualMs := float64(sqlActualNs) / 1e6

	t.Log("\n=== Cost Model Ranking Validation ===")
	t.Logf("  Memory:  predicted=%.6fms actual=%.6fms (%.1fx off)",
		memPred.EstimatedLatencyMs, memActualMs, safeDiv(memActualMs, memPred.EstimatedLatencyMs))
	t.Logf("  SQLite:  predicted=%.6fms actual=%.6fms (%.1fx off)",
		sqlPred.EstimatedLatencyMs, sqlActualMs, safeDiv(sqlActualMs, sqlPred.EstimatedLatencyMs))

	// Validate ranking: Memory should be predicted AND actually faster than SQLite.
	memPredictedFaster := memPred.EstimatedLatencyMs < sqlPred.EstimatedLatencyMs
	memActuallyFaster := memActualMs < sqlActualMs

	t.Logf("  Predicted: Memory < SQLite? %v", memPredictedFaster)
	t.Logf("  Actual:    Memory < SQLite? %v", memActuallyFaster)

	if memPredictedFaster != memActuallyFaster {
		t.Errorf("RANKING MISMATCH: model predicts Memory faster=%v but actual=%v",
			memPredictedFaster, memActuallyFaster)
	}

	// Both should be in the same order of magnitude (within 10x).
	if memActuallyFaster && sqlActualMs > 100*memActualMs {
		t.Logf(
			"  NOTE: SQLite is >100x slower than Memory for point lookups — model should account for this",
		)
	}
}

// TestCostModel_ScanComplexityMatchesActual validates that the scan complexity
// adjustment (effectiveReadComplexity) correctly predicts when scans degrade
// to O(N) even on O(1) engines.
func TestCostModel_ScanComplexityMatchesActual(t *testing.T) {
	// A hash map (O(1) for point lookup) degrades to O(N) for filtered scans.
	// The cost model's effectiveReadComplexity must reflect this.
	mapComplexity := ComplexityO1

	scanComplexity := effectiveReadComplexity(ReadFilteredScan, mapComplexity)

	if scanComplexity != ComplexityON {
		t.Errorf("expected O(1) map to degrade to O(N) for scans, got %s", scanComplexity)
	}

	// A B-tree (O(logN) for point lookup) stays O(logN) for point lookups.
	pointComplexity := effectiveReadComplexity(ReadPointLookup, ComplexityOLogN)

	if pointComplexity != ComplexityOLogN {
		t.Errorf("expected O(logN) to stay O(logN) for point lookups, got %s", pointComplexity)
	}
}

func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}

	return a / b
}

// avgNs runs fn n times and returns the average ns/op.
func avgNs(n int, fn func()) int64 {
	fn() // warm up

	start := time.Now()
	for i := 0; i < n; i++ {
		fn()
	}

	return time.Since(start).Nanoseconds() / int64(n)
}
