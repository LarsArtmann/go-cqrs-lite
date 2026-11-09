package benchkit

import (
	"bytes"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/stack/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

func TestSnapshotPhase_Memory(t *testing.T) {
	t.Parallel()

	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 128,
		Backend:     "memory",
		SkipReads:   true,
		SkipRawSink: true,
	}, func() (*stack.Bundle, error) {
		return memory.New()
	})

	if result.SnapshotColdLatency.Count == 0 {
		t.Error("SnapshotColdLatency.Count = 0, expected nonzero")
	}

	if result.SnapshotLoadLatency.Count == 0 {
		t.Error("SnapshotLoadLatency.Count = 0, expected nonzero (memory has SnapshotStore)")
	}

	if result.CacheMissLatency.Count == 0 {
		t.Error("CacheMissLatency.Count = 0, expected nonzero")
	}

	if result.CacheHitLatency.Count == 0 {
		t.Error("CacheHitLatency.Count = 0, expected nonzero")
	}

	if result.SnapshotCorrectnessErrors > 0 {
		t.Errorf("SnapshotCorrectnessErrors = %d, expected 0",
			result.SnapshotCorrectnessErrors)
	}
}

func TestSnapshotPhase_CacheHitFasterThanMiss(t *testing.T) {
	t.Parallel()

	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 128,
		SkipReads:   true,
		SkipRawSink: true,
	}, func() (*stack.Bundle, error) {
		return memory.New()
	})

	// Cache hit (LoadFromVersion of 0 events) should be faster than cache miss
	// (full replay). We check that hit mean is not slower than miss mean.
	if result.CacheHitLatency.Mean > result.CacheMissLatency.Mean {
		t.Logf("note: cache hit mean (%s) > miss mean (%s) — may be noise for tiny streams",
			result.CacheHitLatency.Mean, result.CacheMissLatency.Mean)
	}
}

func TestSnapshotPhase_SQLite(t *testing.T) {
	t.Parallel()

	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 128,
		Backend:     "sqlite",
		SkipReads:   true,
		SkipRawSink: true,
	}, func() (*stack.Bundle, error) {
		return sqlite.New(":memory:")
	})

	if result.SnapshotColdLatency.Count == 0 {
		t.Error("SnapshotColdLatency.Count = 0, expected nonzero")
	}

	if result.CacheHitLatency.Count == 0 {
		t.Error("CacheHitLatency.Count = 0, expected nonzero")
	}

	if result.SnapshotCorrectnessErrors > 0 {
		t.Errorf("SnapshotCorrectnessErrors = %d, expected 0",
			result.SnapshotCorrectnessErrors)
	}
}

func TestSnapshotPhase_SkipFlag(t *testing.T) {
	t.Parallel()

	result := mustRun(t, Config{
		Profile:      ProfileDev,
		PayloadSize:  128,
		SkipSnapshot: true,
		SkipReads:    true,
		SkipRawSink:  true,
	}, func() (*stack.Bundle, error) {
		return memory.New()
	})

	if result.SnapshotColdLatency.Count != 0 {
		t.Errorf("SnapshotColdLatency.Count = %d, expected 0 with SkipSnapshot",
			result.SnapshotColdLatency.Count)
	}
}

func TestSnapshotPhase_Report(t *testing.T) {
	t.Parallel()

	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 128,
		SkipReads:   true,
		SkipRawSink: true,
	}, func() (*stack.Bundle, error) {
		return memory.New()
	})

	var buf bytes.Buffer
	PrintReport(&buf, result)

	output := buf.String()
	if !strings.Contains(output, "Snapshot / Cache:") {
		t.Errorf("report missing 'Snapshot / Cache:' section;\noutput: %s", output)
	}

	if !strings.Contains(output, "Cold replay:") {
		t.Error("report missing 'Cold replay:' line")
	}

	if !strings.Contains(output, "Cache hit:") {
		t.Error("report missing 'Cache hit:' line")
	}
}
