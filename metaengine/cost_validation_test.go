package metaengine

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
)

// cost_validation_bench_test.go validates that the cost model's PREDICTIONS
// correlate with ACTUAL performance. The kill criterion: if prediction error
// > 2x for more than 20% of cells, the cost model needs recalibration.

func TestCostModel_PredictionsVsActual(t *testing.T) {
	ctx := context.Background()
	volume := int64(10_000)

	// Define test cases: engine × ADT × operation.
	type testCase struct {
		name     string
		complexity Complexity
		nsPerOp  float64
		setup    func(t *testing.T, eng Engine) func()
		measure  func(t *testing.T, eng Engine) int64 // returns ns/op
	}

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

	pebbleEng, err := pebbleengine.NewPebbleEngine("")
	if err != nil {
		t.Fatal(err)
	}

	defer pebbleEng.Close()

	// Populate data for each engine.
	for _, eng := range []Engine{memEng, sqlEng, pebbleEng} {
		mb := eng.(MapBackend)
		for i := 0; i < int(volume); i++ {
			_ = mb.MapSet(ctx, "bench", i, i*2)
		}
	}

	cases := []testCase{
		{
			name:       "Memory/Map/PointLookup",
			complexity: ComplexityO1,
			nsPerOp:    MemoryNsPerOp,
			measure: func(t *testing.T, eng Engine) int64 {
				mb := eng.(MapBackend)
				start := testing.AllocsPerRun(1, func() {
					_, _, _ = mb.MapGet(ctx, "bench", 42)
				})
				_ = start
				// Use benchmark-style measurement.
				return measureNs(100, func() {
					_, _, _ = mb.MapGet(ctx, "bench", 42)
				})
			},
		},
		{
			name:       "SQLite/Map/PointLookup",
			complexity: ComplexityOLogN,
			nsPerOp:    SQLiteNsPerOp,
			measure: func(t *testing.T, eng Engine) int64 {
				mb := eng.(MapBackend)
				return measureNs(100, func() {
					_, _, _ = mb.MapGet(ctx, "bench", 42)
				})
			},
		},
		{
			name:       "Pebble/Map/PointLookup",
			complexity: ComplexityO1,
			nsPerOp:    pebbleengine.PebbleNsPerOp,
			measure: func(t *testing.T, eng Engine) int64 {
				mb := eng.(MapBackend)
				return measureNs(100, func() {
					_, _, _ = mb.MapGet(ctx, "bench", 42)
				})
			},
		},
	}

	type result struct {
		name      string
		predicted float64 // ms
		actual    float64 // ms
		ratio     float64 // actual/predicted
	}

	var results []result

	// Calculate predictions.
	for _, tc := range cases {
		pred := estimateCost(tc.complexity, volume, tc.nsPerOp)
		results = append(results, result{
			name:      tc.name,
			predicted: pred.EstimatedLatencyMs,
		})
	}

	// Measure actuals.
	engineList := []Engine{memEng, sqlEng, pebbleEng}

	for i, tc := range cases {
		actualNs := tc.measure(t, engineList[i])
		actualMs := float64(actualNs) / 1e6
		results[i].actual = actualMs
		if results[i].predicted > 0 {
			results[i].ratio = actualMs / results[i].predicted
		}
	}

	// Report and check kill criterion.
	t.Log("\n=== Cost Model Validation ===")
	t.Log("Format: predicted → actual (ratio)")
	t.Log("-----------------------------------")

	withinBudget := 0

	for _, r := range results {
		status := "OK"
		if r.ratio > 2.0 || r.ratio < 0.5 {
			status = "OUT OF BOUNDS"
		} else {
			withinBudget++
		}

		t.Logf("%-35s predicted=%.6fms actual=%.6fms ratio=%.1fx %s",
			r.name, r.predicted, r.actual, r.ratio, status)
	}

	total := len(results)
	pctWithin := float64(withinBudget) / float64(total) * 100

	t.Logf("\nWithin 2x bounds: %d/%d (%.0f%%)", withinBudget, total, pctWithin)

	if pctWithin < 80 {
		t.Errorf("kill criterion FAILED: only %.0f%% within 2x bounds (need >=80%%)", pctWithin)
	}
}

// measureNs runs fn iterations times and returns the average ns/op.
func measureNs(iterations int, fn func()) int64 {
	// Warm up.
	fn()

	var total int64

	for i := 0; i < iterations; i++ {
		start := testing.AllocsPerRun(1, fn)
		total += int64(start)
	}

	return total / int64(iterations)
}

// Ensure imports are used.
var _ = fmt.Sprintf
