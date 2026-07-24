package benchkit

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"sync/atomic"
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

	var factoryCalls atomic.Int32

	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 128,
		Warmup:      2,
	}, func() (*stack.Bundle, error) {
		factoryCalls.Add(1)
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

	// Warmup uses a separate Bundle, so factory is called twice:
	// once for measurement, once for warmup.
	if got := factoryCalls.Load(); got != 2 {
		t.Errorf("factory called %d times, want 2 (measurement + separate warmup bundle)", got)
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

func TestRun_FactoryError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := Run(ctx, Config{Profile: ProfileDev}, func() (*stack.Bundle, error) {
		return nil, errTestFactory
	})

	if err == nil {
		t.Fatal("expected error from factory failure, got nil")
	}

	if !strings.Contains(err.Error(), "factory") {
		t.Errorf("error should mention factory, got: %v", err)
	}
}

func TestRun_NilBundle(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := Run(ctx, Config{Profile: ProfileDev}, func() (*stack.Bundle, error) {
		return nil, nil
	})

	if err == nil {
		t.Fatal("expected error for nil bundle, got nil")
	}

	if !strings.Contains(err.Error(), "nil bundle") {
		t.Errorf("error should mention nil bundle, got: %v", err)
	}
}

func TestRun_NilEventSink(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := Run(ctx, Config{Profile: ProfileDev}, func() (*stack.Bundle, error) {
		b, _ := memory.New()
		b.EventSink = nil

		return b, nil
	})

	if err == nil {
		t.Fatal("expected error for nil EventSink, got nil")
	}

	if !strings.Contains(err.Error(), "EventSink") {
		t.Errorf("error should mention EventSink, got: %v", err)
	}
}

func TestRun_NilEventSource(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := Run(ctx, Config{Profile: ProfileDev}, func() (*stack.Bundle, error) {
		b, _ := memory.New()
		b.EventSource = nil

		return b, nil
	})

	if err == nil {
		t.Fatal("expected error for nil EventSource, got nil")
	}

	if !strings.Contains(err.Error(), "EventSource") {
		t.Errorf("error should mention EventSource, got: %v", err)
	}
}

func TestRun_ClosedStore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := Run(ctx, Config{
		Profile:     ProfileDev,
		PayloadSize: 64,
	}, func() (*stack.Bundle, error) {
		b, bErr := sqlite.New(filepath.Join(dir, "closed.db"))
		if bErr != nil {
			return nil, bErr
		}

		_ = b.Close()

		return b, nil
	})

	if err == nil {
		t.Fatal("expected error from closed store, got nil")
	}
}

func TestRun_DurationAborts(t *testing.T) {
	t.Parallel()

	// Use a profile large enough that it cannot finish within the duration.
	bigProfile := Profile{
		Name: "test-duration", Streams: 100_000, EventsPerStream: 10,
		Concurrency: 1, ReadRatio: 0.5, BatchSize: 1,
	}

	start := time.Now()

	result := mustRun(t, Config{
		Profile:     bigProfile,
		PayloadSize: 64,
		Duration:    5 * time.Millisecond,
	}, func() (*stack.Bundle, error) { return memory.New() })

	elapsed := time.Since(start)

	// The run should finish well under the time needed for the full 1M events.
	if elapsed > 2*time.Second {
		t.Errorf("run took %v with Duration=5ms; expected < 2s", elapsed)
	}

	// The duration cap should prevent all 1M events from being written.
	if result.TotalEvents >= bigProfile.TotalEvents() {
		t.Errorf("TotalEvents = %d, expected < %d (duration should have limited writes)",
			result.TotalEvents, bigProfile.TotalEvents())
	}
}

func TestRun_CancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled before the run starts

	start := time.Now()

	_, err := Run(ctx, Config{
		Profile:     ProfileDev,
		PayloadSize: 64,
	}, func() (*stack.Bundle, error) { return memory.New() })

	elapsed := time.Since(start)

	// The run must not hang — it should complete quickly regardless of
	// whether individual phases see the cancellation.
	if elapsed > 5*time.Second {
		t.Errorf("run with cancelled context took %v, expected < 5s", elapsed)
	}

	// An error is expected: with a cancelled context, writes may not complete,
	// causing the read phase to fail on missing aggregates.
	if err == nil {
		// Some writes may squeeze through before cancellation is noticed;
		// that's fine. The important thing is no hang or panic.
		return
	}
}

func TestReadRatio(t *testing.T) {
	t.Parallel()

	// Use the same profile base but vary ReadRatio to verify the number of
	// read passes changes the LoadLatency sample count.
	base := Profile{
		Name: "test-ratio", Streams: 50, EventsPerStream: 5,
		Concurrency: 1, ReadRatio: 0.1, BatchSize: 1,
	}

	writeHeavy := base
	writeHeavy.ReadRatio = 0.1 // readPassesFor → 1 pass

	readHeavy := base
	readHeavy.ReadRatio = 0.8 // readPassesFor → 8 passes

	whResult := mustRun(t, Config{Profile: writeHeavy, PayloadSize: 64},
		func() (*stack.Bundle, error) { return memory.New() })

	rhResult := mustRun(t, Config{Profile: readHeavy, PayloadSize: 64},
		func() (*stack.Bundle, error) { return memory.New() })

	// WriteHeavy (1 pass) → 50 reads; ReadHeavy (8 passes) → 400 reads.
	whExpected := int64(writeHeavy.Streams * 1)
	rhExpected := int64(readHeavy.Streams * 8)

	if whResult.LoadLatency.Count != whExpected {
		t.Errorf("WriteHeavy LoadLatency.Count = %d, want %d (1 pass × %d streams)",
			whResult.LoadLatency.Count, whExpected, writeHeavy.Streams)
	}

	if rhResult.LoadLatency.Count != rhExpected {
		t.Errorf("ReadHeavy LoadLatency.Count = %d, want %d (8 passes × %d streams)",
			rhResult.LoadLatency.Count, rhExpected, readHeavy.Streams)
	}

	if rhResult.LoadLatency.Count <= whResult.LoadLatency.Count {
		t.Errorf("ReadHeavy reads (%d) should exceed WriteHeavy reads (%d)",
			rhResult.LoadLatency.Count, whResult.LoadLatency.Count)
	}
}

func TestReadPassesFor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ratio float64
		want  int
	}{
		{0.0, 1},
		{0.1, 1},
		{0.2, 2},
		{0.3, 3},
		{0.5, 5},
		{0.8, 8},
		{1.0, 10},
	}

	for _, tt := range tests {
		got := readPassesFor(tt.ratio)
		if got != tt.want {
			t.Errorf("readPassesFor(%.1f) = %d, want %d", tt.ratio, got, tt.want)
		}
	}
}

var errTestFactory = errTest("factory failed")

type errTest string

func (e errTest) Error() string { return string(e) }
