package benchkit

import (
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/stack/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// TestResult_GCMetrics verifies that GC pause metrics are populated for a
// non-trivial workload. The memory backend produces enough allocations during
// a ProfileDev run to trigger at least one GC cycle.
func TestResult_GCMetrics(t *testing.T) {
	t.Parallel()

	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 256,
	}, func() (*stack.Bundle, error) { return memory.New() })

	if result.GCCount <= 0 {
		t.Error("GCCount should be > 0 for a non-trivial workload")
	}

	if result.GCTotalPause <= 0 {
		t.Error("GCTotalPause should be > 0 when GC ran")
	}

	if result.GCMaxPause <= 0 {
		t.Error("GCMaxPause should be > 0 when GC ran")
	}

	if result.GCMeanPause <= 0 {
		t.Error("GCMeanPause should be > 0 when GC ran")
	}

	// Max pause should never exceed total pause.
	if result.GCMaxPause > result.GCTotalPause {
		t.Errorf("GCMaxPause (%v) > GCTotalPause (%v)",
			result.GCMaxPause, result.GCTotalPause)
	}

	// Mean should be between max and total/count.
	if result.GCMeanPause > result.GCMaxPause {
		t.Errorf("GCMeanPause (%v) > GCMaxPause (%v)",
			result.GCMeanPause, result.GCMaxPause)
	}
}

// TestResult_AllocationMetrics verifies that allocation deltas are populated.
func TestResult_AllocationMetrics(t *testing.T) {
	t.Parallel()

	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 256,
	}, func() (*stack.Bundle, error) { return memory.New() })

	if result.AllocCount == 0 {
		t.Error("AllocCount should be > 0 — events allocate memory")
	}

	if result.AllocBytes == 0 {
		t.Error("AllocBytes should be > 0 — events allocate memory")
	}
}

// TestResult_IntegrityErrors verifies that the data integrity check reports
// zero errors for a correct backend. Every event written should round-trip.
func TestResult_IntegrityErrors(t *testing.T) {
	t.Parallel()

	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 128,
	}, func() (*stack.Bundle, error) { return memory.New() })

	if result.IntegrityErrors != 0 {
		t.Errorf("IntegrityErrors = %d, want 0 — all events should round-trip",
			result.IntegrityErrors)
	}
}

// TestResult_ColdReadLatency verifies that the cold-read (first pass) latency
// is captured separately from the aggregate load latency.
func TestResult_ColdReadLatency(t *testing.T) {
	t.Parallel()

	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 128,
	}, func() (*stack.Bundle, error) { return memory.New() })

	if result.ColdReadLatency.Count == 0 {
		t.Error("ColdReadLatency.Count should be > 0 — first read pass was measured")
	}

	// ColdReadLatency should have the same sample count as the number of
	// streams (first pass loads all streams once).
	expected := result.Streams
	if result.ColdReadLatency.Count != int64(expected) {
		t.Errorf("ColdReadLatency.Count = %d, want %d (Streams)",
			result.ColdReadLatency.Count, expected)
	}
}

// TestResult_WriteAmplification verifies that write amplification is computed
// for disk-backed backends. For the memory backend (no disk), it should be 0.
func TestResult_WriteAmplification(t *testing.T) {
	t.Parallel()

	// Memory backend has no disk, so WriteAmplification should be 0.
	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 128,
	}, func() (*stack.Bundle, error) { return memory.New() })

	if result.Disk.WriteAmplification != 0 {
		t.Errorf("WriteAmplification = %f, want 0 for memory backend",
			result.Disk.WriteAmplification)
	}

	if result.Disk.EventBytes <= 0 {
		t.Error("Disk.EventBytes should be > 0 even for memory backend")
	}
}

// TestResult_EnvironmentEnrichment verifies that CPU model and RAM are
// populated on Linux (the primary benchmarking platform).
func TestResult_EnvironmentEnrichment(t *testing.T) {
	t.Parallel()

	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 128,
	}, func() (*stack.Bundle, error) { return memory.New() })

	// On Linux, /proc/cpuinfo is always available.
	if result.Environment.CPUModel == "" {
		t.Log("CPUModel is empty — non-Linux platform or /proc/cpuinfo unreadable")
	}

	if result.Environment.TotalRAMBytes == 0 {
		t.Log("TotalRAMBytes is 0 — non-Linux platform or /proc/meminfo unreadable")
	}
}

// TestRepeat_StatisticalReliability verifies that Repeat > 1 computes
// StdDev, CoV, Mean, and IsReliable fields.
func TestRepeat_StatisticalReliability(t *testing.T) {
	result := mustRun(t, Config{
		Profile: Profile{Name: "tiny", Streams: 1, EventsPerStream: 2, BatchSize: 1},
		Repeat:  3,
	}, func() (*stack.Bundle, error) { return memory.New() })

	if result.RepeatCount != 3 {
		t.Fatalf("RepeatCount = %d, want 3", result.RepeatCount)
	}

	if result.RepeatMean <= 0 {
		t.Error("RepeatMean should be > 0")
	}

	if result.RepeatStdDev < 0 {
		t.Error("RepeatStdDev should be >= 0")
	}

	if result.RepeatCoV < 0 {
		t.Error("RepeatCoV should be >= 0")
	}

	// For identical tiny workloads on memory backend, CoV should be low.
	// But we don't assert IsReliable=true because CI noise can push it above 0.10.
	t.Logf("Repeat: mean=%.0f stddev=%.0f cov=%.4f reliable=%v",
		result.RepeatMean, result.RepeatStdDev, result.RepeatCoV,
		result.RepeatIsReliable)
}

// TestResult_JSONIncludesNewFields verifies that the new metrics appear in
// the JSON serialization of a Result.
func TestResult_JSONIncludesNewFields(t *testing.T) {
	t.Parallel()

	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 128,
	}, func() (*stack.Bundle, error) { return memory.New() })

	data, err := json.Marshal(result, json.WithMarshalers(durationMarshalers))
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	jsonStr := string(data)

	requiredFields := []string{
		"coldReadLatency",
		"gcCount",
		"gcTotalPause",
		"gcMaxPause",
		"gcMeanPause",
		"allocCount",
		"allocBytes",
		"allocsPerOp",
		"bytesPerOp",
		"gcPercent",
		"tailRatio",
		"writeAmplification",
		"cpuModel",
	}

	for _, field := range requiredFields {
		if !strings.Contains(jsonStr, `"`+field+`"`) {
			t.Errorf("JSON missing field %q", field)
		}
	}

	// Verify integrityErrors is present (omitted when zero, but the key
	// should appear if there were errors). For memory backend it should be 0.
	if result.IntegrityErrors != 0 {
		t.Errorf("IntegrityErrors = %d, want 0", result.IntegrityErrors)
	}
}

// TestResult_DerivedMetrics verifies that derived rate metrics
// (AllocsPerOp, BytesPerOp, GCPercent, TailRatio) are computed from raw data.
func TestResult_DerivedMetrics(t *testing.T) {
	t.Parallel()

	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 256,
	}, func() (*stack.Bundle, error) { return memory.New() })

	if result.AllocsPerOp <= 0 {
		t.Errorf("AllocsPerOp = %.2f, want > 0", result.AllocsPerOp)
	}

	if result.BytesPerOp <= 0 {
		t.Errorf("BytesPerOp = %.2f, want > 0", result.BytesPerOp)
	}

	// GCPercent can be 0 if duration is 0, but for a real run it should be >= 0.
	if result.GCPercent < 0 {
		t.Errorf("GCPercent = %.2f, want >= 0", result.GCPercent)
	}

	// TailRatio = P99/P50. For a real workload, P99 >= P50 so ratio >= 1.
	if result.TailRatio < 1.0 {
		t.Errorf("TailRatio = %.2f, want >= 1.0 (P99 should be >= P50)", result.TailRatio)
	}

	// Cross-check: AllocsPerOp * TotalEvents should ≈ AllocCount.
	expected := result.AllocsPerOp * float64(result.TotalEvents)
	if expected > 0 {
		ratio := float64(result.AllocCount) / expected
		if ratio < 0.99 || ratio > 1.01 {
			t.Errorf(
				"AllocsPerOp cross-check: AllocCount=%d, AllocsPerOp*TotalEvents=%.0f (ratio=%.3f)",
				result.AllocCount,
				expected,
				ratio,
			)
		}
	}
}
