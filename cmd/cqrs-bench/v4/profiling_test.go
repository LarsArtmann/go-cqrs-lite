package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// profilingOrFail starts profiling for a test and fails when it cannot.
func profilingOrFail(t *testing.T, cpuProfile, memProfile string) func() {
	t.Helper()

	stop, err := startProfiling(cpuProfile, memProfile)
	if err != nil {
		t.Fatalf("startProfiling(%q, %q): %v", cpuProfile, memProfile, err)
	}

	return stop
}

// TestStartProfiling_TeardownOrderFlushesBeforeClose pins the LIFO teardown
// contract: the stop func must run pprof.StopCPUProfile BEFORE closing the
// cpu-profile file (flush-then-close) and write the heap profile to disk. A
// reordered teardown closes the file first and leaves an EMPTY cpu profile on
// disk, exactly the regression this guard catches, since a valid gzip pprof
// stream is impossible unless the profiler flushed before the close.
func TestStartProfiling_TeardownOrderFlushesBeforeClose(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cpuPath := filepath.Join(dir, "cpu.pprof")
	memPath := filepath.Join(dir, "mem.pprof")

	stop := profilingOrFail(t, cpuPath, memPath)

	sink := 0

	for i := range 1000 {
		sink += i
	}

	stop()

	assertGzipFile(t, cpuPath, "cpu profile")
	assertGzipFile(t, memPath, "mem profile")
}

// assertGzipFile fails when path is missing, empty, or not a gzip stream.
func assertGzipFile(t *testing.T, path, what string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", what, err)
	}

	if !bytes.HasPrefix(content, []byte{0x1f, 0x8b}) {
		t.Fatalf("%s at %s is not a gzipped pprof stream (teardown closed before flush?): %d bytes",
			what, path, len(content))
	}
}

// TestStartProfiling_EmptyFlagsIsNoOp pins that absent flags produce a
// callable, panic-free stop func touching no files.
func TestStartProfiling_EmptyFlagsIsNoOp(t *testing.T) {
	t.Parallel()

	profilingOrFail(t, "", "")()
}

// TestStartProfiling_UnwritablePathIsAnError: a profile that cannot start is
// reported, not exited from, so the caller's cleanup still runs.
func TestStartProfiling_UnwritablePathIsAnError(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "missing", "cpu.pprof")

	stop, err := startProfiling(missing, "")
	if !errors.Is(err, errFlagValue) {
		t.Fatalf("startProfiling with an unwritable path: error = %v, want errFlagValue", err)
	}

	if stop != nil {
		t.Error("a profiling that did not start must not return a stop func")
	}
}
