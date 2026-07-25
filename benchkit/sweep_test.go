package benchkit

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestScalingSweep_SynthesizesFailedWhenRunReturnsNil(t *testing.T) {
	t.Parallel()

	// A zero Profile fails Run's validation, so Run returns (nil, err) for every
	// data point without ever invoking the factory. Before the fix, ScalingSweep
	// stored a nil *Result, and any caller reading Result.Error (e.g. PrintSweep)
	// nil-dereferenced it. The fix synthesizes a FAILED stub instead.
	results := ScalingSweep(context.Background(), Config{}, nil,
		"workers", []int{1, 2, 4}, func(cfg *Config, v int) {
			cfg.Concurrency = v
		})

	if len(results) != 3 {
		t.Fatalf("expected 3 sweep results, got %d", len(results))
	}

	for i, r := range results {
		if r.Result == nil {
			t.Fatalf("result %d (%s=%d): Result is nil; expected synthesized FAILED stub",
				i, r.Parameter, r.Value)
		}

		if r.Result.Error == "" {
			t.Errorf("result %d (%s=%d): expected non-empty Error on failed run, got empty",
				i, r.Parameter, r.Value)
		}
	}

	// PrintSweep must not panic on the synthesized FAILED rows.
	var buf bytes.Buffer

	PrintSweep(&buf, results)

	if !strings.Contains(buf.String(), "FAILED") {
		t.Errorf("expected PrintSweep output to mark rows FAILED, got: %s", buf.String())
	}
}
