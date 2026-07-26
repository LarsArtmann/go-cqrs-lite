package benchkit

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

func TestRunSoak_Memory(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping soak test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := RunSoak(ctx, SoakConfig{
		Duration:       5 * time.Second,
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
		t.Errorf("Iterations = %d, expected >= 2 in 5s", result.Iterations)
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
	if testing.Short() {
		t.Skip("skipping soak test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := RunSoak(ctx, SoakConfig{
		Duration: 5 * time.Second,
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
		maxHeapLeak = float64(32 << 20) // 32MB/iteration (race detector inflates 5-10x)
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
	if testing.Short() {
		t.Skip("skipping soak test in short mode")
	}

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
	if testing.Short() {
		t.Skip("skipping soak test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := RunSoak(ctx, SoakConfig{
		Duration: 5 * time.Second,
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

	// When the new phases ran, the report should include their drift lines
	// and the per-iteration phase table. PrintSoakReport requires BOTH first
	// and last samples to have JourneyP99 > 0 before printing the drift line.
	if len(result.Samples) >= 2 &&
		result.Samples[0].JourneyP99 > 0 &&
		result.Samples[len(result.Samples)-1].JourneyP99 > 0 {
		if !strings.Contains(output, "Journey P99:") {
			t.Errorf("soak report missing 'Journey P99:' line;\noutput: %s", output)
		}

		if !strings.Contains(output, "Phase P99 per iteration:") {
			t.Errorf("soak report missing phase per-iteration table;\noutput: %s", output)
		}
	}
}

func TestWriteSoakJSON_RoundTrip(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping soak test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	original, err := RunSoak(ctx, SoakConfig{
		Duration: 5 * time.Second,
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

	// Verify the new-phase P99 fields round-trip per-sample.
	for i, want := range original.Samples {
		got := decoded.Samples[i]
		if got.JourneyP99 != want.JourneyP99 {
			t.Errorf("sample %d JourneyP99: got %s, want %s", i, got.JourneyP99, want.JourneyP99)
		}

		if got.QueryHitP99 != want.QueryHitP99 {
			t.Errorf("sample %d QueryHitP99: got %s, want %s", i, got.QueryHitP99, want.QueryHitP99)
		}

		if got.CacheHitP99 != want.CacheHitP99 {
			t.Errorf("sample %d CacheHitP99: got %s, want %s", i, got.CacheHitP99, want.CacheHitP99)
		}
	}
}

func TestConfig_CodecRoundTrip(t *testing.T) {
	t.Parallel()

	original := &SoakResult{
		Backend: "memory",
		Config: SoakConfig{
			Config: Config{
				Profile:     ProfileDev,
				PayloadSize: 64,
				Backend:     "memory",
				Codec:       codec.JSONCodec{},
			},
		},
	}

	var buf bytes.Buffer
	if err := WriteSoakJSON(&buf, original); err != nil {
		t.Fatalf("WriteSoakJSON: %v", err)
	}

	var decoded SoakResult
	if err := json.Unmarshal(buf.Bytes(), &decoded, jsonOpts); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if decoded.Config.Config.Codec == nil {
		t.Fatal("Config.Codec is nil after round-trip, expected JSONCodec")
	}

	if got := decoded.Config.Config.Codec.Encoding(); got != codec.EncodingJSON {
		t.Errorf("Config.Codec.Encoding(): got %q, want %q", got, codec.EncodingJSON)
	}
}
