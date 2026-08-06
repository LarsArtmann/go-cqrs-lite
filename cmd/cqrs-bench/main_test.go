package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// buildBinary builds the cqrs-bench binary and returns its path.
func buildBinary(t *testing.T) string {
	t.Helper()

	bin := t.TempDir() + "/cqrs-bench"

	cmd := exec.Command("go", "build", "-tags", "goexperiment.jsonv2", "-o", bin, ".")
	cmd.Env = append(cmd.Environ(), "GOEXPERIMENT=jsonv2")

	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}

	return bin
}

func TestCLI_Version(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)

	out, err := exec.Command(bin, "--version").Output()
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	output := string(out)
	if !strings.Contains(output, "cqrs-bench") {
		t.Errorf("version output missing 'cqrs-bench': %s", output)
	}

	// Accept both tagged builds (v0.1.0) and development builds ((devel, abc1234)).
	if !strings.Contains(output, "version") {
		t.Errorf("version output missing 'version': %s", output)
	}

	if !strings.Contains(output, "devel") && !strings.Contains(output, "v") {
		t.Errorf("version output missing version or devel marker: %s", output)
	}
}

func TestCLI_Help(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)

	out, err := exec.Command(bin, "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("help command failed: %v", err)
	}

	output := string(out)
	if !strings.Contains(strings.ToLower(output), "usage") {
		t.Error("help output missing 'usage' section")
	}

	if !strings.Contains(output, "run") {
		t.Error("help output missing 'run' subcommand")
	}

	if !strings.Contains(output, "compare") {
		t.Error("help output missing 'compare' subcommand")
	}
}

func TestCLI_Run_Memory(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)

	out, err := exec.Command(
		bin, "run",
		"--backend", "memory",
		"--profile", "dev",
		"--payload-size", "64",
	).CombinedOutput()
	if err != nil {
		t.Fatalf("run --backend memory failed: %v\n%s", err, out)
	}

	output := string(out)
	if !strings.Contains(output, "Benchmark:") {
		t.Error("run output missing 'Benchmark:' header")
	}

	if !strings.Contains(output, "Write Performance") {
		t.Error("run output missing 'Write Performance' section")
	}
}

func TestCLI_Run_JSON(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)

	out, err := exec.Command(
		bin, "run",
		"--backend", "memory",
		"--profile", "dev",
		"--format", "json",
		"--payload-size", "64",
	).CombinedOutput()
	if err != nil {
		t.Fatalf("run --format json failed: %v\n%s", err, out)
	}

	if !strings.Contains(string(out), `"backend"`) {
		t.Error("JSON output missing 'backend' field")
	}
}

func TestCLI_Compare(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)

	out, err := exec.Command(
		bin, "compare",
		"--profile", "dev",
		"--backends", "memory",
		"--format", "markdown",
		"--payload-size", "64",
	).CombinedOutput()
	if err != nil {
		t.Fatalf("compare failed: %v\n%s", err, out)
	}

	output := string(out)
	if !strings.Contains(output, "Backend") {
		t.Error("compare markdown output missing table header")
	}

	if !strings.Contains(output, "| memory") {
		t.Error("compare markdown output missing memory row")
	}
}

func TestCLI_UnknownProfile(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)

	out, err := exec.Command(
		bin, "run", "--backend", "memory", "--profile", "bogus",
	).CombinedOutput()

	if err == nil {
		t.Fatal("expected non-zero exit for unknown profile")
	}

	if !strings.Contains(string(out), "unknown profile") {
		t.Errorf("output should mention 'unknown profile': %s", out)
	}
}

func TestCLI_UnknownBackend(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)

	out, err := exec.Command(
		bin, "run", "--backend", "bogus", "--profile", "dev",
	).CombinedOutput()

	if err == nil {
		t.Fatal("expected non-zero exit for unknown backend")
	}

	if !strings.Contains(string(out), "unknown backend") {
		t.Errorf("output should mention 'unknown backend': %s", out)
	}
}

func TestCLI_CodecCBOR(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)

	out, err := exec.Command(
		bin, "run",
		"--backend", "memory",
		"--profile", "dev",
		"--codec", "cbor",
		"--format", "json",
		"--payload-size", "64",
	).CombinedOutput()
	if err != nil {
		t.Fatalf("run --codec cbor failed: %v\n%s", err, out)
	}

	if !strings.Contains(string(out), `"codec": "cbor"`) {
		t.Errorf("JSON output should show cbor codec: %s", out)
	}
}

func TestCLI_WarmupFlag(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)

	out, err := exec.Command(
		bin, "run",
		"--backend", "memory",
		"--profile", "dev",
		"--warmup", "50",
		"--format", "json",
		"--payload-size", "64",
	).CombinedOutput()
	if err != nil {
		t.Fatalf("run --warmup 50 failed: %v\n%s", err, out)
	}

	if !strings.Contains(string(out), `"warmupEvents": 50`) {
		t.Errorf("JSON output should show warmupEvents=50: %s", out)
	}
}

func TestCLI_OutputFile(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)

	tmpFile := t.TempDir() + "/result.json"

	out, err := exec.Command(
		bin, "run",
		"--backend", "memory",
		"--profile", "dev",
		"--format", "json",
		"--output", tmpFile,
		"--payload-size", "64",
	).CombinedOutput()
	if err != nil {
		t.Fatalf("run --output failed: %v\n%s", err, out)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("output file not readable: %v", err)
	}

	if !strings.Contains(string(data), `"backend"`) {
		t.Error("output file missing 'backend' field")
	}
}

func TestCLI_Compare_DiskNonZero(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)

	// compare should set DiskPath per-backend so sqlite shows non-zero disk.
	out, err := exec.Command(
		bin, "compare",
		"--profile", "dev",
		"--backends", "memory,sqlite",
		"--format", "markdown",
		"--payload-size", "64",
	).CombinedOutput()
	if err != nil {
		t.Fatalf("compare failed: %v\n%s", err, out)
	}

	output := string(out)

	if !strings.Contains(output, "| sqlite") {
		t.Fatalf("compare output missing sqlite row:\n%s", output)
	}

	// sqlite row should show a non-zero disk value (M, KB, B — not "0 B").
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "| sqlite") {
			if strings.Contains(line, "| 0 B |") {
				t.Errorf("sqlite disk should be non-zero in compare:\n%s", line)
			}

			return
		}
	}
}

func TestCLI_Repeat(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)

	out, err := exec.Command(
		bin, "run",
		"--backend", "memory",
		"--profile", "dev",
		"--repeat", "3",
		"--payload-size", "64",
	).CombinedOutput()
	if err != nil {
		t.Fatalf("run --repeat failed: %v\n%s", err, out)
	}

	output := string(out)

	if !strings.Contains(output, "median of 3 runs") {
		t.Errorf("expected 'median of 3 runs' in output:\n%s", output)
	}
}

func TestCLI_PayloadSizesFlagConsumed(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)

	// pflag blindly consumes the next token as a string flag's value.
	// "--payload-sizes --profile stress" should detect the flag-like value
	// and return a clear, actionable error instead of a confusing strconv error.
	out, err := exec.Command(
		bin, "compare", "--payload-sizes", "--profile", "stress",
	).CombinedOutput()

	if err == nil {
		t.Fatal("expected non-zero exit when --payload-sizes consumes a flag name")
	}

	output := string(out)
	if !strings.Contains(output, "looks like a flag name") {
		t.Errorf("error should explain the flag-like value issue:\n%s", output)
	}

	if strings.Contains(output, "strconv.Atoi") {
		t.Errorf("error should not leak raw strconv error:\n%s", output)
	}
}

func TestCLI_Quiet(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)

	cmd := exec.Command(
		bin, "run",
		"--backend", "memory",
		"--profile", "dev",
		"--quiet",
		"--format", "json",
		"--payload-size", "64",
	)

	var stderr strings.Builder
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("run --quiet failed: %v\nstderr: %s", err, stderr.String())
	}

	if stderr.Len() > 0 {
		t.Errorf("--quiet should suppress all stderr output, got %d bytes:\n%s",
			stderr.Len(), stderr.String())
	}

	if !strings.Contains(string(out), `"backend"`) {
		t.Error("--quiet should still produce JSON result on stdout")
	}
}
