package benchkit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/stack/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// errFactory always fails, fast — used to exercise sweep plumbing (ordering,
// parameter stamping, GOMAXPROCS restore) without paying for real benchmark
// runs. ScalingSweep synthesizes a FAILED row on factory error, so the sweep
// still returns one result per input value in order.
func errFactory() (*stack.Bundle, error) { return nil, errors.New("factory unavailable") }

// TestGOMAXPROCSSweep_RestoresOriginalAfterRun verifies the original GOMAXPROCS
// is restored even when every run fails. GOMAXPROCSSweep is NOT parallel-safe
// (it mutates a global), so this test is deliberately not t.Parallel.
func TestGOMAXPROCSSweep_RestoresOriginalAfterRun(t *testing.T) {
	before := runtime.GOMAXPROCS(0)

	_ = GOMAXPROCSSweep(
		context.Background(),
		Config{Profile: ProfileDev},
		errFactory,
		[]int{2, 4},
	)

	after := runtime.GOMAXPROCS(0)
	if before != after {
		t.Fatalf("GOMAXPROCS not restored after sweep: before=%d after=%d", before, after)
	}
}

// TestScalingSweep_PreservesInputOrder verifies results come back in the SAME
// order as the input values (not sorted). Callers rely on positional
// correspondence between values and results.
func TestScalingSweep_PreservesInputOrder(t *testing.T) {
	t.Parallel()
	values := []int{3, 1, 2, 5, 4}
	results := ScalingSweep(
		context.Background(),
		Config{Profile: ProfileDev},
		errFactory,
		"x",
		values,
		func(*Config, int) {},
	)
	if len(results) != len(values) {
		t.Fatalf("got %d results, want %d", len(results), len(values))
	}
	for i, v := range values {
		if results[i].Value != v {
			t.Fatalf("order broken at index %d: want value %d, got %d", i, v, results[i].Value)
		}
		if results[i].Result == nil || results[i].Result.Error == "" {
			t.Fatalf("index %d: expected synthesized FAILED row from erroring factory", i)
		}
	}
}

// TestWorkerSweep_ResultsMatchInputOrder verifies WorkerSweep stamps the
// "workers" parameter and preserves input order.
func TestWorkerSweep_ResultsMatchInputOrder(t *testing.T) {
	t.Parallel()
	workers := []int{4, 2, 8}
	results := WorkerSweep(
		context.Background(),
		Config{Profile: ProfileDev},
		errFactory,
		workers,
	)
	if len(results) != len(workers) {
		t.Fatalf("got %d results, want %d", len(results), len(workers))
	}
	for i, w := range workers {
		if results[i].Parameter != "workers" {
			t.Fatalf("index %d: Parameter=%q, want %q", i, results[i].Parameter, "workers")
		}
		if results[i].Value != w {
			t.Fatalf("index %d: Value=%d, want %d", i, results[i].Value, w)
		}
	}
}

// TestPrintSweep_HandlesMixedFailedAndSuccess verifies PrintSweep renders a
// mix of FAILED and successful rows without panicking.
func TestPrintSweep_HandlesMixedFailedAndSuccess(t *testing.T) {
	t.Parallel()
	results := []SweepResult{
		{Parameter: "workers", Value: 1, Result: &Result{Error: "boom"}},
		{Parameter: "workers", Value: 2, Result: &Result{WriteThroughput: 1234.5}},
	}
	var buf bytes.Buffer
	PrintSweep(&buf, results) // must not panic

	out := buf.String()
	if !strings.Contains(out, "FAILED") {
		t.Fatalf("output missing FAILED row:\n%s", out)
	}
	// formatFloat renders 1234.5 as "1.2K"; the success row must appear (not be
	// swallowed by the FAILED branch).
	if !strings.Contains(out, "1.2K") {
		t.Fatalf("output missing successful row throughput:\n%s", out)
	}
}

// TestRepeat_ReportsMedianWithSamples verifies that Config.Repeat>1 runs N
// times and annotates the median result with the sample set.
func TestRepeat_ReportsMedianWithSamples(t *testing.T) {
	result := mustRun(t, Config{
		Profile: Profile{Name: "tiny", Streams: 1, EventsPerStream: 2, BatchSize: 1},
		Repeat:  3,
	}, func() (*stack.Bundle, error) { return memory.New() })

	if result.RepeatCount != 3 {
		t.Fatalf("RepeatCount = %d, want 3", result.RepeatCount)
	}
	if len(result.RepeatSamples) != 3 {
		t.Fatalf("RepeatSamples len = %d, want 3", len(result.RepeatSamples))
	}
}

func TestBatchSizeSweep_StampParameter(t *testing.T) {
	t.Parallel()
	sizes := []int{10, 50, 100}
	results := BatchSizeSweep(
		context.Background(),
		Config{Profile: ProfileDev},
		errFactory,
		sizes,
	)
	if len(results) != len(sizes) {
		t.Fatalf("got %d results, want %d", len(results), len(sizes))
	}
	for i, r := range results {
		if r.Parameter != "batchSize" {
			t.Errorf("result[%d].Parameter = %q, want %q", i, r.Parameter, "batchSize")
		}
		if r.Value != sizes[i] {
			t.Errorf("result[%d].Value = %d, want %d", i, r.Value, sizes[i])
		}
	}
}

func TestStreamLengthSweep_StampParameter(t *testing.T) {
	t.Parallel()
	lengths := []int{5, 20, 80}
	results := StreamLengthSweep(
		context.Background(),
		Config{Profile: ProfileDev},
		errFactory,
		lengths,
	)
	if len(results) != len(lengths) {
		t.Fatalf("got %d results, want %d", len(results), len(lengths))
	}
	for i, r := range results {
		if r.Parameter != "streamLength" {
			t.Errorf("result[%d].Parameter = %q, want %q", i, r.Parameter, "streamLength")
		}
	}
}

func TestSortedSweepResults_NonMutating(t *testing.T) {
	t.Parallel()
	original := []SweepResult{
		{Parameter: "x", Value: 3},
		{Parameter: "x", Value: 1},
		{Parameter: "x", Value: 2},
	}
	sorted := SortedSweepResults(original)
	// Verify sorted ascending by Value.
	if sorted[0].Value != 1 || sorted[1].Value != 2 || sorted[2].Value != 3 {
		t.Fatalf("not sorted ascending: %d, %d, %d", sorted[0].Value, sorted[1].Value, sorted[2].Value)
	}
	// Verify original is NOT mutated.
	if original[0].Value != 3 {
		t.Errorf("original slice was mutated: original[0].Value = %d, want 3", original[0].Value)
	}
}

func TestWriteSweepJSON_ExportsValidJSON(t *testing.T) {
	t.Parallel()
	results := []SweepResult{
		{Parameter: "batchSize", Value: 10, Result: &Result{Error: "fail"}},
	}
	var buf bytes.Buffer
	if err := WriteSweepJSON(&buf, results); err != nil {
		t.Fatalf("WriteSweepJSON: %v", err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if len(decoded) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(decoded))
	}
	if decoded[0]["parameter"] != "batchSize" {
		t.Errorf("parameter = %v, want batchSize", decoded[0]["parameter"])
	}
}
