package benchkit

import (
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/stack/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

func TestMixedWorkload_Memory(t *testing.T) {
	t.Parallel()

	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 64,
		Backend:     "memory",
	}, func() (*stack.Bundle, error) {
		return memory.New()
	})

	if result.MixedWorkload.WriteOps == 0 {
		t.Fatal("MixedWorkload.WriteOps = 0, want > 0")
	}

	if result.MixedWorkload.ReadOps == 0 {
		t.Fatal("MixedWorkload.ReadOps = 0, want > 0")
	}

	if result.MixedWorkload.WriteLatency.Count == 0 {
		t.Fatal("MixedWorkload.WriteLatency.Count = 0, want > 0")
	}

	if result.MixedWorkload.ReadLatency.Count == 0 {
		t.Fatal("MixedWorkload.ReadLatency.Count = 0, want > 0")
	}

	if result.MixedWorkload.Readers < 1 {
		t.Fatalf("MixedWorkload.Readers = %d, want >= 1", result.MixedWorkload.Readers)
	}
}

func TestMixedWorkload_SQLite(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 64,
		Backend:     "sqlite",
		DiskPath:    dir,
	}, func() (*stack.Bundle, error) {
		return sqlite.New(filepath.Join(dir, "mixed.db"))
	})

	if result.MixedWorkload.WriteOps == 0 {
		t.Fatal("MixedWorkload.WriteOps = 0, want > 0")
	}

	if result.MixedWorkload.WriteErrors > 0 {
		t.Logf("SQLite mixed workload had %d write errors (expected with single-conn pool)",
			result.MixedWorkload.WriteErrors)
	}
}

func TestMixedWorkload_SkipMixed(t *testing.T) {
	t.Parallel()

	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 64,
		Backend:     "memory",
		SkipMixed:   true,
	}, func() (*stack.Bundle, error) {
		return memory.New()
	})

	if result.MixedWorkload.WriteOps != 0 {
		t.Fatalf("SkipMixed: WriteOps = %d, want 0", result.MixedWorkload.WriteOps)
	}
}
