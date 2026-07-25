package benchkit

import (
	"bytes"
	"context"
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
	// Allow up to 1MB/iteration tolerance for GC noise.
	if result.HeapLeakRate > 1<<20 {
		t.Errorf("HeapLeakRate = %.0f bytes/iter, expected < 1MB", result.HeapLeakRate)
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
