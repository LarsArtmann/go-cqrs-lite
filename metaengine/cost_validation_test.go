package metaengine

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// cost_validation_test.go validates that the cost model's PREDICTIONS
// correlate with ACTUAL performance for the Memory and SQLite engines.
// The Pebble engine has its own calibration benchmarks in pebbleengine/.
//
// Kill criterion: if prediction error > 2x for more than 20% of cells,
// the cost model needs recalibration — not tuning.

func TestCostModel_PredictionsVsActual(t *testing.T) {
	if testing.Short() {
		t.Skip("cost validation requires non-short mode")
	}

	ctx := context.Background()
	volume := int64(10_000)

	memEng := NewMemoryEngine()
	defer memEng.Close()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	sqlEng, err := NewSQLiteEngine(db)
	if err != nil {
		t.Fatal(err)
	}

	defer sqlEng.Close()

	// Populate data.
	mbMem := memEng.(MapBackend)
	mbSQL := sqlEng.(MapBackend)

	for i := 0; i < int(volume); i++ {
		_ = mbMem.MapSet(ctx, "bench", i, i*2)
		_ = mbSQL.MapSet(ctx, "bench", i, i*2)
	}

	type result struct {
		name      string
		predicted float64 // ms
		actual    float64 // ms
		ratio     float64 // actual/predicted
	}

	// Calculate predictions and measure actuals.
	memPred := estimateCost(ComplexityO1, volume, MemoryNsPerOp)
	sqlPred := estimateCost(ComplexityOLogN, volume, SQLiteNsPerOp)

	memActualNs := avgNs(200, func() {
		_, _, _ = mbMem.MapGet(ctx, "bench", 42)
	})

	sqlActualNs := avgNs(200, func() {
		_, _, _ = mbSQL.MapGet(ctx, "bench", 42)
	})

	results := []result{
		{
			name:      "Memory/Map/PointLookup",
			predicted: memPred.EstimatedLatencyMs,
			actual:    float64(memActualNs) / 1e6,
		},
		{
			name:      "SQLite/Map/PointLookup",
			predicted: sqlPred.EstimatedLatencyMs,
			actual:    float64(sqlActualNs) / 1e6,
		},
	}

	for i := range results {
		if results[i].predicted > 0 {
			results[i].ratio = results[i].actual / results[i].predicted
		}
	}

	// Report.
	t.Log("\n=== Cost Model Validation ===")
	t.Log("Format: predicted → actual (ratio)")

	withinBounds := 0

	for _, r := range results {
		status := "OK"
		if r.ratio > 2.0 || r.ratio < 0.5 {
			status = "OUT OF BOUNDS"
		} else {
			withinBounds++
		}

		t.Logf("  %-30s predicted=%.6fms actual=%.6fms ratio=%.1fx %s",
			r.name, r.predicted, r.actual, r.ratio, status)
	}

	pctWithin := float64(withinBounds) / float64(len(results)) * 100
	t.Logf("  Within 2x bounds: %d/%d (%.0f%%)", withinBounds, len(results), pctWithin)

	if pctWithin < 80 {
		t.Errorf("cost model validation FAILED: only %.0f%% within 2x bounds (need >=80%%)", pctWithin)
	}
}

// avgNs runs fn n times and returns the average ns/op.
func avgNs(n int, fn func()) int64 {
	fn() // warm up

	start := time.Now()

	for i := 0; i < n; i++ {
		fn()
	}

	elapsed := time.Since(start)

	return elapsed.Nanoseconds() / int64(n)
}
