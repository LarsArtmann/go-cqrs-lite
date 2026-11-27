package main

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// startBench starts cqrs-bench in the background and returns its command and
// a pipe on its stderr.
func startBench(t *testing.T, scratch string, args ...string) (*exec.Cmd, io.ReadCloser) {
	t.Helper()

	cmd := exec.Command(benchBin, args...)
	cmd.Env = benchEnv(scratch)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("start cqrs-bench: %v", err)
	}

	return cmd, stderr
}

// TestCLI_FailingRunLeavesNoTemporaryStore pins the fix for handlers that
// exited the process on failure: a validation error found after the store
// was created used to leave that store behind.
func TestCLI_FailingRunLeavesNoTemporaryStore(t *testing.T) {
	t.Parallel()

	scratch := t.TempDir()

	run := runBench(scratch, "run", "--backend", "sqlite", "--profile", "dev", "--payload-sizes", "64")

	if !run.failed() {
		t.Fatal("--payload-sizes with one size should fail")
	}

	if !strings.Contains(run.stderr, "--payload-sizes") {
		t.Errorf("error should name the flag:\n%s", run.stderr)
	}

	assertNoLeakedStores(t, scratch)
}

// TestCLI_InterruptedRunCleansUp: Ctrl-C must exit non-zero, say so, remove
// the temporary store, and still flush the profiles.
func TestCLI_InterruptedRunCleansUp(t *testing.T) {
	t.Parallel()

	scratch := t.TempDir()

	cpuPath := filepath.Join(t.TempDir(), "cpu.pprof")
	memPath := filepath.Join(t.TempDir(), "mem.pprof")

	cmd, stderr := startBench(t, scratch,
		"run", "--backend", "sqlite", "--profile", "small", "--repeat", "20",
		"--progress", "100ms", "--cpuprofile", cpuPath, "--memprofile", memPath)

	// The first progress line means the store exists and is being written.
	scanner := bufio.NewScanner(stderr)
	if !scanner.Scan() {
		_ = cmd.Process.Kill()

		t.Fatalf("cqrs-bench printed no progress before it ended: %v", cmd.Wait())
	}

	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		_ = cmd.Process.Kill()

		t.Fatalf("interrupt cqrs-bench: %v", err)
	}

	rest, _ := io.ReadAll(stderr)

	if err := cmd.Wait(); err == nil {
		t.Error("an interrupted run must exit non-zero")
	}

	if output := scanner.Text() + string(rest); !mentions(output, "interrupted") {
		t.Errorf("an interrupted run should say so:\n%s", output)
	}

	assertNoLeakedStores(t, scratch)
	assertGzipFile(t, cpuPath, "cpu profile")
	assertGzipFile(t, memPath, "mem profile")
}

// TestCLI_UnwritableOutputFailsBeforeBenchmark: an --output that cannot be
// written is reported up front instead of after the whole benchmark ran.
func TestCLI_UnwritableOutputFailsBeforeBenchmark(t *testing.T) {
	t.Parallel()

	scratch := t.TempDir()
	report := filepath.Join(t.TempDir(), "missing", "report.json")

	run := runBench(scratch, "run", "--backend", "sqlite", "--profile", "dev", "--output", report)

	if !run.failed() {
		t.Fatal("--output into a missing directory should fail")
	}

	if !strings.Contains(run.stderr, "--output") {
		t.Errorf("error should name the flag:\n%s", run.stderr)
	}

	if run.stdout != "" {
		t.Errorf("no benchmark should have run:\n%s", run.stdout)
	}

	assertNoLeakedStores(t, scratch)
}

// TestCLI_TimeBudgetFlushesProfiles: a run that outlives --timeout fails
// with an error that names the flag, and its profiles are complete.
func TestCLI_TimeBudgetFlushesProfiles(t *testing.T) {
	t.Parallel()

	cpuPath := filepath.Join(t.TempDir(), "cpu.pprof")
	memPath := filepath.Join(t.TempDir(), "mem.pprof")

	run := runBench(t.TempDir(), "run", "--backend", "memory", "--profile", "dev",
		"--timeout", "1ns", "--cpuprofile", cpuPath, "--memprofile", memPath)

	if !run.failed() {
		t.Fatal("a run past its time budget should fail")
	}

	for _, teaches := range []string{"budget", "--timeout"} {
		if !strings.Contains(run.stderr, teaches) {
			t.Errorf("error should mention %q:\n%s", teaches, run.stderr)
		}
	}

	assertGzipFile(t, cpuPath, "cpu profile")
	assertGzipFile(t, memPath, "mem profile")
}

// TestCLI_StrayPositionalArgumentIsRejected: `run sqlite` used to benchmark
// the default backend in silence.
func TestCLI_StrayPositionalArgumentIsRejected(t *testing.T) {
	t.Parallel()

	scratch := t.TempDir()

	run := runBench(scratch, "run", "sqlite", "--profile", "dev")

	if !run.failed() {
		t.Fatal("a stray positional argument should fail")
	}

	if run.stdout != "" {
		t.Errorf("no benchmark should have run:\n%s", run.stdout)
	}

	assertNoLeakedStores(t, scratch)
}

// TestCLI_EnumFlagsAreRejectedAtParseTime: a bad --profile, --codec, or
// --format value fails before anything is created.
func TestCLI_EnumFlagsAreRejectedAtParseTime(t *testing.T) {
	t.Parallel()

	cases := [][]string{
		{"run", "--backend", "memory", "--profile", "bogus"},
		{"run", "--backend", "memory", "--codec", "xml"},
		{"run", "--backend", "memory", "--format", "xmml"},
		{"run", "--backend", "memory", "--durability", "ultra"},
	}

	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Parallel()

			scratch := t.TempDir()

			run := runBench(scratch, args...)

			if !run.failed() {
				t.Fatalf("cqrs-bench %s should fail", strings.Join(args, " "))
			}

			if !strings.Contains(run.stderr, args[len(args)-1]) {
				t.Errorf("error should name the rejected value:\n%s", run.stderr)
			}

			assertNoLeakedStores(t, scratch)
		})
	}
}

// TestCLI_StrictRejectsContradictions: --strict demands every phase, so a
// skip flag or --replay contradicts it.
func TestCLI_StrictRejectsContradictions(t *testing.T) {
	t.Parallel()

	cases := [][]string{
		{"run", "--backend", "memory", "--strict", "--skip-snapshot"},
		{"run", "--backend", "memory", "--strict", "--replay"},
		{"run", "--backend", "memory", "--strict", "--soak", "1s"},
	}

	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Parallel()

			run := runBench(t.TempDir(), args...)

			if !run.failed() {
				t.Fatalf("cqrs-bench %s should fail", strings.Join(args, " "))
			}

			conflict := args[len(args)-1]
			if !strings.HasPrefix(conflict, "--") {
				conflict = args[len(args)-2]
			}

			if !mentions(run.stderr, "--strict", conflict) {
				t.Errorf("error should mention --strict and %q:\n%s", conflict, run.stderr)
			}
		})
	}
}

// TestCLI_DirIsCreatedAndKeptForReplay: --dir may name a missing directory;
// the store it leaves is what --replay reads back.
func TestCLI_DirIsCreatedAndKeptForReplay(t *testing.T) {
	t.Parallel()

	scratch := t.TempDir()
	dir := filepath.Join(t.TempDir(), "does", "not", "exist")

	mustRunBench(t, scratch, "run", "--backend", "sqlite", "--dir", dir, "--profile", "dev",
		"--format", "json", "--quiet")

	store := filepath.Join(dir, toolDataDir, "sqlite", storeDirName, storeFileName)
	if _, err := os.Stat(store); err != nil {
		t.Fatalf("the store must be kept under --dir: %v", err)
	}

	assertNoLeakedStores(t, scratch)

	replay := mustRunBench(t, scratch, "run", "--backend", "sqlite", "--dir", dir, "--profile", "dev",
		"--replay", "--format", "json", "--quiet")

	result := decodeResult(t, replay.stdout)
	if result.TotalEvents == 0 {
		t.Error("--replay read no events from the kept store")
	}

	inherent := inherentSkippedPhases("sqlite")

	for _, phase := range phaseDocs() {
		replaySkips := slices.Contains(phase.flags, "--replay")
		listed := slices.Contains(result.SkippedPhases, phase.label) &&
			!slices.Contains(inherent, phase.label)

		if replaySkips != listed {
			t.Errorf("phase %q: --replay skips it = %v, the replay report lists it as skipped = %v (skipped: %v)",
				phase.name, replaySkips, listed, result.SkippedPhases)
		}
	}
}

// TestCLI_RepeatsReportTheirOwnFootprint pins the fix for repeat
// contamination: each repeat's disk size describes its own store. Before the
// fix every repeat reopened one store and the size grew by a full run per
// repeat.
func TestCLI_RepeatsReportTheirOwnFootprint(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "bench")

	run := mustRunBench(t, t.TempDir(), "run", "--backend", "sqlite", "--dir", dir, "--profile", "dev",
		"--repeat", "3", "--format", "manifest", "--include-runs", "--quiet")

	var manifest struct {
		Runs []benchResult `json:"runs"`
	}

	if err := json.Unmarshal([]byte(run.stdout), &manifest); err != nil {
		t.Fatalf("invalid manifest: %v\n%s", err, run.stdout)
	}

	if len(manifest.Runs) != 3 {
		t.Fatalf("manifest holds %d runs, want 3", len(manifest.Runs))
	}

	first := manifest.Runs[0].Disk.DatabaseBytes
	if first <= 0 {
		t.Fatalf("the first repeat reports no disk footprint: %d", first)
	}

	for i, repeat := range manifest.Runs {
		if size := repeat.Disk.DatabaseBytes; size > first*120/100 || size < first*80/100 {
			t.Errorf("repeat %d reports %d disk bytes, want the %d of one run (repeats are not isolated)",
				i+1, size, first)
		}
	}
}

// TestCLI_ReplayNeedsAnExistingStore: without --dir there is nothing to
// replay, and the error says so before anything runs.
func TestCLI_ReplayNeedsAnExistingStore(t *testing.T) {
	t.Parallel()

	scratch := t.TempDir()

	run := runBench(scratch, "run", "--backend", "sqlite", "--profile", "dev", "--replay")

	if !run.failed() {
		t.Fatal("--replay without --dir should fail")
	}

	if !strings.Contains(run.stderr, "--dir") {
		t.Errorf("error should name the flag:\n%s", run.stderr)
	}

	assertNoLeakedStores(t, scratch)
}
