package main

import (
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

	if !strings.Contains(string(out), "cqrs-bench") {
		t.Errorf("version output missing 'cqrs-bench': %s", out)
	}

	if !strings.Contains(string(out), "v4") {
		t.Errorf("version output missing version: %s", out)
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
	if !strings.Contains(output, "Usage:") {
		t.Error("help output missing 'Usage:'")
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
	if !strings.Contains(output, "| Backend |") {
		t.Error("compare markdown output missing table header")
	}

	if !strings.Contains(output, "| memory |") {
		t.Error("compare markdown output missing memory row")
	}
}
