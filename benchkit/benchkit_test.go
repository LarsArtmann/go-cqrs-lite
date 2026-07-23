package benchkit

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/stack/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

func mustRun(t *testing.T, config Config, factory Factory) *Result {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := Run(ctx, config, factory)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	return result
}

func TestRun_Memory(t *testing.T) {
	t.Parallel()

	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 128,
		Warmup:      2,
	}, func() (*stack.Bundle, error) {
		return memory.New()
	})

	if result.Backend != "unknown" {
		t.Errorf("Backend = %q, want %q", result.Backend, "unknown")
	}

	if result.TotalEvents != ProfileDev.TotalEvents() {
		t.Errorf("TotalEvents = %d, want %d",
			result.TotalEvents, ProfileDev.TotalEvents())
	}

	if result.WriteLatency.Count == 0 {
		t.Error("WriteLatency.Count is 0, expected nonzero")
	}

	if result.WriteThroughput <= 0 {
		t.Error("WriteThroughput is 0, expected positive")
	}

	if result.LoadLatency.Count == 0 {
		t.Error("LoadLatency.Count is 0, expected nonzero")
	}

	if result.ReadModelSet.Count == 0 {
		t.Error("ReadModelSet.Count is 0, expected nonzero")
	}

	if result.Memory.Delta > 1<<30 { // > 1GB is suspicious for a dev profile
		t.Errorf("Memory.Delta = %d, expected < 1GB", result.Memory.Delta)
	}
}

func TestRun_Memory_WithDiskPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 128,
		Backend:     "memory",
		DiskPath:    dir,
	}, func() (*stack.Bundle, error) {
		return memory.New()
	})

	// Memory backend doesn't write to disk, so disk size should be 0
	if result.Disk.DatabaseBytes != 0 {
		t.Errorf("Disk.DatabaseBytes = %d, expected 0 for memory backend",
			result.Disk.DatabaseBytes)
	}

	// But event bytes should be tracked
	if result.Disk.EventBytes <= 0 {
		t.Error("Disk.EventBytes is 0, expected positive")
	}
}

func TestRun_SQLite(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 128,
		Backend:     "sqlite",
		DiskPath:    dir,
	}, func() (*stack.Bundle, error) {
		return sqlite.New(filepath.Join(dir, "bench.db"))
	})

	if result.Backend != "sqlite" {
		t.Errorf("Backend = %q, want %q", result.Backend, "sqlite")
	}

	if result.WriteLatency.Count == 0 {
		t.Error("WriteLatency.Count is 0")
	}

	if result.LoadLatency.Count == 0 {
		t.Error("LoadLatency.Count is 0")
	}

	// SQLite should have journal scanning support
	if result.ReadAllTime == 0 {
		t.Error("ReadAllTime is 0, expected journal support")
	}
}

func TestRun_SQLite_ReadModelAndJournal(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 64,
		Backend:     "sqlite",
		DiskPath:    dir,
	}, func() (*stack.Bundle, error) {
		return sqlite.New(filepath.Join(dir, "bench.db"))
	})

	if result.ReadModelGet.Count == 0 {
		t.Error("ReadModelGet.Count is 0")
	}

	if result.ReadModelSet.Count == 0 {
		t.Error("ReadModelSet.Count is 0")
	}

	if result.ReadAllTime <= 0 {
		t.Error("ReadAllTime should be positive")
	}
}

func TestCompare(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	results, err := Compare(ctx, Config{
		Profile:     ProfileDev,
		PayloadSize: 64,
	}, map[string]Factory{
		"memory": func() (*stack.Bundle, error) { return memory.New() },
		"sqlite": func() (*stack.Bundle, error) {
			return sqlite.New(filepath.Join(t.TempDir(), "cmp.db"))
		},
	})
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}

	if results["memory"] == nil || results["memory"].Error != "" {
		t.Error("memory result is missing or errored")
	}

	if results["sqlite"] == nil || results["sqlite"].Error != "" {
		t.Error("sqlite result is missing or errored")
	}
}

func TestCompare_WithFailure(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, err := Compare(ctx, Config{
		Profile:     ProfileDev,
		PayloadSize: 64,
	}, map[string]Factory{
		"good": func() (*stack.Bundle, error) { return memory.New() },
		"bad": func() (*stack.Bundle, error) {
			return nil, errTestFactory
		},
	})
	if err != nil {
		t.Fatalf("Compare should not fail entirely: %v", err)
	}

	if results["good"].Error != "" {
		t.Errorf("good result has error: %s", results["good"].Error)
	}

	if results["bad"].Error == "" {
		t.Error("bad result should have an error message")
	}
}

func TestPrintReport(t *testing.T) {
	t.Parallel()

	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 64,
		Backend:     "memory",
	}, func() (*stack.Bundle, error) { return memory.New() })

	var buf bytes.Buffer
	PrintReport(&buf, result)

	output := buf.String()
	if !strings.Contains(output, "Benchmark:") {
		t.Error("PrintReport output missing 'Benchmark:' header")
	}

	if !strings.Contains(output, "Write Performance:") {
		t.Error("PrintReport output missing 'Write Performance:' section")
	}
}

func TestPrintComparison(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, _ := Compare(ctx, Config{
		Profile:     ProfileDev,
		PayloadSize: 64,
	}, map[string]Factory{
		"memory": func() (*stack.Bundle, error) { return memory.New() },
	})

	var buf bytes.Buffer
	PrintComparison(&buf, results)

	output := buf.String()
	if !strings.Contains(output, "Backend Comparison") {
		t.Error("PrintComparison output missing header")
	}

	if !strings.Contains(output, "memory") {
		t.Error("PrintComparison output missing 'memory' row")
	}
}

func TestWriteJSON(t *testing.T) {
	t.Parallel()

	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 64,
		Backend:     "memory",
	}, func() (*stack.Bundle, error) { return memory.New() })

	var buf bytes.Buffer
	if err := WriteJSON(&buf, result); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	if !strings.Contains(buf.String(), `"backend": "memory"`) {
		t.Error("WriteJSON output missing backend field")
	}
}

func TestPrintMarkdown(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, _ := Compare(ctx, Config{
		Profile:     ProfileDev,
		PayloadSize: 64,
	}, map[string]Factory{
		"memory": func() (*stack.Bundle, error) { return memory.New() },
	})

	var buf bytes.Buffer
	PrintMarkdown(&buf, results)

	output := buf.String()
	if !strings.Contains(output, "| Backend |") {
		t.Error("PrintMarkdown output missing table header")
	}

	if !strings.Contains(output, "| memory |") {
		t.Error("PrintMarkdown output missing memory row")
	}
}

var errTestFactory = errTest("factory failed")

type errTest string

func (e errTest) Error() string { return string(e) }
