package benchkit

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/stack/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

func TestRunSoak_Memory(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := RunSoak(ctx, SoakConfig{
		Duration:       3 * time.Second,
		ReportInterval: 0, // no intermediate output during test
		Config: Config{
			Profile:     ProfileDev,
			PayloadSize: 64,
			Backend:     "memory",
			SkipReads:   true,
			SkipRawSink: true,
		},
	}, func() (*stack.Bundle, error) {
		return memory.New()
	})
	if err != nil {
		t.Fatalf("RunSoak failed: %v", err)
	}

	if result.Iterations < 2 {
		t.Errorf("Iterations = %d, expected >= 2 in 3s", result.Iterations)
	}

	if len(result.Samples) != result.Iterations {
		t.Errorf("len(Samples) = %d, Iterations = %d, expected equal",
			len(result.Samples), result.Iterations)
	}

	// Each sample should have positive throughput.
	for i, s := range result.Samples {
		if s.Throughput <= 0 {
			t.Errorf("sample %d throughput = %.0f, expected positive", i, s.Throughput)
		}

		if s.TotalEvents == 0 {
			t.Errorf("sample %d TotalEvents = 0, expected positive", i)
		}
	}
}

func TestRunSoak_TrendsPopulated(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := RunSoak(ctx, SoakConfig{
		Duration: 2 * time.Second,
		Config: Config{
			Profile:     ProfileDev,
			PayloadSize: 64,
			Backend:     "memory",
			SkipReads:   true,
			SkipRawSink: true,
		},
	}, func() (*stack.Bundle, error) {
		return memory.New()
	})
	if err != nil {
		t.Fatalf("RunSoak failed: %v", err)
	}

	// Trends require at least 2 samples.
	if len(result.Samples) < 2 {
		t.Fatalf("need >= 2 samples for trend analysis, got %d", len(result.Samples))
	}

	// Heap leak rate should be near-zero for memory backend (no per-cycle leak).
	// The race detector inflates allocations ~5-10x, so allow more headroom.
	maxHeapLeak := float64(1 << 20) // 1MB/iteration
	if raceEnabled {
		maxHeapLeak = float64(16 << 20) // 16MB/iteration
	}

	if result.HeapLeakRate > maxHeapLeak {
		t.Errorf(
			"HeapLeakRate = %.0f bytes/iter, expected < %.0f",
			result.HeapLeakRate,
			maxHeapLeak,
		)
	}
}

func TestRunSoak_ProgressReport(t *testing.T) {
	t.Parallel()

	var progress bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := RunSoak(ctx, SoakConfig{
		Duration:       2 * time.Second,
		ReportInterval: 500 * time.Millisecond,
		ProgressWriter: &progress,
		Config: Config{
			Profile:     ProfileDev,
			PayloadSize: 64,
			Backend:     "memory",
			SkipReads:   true,
			SkipRawSink: true,
		},
	}, func() (*stack.Bundle, error) {
		return memory.New()
	})
	if err != nil {
		t.Fatalf("RunSoak failed: %v", err)
	}

	output := progress.String()
	if output == "" {
		t.Error("expected progress output, got empty string")
	}

	if !strings.Contains(output, "soak:") {
		t.Errorf("progress output missing 'soak:' prefix;\noutput: %s", output)
	}
}

func TestPrintSoakReport(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := RunSoak(ctx, SoakConfig{
		Duration: 1 * time.Second,
		Config: Config{
			Profile:     ProfileDev,
			PayloadSize: 64,
			Backend:     "memory",
			SkipReads:   true,
			SkipRawSink: true,
		},
	}, func() (*stack.Bundle, error) {
		return memory.New()
	})
	if err != nil {
		t.Fatalf("RunSoak failed: %v", err)
	}

	var buf bytes.Buffer
	PrintSoakReport(&buf, result)

	output := buf.String()
	for _, want := range []string{"Soak Test:", "Throughput:", "Write P99:", "Heap:", "Per-iteration"} {
		if !strings.Contains(output, want) {
			t.Errorf("soak report missing %q;\noutput: %s", want, output)
		}
	}
}

func TestWriteSoakJSON_RoundTrip(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	original, err := RunSoak(ctx, SoakConfig{
		Duration: 1 * time.Second,
		Config: Config{
			Profile:     ProfileDev,
			PayloadSize: 64,
			Backend:     "memory",
			SkipReads:   true,
			SkipRawSink: true,
		},
	}, func() (*stack.Bundle, error) {
		return memory.New()
	})
	if err != nil {
		t.Fatalf("RunSoak failed: %v", err)
	}

	if len(original.Samples) < 2 {
		t.Fatalf("need >= 2 samples for round-trip, got %d", len(original.Samples))
	}

	var buf bytes.Buffer
	if err := WriteSoakJSON(&buf, original); err != nil {
		t.Fatalf("WriteSoakJSON: %v", err)
	}

	var decoded SoakResult
	if err := json.Unmarshal(buf.Bytes(), &decoded, jsonOpts); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if decoded.Backend != original.Backend {
		t.Errorf("Backend: got %q, want %q", decoded.Backend, original.Backend)
	}

	if decoded.Iterations != original.Iterations {
		t.Errorf("Iterations: got %d, want %d", decoded.Iterations, original.Iterations)
	}

	if len(decoded.Samples) != len(original.Samples) {
		t.Fatalf("len(Samples): got %d, want %d", len(decoded.Samples), len(original.Samples))
	}

	for i, want := range original.Samples {
		got := decoded.Samples[i]
		if got.Iteration != want.Iteration {
			t.Errorf("sample %d Iteration: got %d, want %d", i, got.Iteration, want.Iteration)
		}

		if got.TotalEvents != want.TotalEvents {
			t.Errorf("sample %d TotalEvents: got %d, want %d", i, got.TotalEvents, want.TotalEvents)
		}

		if got.Duration != want.Duration {
			t.Errorf("sample %d Duration: got %s, want %s", i, got.Duration, want.Duration)
		}

		if got.WriteP99 != want.WriteP99 {
			t.Errorf("sample %d WriteP99: got %s, want %s", i, got.WriteP99, want.WriteP99)
		}
	}

	if decoded.HeapGrowthBytes != original.HeapGrowthBytes {
		t.Errorf(
			"HeapGrowthBytes: got %d, want %d",
			decoded.HeapGrowthBytes,
			original.HeapGrowthBytes,
		)
	}

	if decoded.ThroughputDriftPct != original.ThroughputDriftPct {
		t.Errorf("ThroughputDriftPct: got %f, want %f",
			decoded.ThroughputDriftPct, original.ThroughputDriftPct)
	}
}
