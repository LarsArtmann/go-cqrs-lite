package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// benchBin is the cqrs-bench binary, built once per test run by TestMain.
var benchBin string

// TestMain builds the binary once. Every end-to-end test used to run its own
// `go build`, which dominated the package's test time.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "cqrs-bench-bin-*")
	if err != nil {
		panic("create temp dir for the cqrs-bench binary: " + err.Error())
	}

	benchBin = filepath.Join(dir, "cqrs-bench")

	if out, err := exec.Command("go", "build", "-o", benchBin, ".").CombinedOutput(); err != nil {
		panic("go build failed: " + err.Error() + "\n" + string(out))
	}

	code := m.Run()

	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// benchRun is the outcome of one cqrs-bench invocation.
type benchRun struct {
	stdout string
	stderr string
	err    error
}

// failed reports whether the command exited non-zero.
func (run benchRun) failed() bool {
	return run.err != nil
}

// benchEnv is the environment for a test invocation: a private TMPDIR, so a
// leaked temporary store is visible in that directory, and no forced color,
// so output does not depend on the ambient terminal settings.
func benchEnv(scratch string) []string {
	env := make([]string, 0, len(os.Environ())+1)

	for _, pair := range os.Environ() {
		name, _, _ := strings.Cut(pair, "=")

		switch name {
		case "TMPDIR", "FORCE_COLOR", "GO_OUTPUT_FORCE_COLOR":
			continue
		}

		env = append(env, pair)
	}

	return append(env, "TMPDIR="+scratch)
}

// runBench runs the built cqrs-bench with args and returns its output.
func runBench(scratch string, args ...string) benchRun {
	cmd := exec.Command(benchBin, args...)
	cmd.Env = benchEnv(scratch)

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	return benchRun{stdout: stdout.String(), stderr: stderr.String(), err: cmd.Run()}
}

// mustRunBench runs cqrs-bench and fails the test when it exits non-zero.
func mustRunBench(t *testing.T, scratch string, args ...string) benchRun {
	t.Helper()

	run := runBench(scratch, args...)
	if run.failed() {
		t.Fatalf("cqrs-bench %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), run.err,
			run.stdout, run.stderr)
	}

	return run
}

// assertNoLeakedStores fails when the command left a temporary store behind.
func assertNoLeakedStores(t *testing.T, scratch string) {
	t.Helper()

	left, _ := filepath.Glob(filepath.Join(scratch, "cqrs-bench-*"))
	if len(left) > 0 {
		t.Errorf("temporary stores left behind: %v", left)
	}
}

// benchResult is the part of a benchkit result the tests assert on.
type benchResult struct {
	Backend       string   `json:"backend"`
	Profile       string   `json:"profile"`
	Codec         string   `json:"codec"`
	TotalEvents   int      `json:"totalEvents"`
	WarmupEvents  int      `json:"warmupEvents"`
	SkippedPhases []string `json:"skippedPhases"`
	Disk          struct {
		DatabaseBytes int64 `json:"databaseBytes"`
	} `json:"disk"`
}

// decodeResult unmarshals a single-run result.
func decodeResult(t *testing.T, document string) benchResult {
	t.Helper()

	var result benchResult
	if err := json.Unmarshal([]byte(document), &result); err != nil {
		t.Fatalf("invalid JSON result: %v\n%s", err, document)
	}

	return result
}

func TestCLI_Version(t *testing.T) {
	t.Parallel()

	run := mustRunBench(t, t.TempDir(), "--version")

	for _, want := range []string{"cqrs-bench", "version"} {
		if !strings.Contains(run.stdout, want) {
			t.Errorf("version output missing %q: %s", want, run.stdout)
		}
	}

	if !strings.Contains(run.stdout, "devel") && !strings.Contains(run.stdout, "v") {
		t.Errorf("version output missing version or devel marker: %s", run.stdout)
	}
}

// TestCLI_RootHelpListsEveryBackend: the long description is generated from
// the backend registry, so a backend can no longer be missing from help.
func TestCLI_RootHelpListsEveryBackend(t *testing.T) {
	t.Parallel()

	run := mustRunBench(t, t.TempDir(), "--help")

	for _, want := range append(backendNames(), "run", "compare", "sweep", "list-phases", "layout") {
		if !strings.Contains(run.stdout, want) {
			t.Errorf("root help missing %q", want)
		}
	}
}

func TestCLI_Run_Memory(t *testing.T) {
	t.Parallel()

	run := mustRunBench(t, t.TempDir(), "run", "--backend", "memory", "--profile", "dev", "--payload-size", "64")

	for _, want := range []string{"Benchmark:", "Write Performance"} {
		if !strings.Contains(run.stdout, want) {
			t.Errorf("run output missing %q", want)
		}
	}
}

func TestCLI_Run_JSON(t *testing.T) {
	t.Parallel()

	run := mustRunBench(t, t.TempDir(), "run", "--backend", "memory", "--profile", "dev",
		"--format", "json", "--payload-size", "64")

	if result := decodeResult(t, run.stdout); result.Backend != "memory" || result.TotalEvents != 500 {
		t.Errorf("result = backend %q with %d events, want memory with 500", result.Backend, result.TotalEvents)
	}
}

// TestCLI_Run_AliasIsReportedCanonically: results carry the canonical
// backend name, however it was spelled.
func TestCLI_Run_AliasIsReportedCanonically(t *testing.T) {
	t.Parallel()

	run := mustRunBench(t, t.TempDir(), "run", "--backend", "mem", "--profile", "dev", "--format", "json", "--quiet")

	if result := decodeResult(t, run.stdout); result.Backend != "memory" {
		t.Errorf("backend = %q, want the canonical \"memory\"", result.Backend)
	}
}

func TestCLI_CodecCBOR(t *testing.T) {
	t.Parallel()

	run := mustRunBench(t, t.TempDir(), "run", "--backend", "memory", "--profile", "dev",
		"--codec", "cbor", "--format", "json", "--payload-size", "64")

	if result := decodeResult(t, run.stdout); result.Codec != "cbor" {
		t.Errorf("codec = %q, want cbor", result.Codec)
	}
}

// TestCLI_WarmupFlag uses the in-memory store: benchkit v4.7.0, the pin this
// module builds against, rejects a second warmup event on a store that
// checks versions (fixed in benchkit on 2026-11-25).
func TestCLI_WarmupFlag(t *testing.T) {
	t.Parallel()

	run := mustRunBench(t, t.TempDir(), "run", "--backend", "memory", "--profile", "dev",
		"--warmup", "50", "--format", "json", "--payload-size", "64")

	if result := decodeResult(t, run.stdout); result.WarmupEvents != 50 {
		t.Errorf("warmupEvents = %d, want 50", result.WarmupEvents)
	}
}

// TestCLI_OutputFileIsCleanJSON: a report file holds exactly the report and
// no staging leftovers.
func TestCLI_OutputFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "result.json")

	mustRunBench(t, t.TempDir(), "run", "--backend", "memory", "--profile", "dev",
		"--format", "json", "--output", path, "--payload-size", "64")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("output file not readable: %v", err)
	}

	if result := decodeResult(t, string(content)); result.Backend != "memory" {
		t.Errorf("output file backend = %q, want memory", result.Backend)
	}

	if leftovers, _ := filepath.Glob(filepath.Join(dir, "*")); len(leftovers) != 1 {
		t.Errorf("the report directory must hold only the report, found %v", leftovers)
	}
}

func TestCLI_Repeat(t *testing.T) {
	t.Parallel()

	run := mustRunBench(t, t.TempDir(), "run", "--backend", "memory", "--profile", "dev",
		"--repeat", "3", "--payload-size", "64")

	for _, want := range []string{"median of 3 runs", "Variation:"} {
		if !strings.Contains(run.stdout, want) {
			t.Errorf("expected %q in repeated run output:\n%s", want, run.stdout)
		}
	}
}

func TestCLI_RepeatBenchstatMultiSample(t *testing.T) {
	t.Parallel()

	run := mustRunBench(t, t.TempDir(), "run", "--backend", "memory", "--profile", "dev",
		"--repeat", "3", "--payload-size", "64", "--format", "benchstat", "--quiet")

	counts := map[string]int{}

	for line := range strings.SplitSeq(run.stdout, "\n") {
		if strings.HasPrefix(line, "Benchmark") {
			counts[strings.Fields(line)[0]]++
		}
	}

	if len(counts) == 0 {
		t.Fatalf("no benchstat lines in output:\n%s", run.stdout)
	}

	for name, count := range counts {
		if count != 3 {
			t.Errorf("metric %s emitted %d samples, want one per repeat run (3)", name, count)
		}
	}
}

func TestCLI_Quiet(t *testing.T) {
	t.Parallel()

	run := mustRunBench(t, t.TempDir(), "run", "--backend", "memory", "--profile", "dev",
		"--quiet", "--format", "json", "--payload-size", "64")

	if run.stderr != "" {
		t.Errorf("--quiet should suppress all stderr output, got:\n%s", run.stderr)
	}

	if result := decodeResult(t, run.stdout); result.Backend != "memory" {
		t.Errorf("--quiet should still produce the JSON result on stdout: %s", run.stdout)
	}
}

// TestCLI_IgnoredFlagIsReported: a flag the backend does not act on is
// warned about instead of being dropped in silence, even under --quiet.
func TestCLI_IgnoredFlagIsReported(t *testing.T) {
	t.Parallel()

	run := mustRunBench(t, t.TempDir(), "run", "--backend", "memory", "--profile", "dev",
		"--dsn", "nowhere", "--dir", "unused", "--quiet", "--format", "json")

	for _, want := range []string{"warning", "--dsn", "--dir"} {
		if !strings.Contains(run.stderr, want) {
			t.Errorf("stderr should mention %q for an ignored flag:\n%s", want, run.stderr)
		}
	}

	decodeResult(t, run.stdout)
}
