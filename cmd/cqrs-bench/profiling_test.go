package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestStartProfiling_TeardownOrderFlushesBeforeClose (benchkit polish-tail
// (c), 2026-09-29) pins the LIFO teardown contract: the stop func must run
// pprof.StopCPUProfile BEFORE closing the cpu-profile file (flush-then-close)
// and write the heap profile to disk. A reordered teardown closes the file
// first and leaves an EMPTY cpu profile on disk — exactly the regression
// this guard catches, since a valid gzip pprof stream is impossible unless
// the profiler flushed before the close.
func TestStartProfiling_TeardownOrderFlushesBeforeClose(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cpuPath := filepath.Join(dir, "cpu.pprof")
	memPath := filepath.Join(dir, "mem.pprof")

	stop := startProfiling(cpuPath, memPath)

	sink := 0
	for i := range 1000 {
		sink += i
	}

	stop()

	cpuBytes, err := os.ReadFile(cpuPath)
	if err != nil {
		t.Fatalf("read cpu profile: %v", err)
	}

	if !bytes.HasPrefix(cpuBytes, []byte{0x1f, 0x8b}) {
		t.Fatalf("cpu profile is not a gzipped pprof stream (teardown closed before flush?): %d bytes", len(cpuBytes))
	}

	memBytes, err := os.ReadFile(memPath)
	if err != nil {
		t.Fatalf("read mem profile: %v", err)
	}

	if !bytes.HasPrefix(memBytes, []byte{0x1f, 0x8b}) {
		t.Fatalf("mem profile is not a gzipped pprof stream: %d bytes", len(memBytes))
	}
}

// TestStartProfiling_EmptyFlagsIsNoOp pins that absent flags produce a
// callable, panic-free stop func touching no files.
func TestStartProfiling_EmptyFlagsIsNoOp(t *testing.T) {
	t.Parallel()

	startProfiling("", "")()
}
