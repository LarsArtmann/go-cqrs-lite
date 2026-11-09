package main

import (
	"os/exec"
	"strings"
	"testing"
)

// TestValidateFormatAcceptsEveryDocumentedFormat walks each handler's
// vocabulary through validateFormat. Rejected values exit the process
// (fatalf), so the reject paths are covered by the TestCLI_* e2e tests
// below; here we pin that no documented value is ever rejected.
func TestValidateFormatAcceptsEveryDocumentedFormat(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		command string
		allowed []string
	}{
		{"run", runFormats},
		{"compare", compareFormats},
		{"sweep", compareFormats},
		{"run --soak", soakFormats},
		{"layout", layoutFormats},
	} {
		for _, format := range tc.allowed {
			validateFormat(tc.command, format, tc.allowed)
		}
	}
}

// TestCLI_Run_InvalidFormat pins the fail-fast contract: an unknown format
// must exit non-zero before any benchmark work, not silently render text.
func TestCLI_Run_InvalidFormat(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)

	out, err := exec.Command(bin, "run", "--backend", "memory", "--format", "xmml").CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit for invalid --format")
	}

	output := string(out)
	if !strings.Contains(output, `invalid --format "xmml"`) {
		t.Errorf("output should quote the rejected format: %s", output)
	}

	if !strings.Contains(output, "supported:") {
		t.Errorf("error should list supported formats: %s", output)
	}
}

// TestCLI_Run_FormatMarkdown pins that run's markdown rendering is real
// (previously --format markdown silently fell back to plain text).
func TestCLI_Run_FormatMarkdown(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)

	out, err := exec.Command(
		bin, "run", "--backend", "memory", "--profile", "dev",
		"--format", "markdown", "--quiet",
	).CombinedOutput()
	if err != nil {
		t.Fatalf("markdown run failed: %v\n%s", err, out)
	}

	if !strings.Contains(string(out), "|") {
		t.Errorf("markdown output should contain table pipes: %s", out)
	}
}

// TestCLI_Compare_ManifestRejected: manifest is a run-only format; compare
// must reject it up front instead of silently rendering text.
func TestCLI_Compare_ManifestRejected(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)

	out, err := exec.Command(bin, "compare", "--format", "manifest").CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit for run-only format on compare")
	}

	if !strings.Contains(string(out), `invalid --format "manifest"`) {
		t.Errorf("output should reject manifest for compare: %s", out)
	}
}

// TestCLI_Sweep_BenchstatRejected: benchstat is a run-only format.
func TestCLI_Sweep_BenchstatRejected(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)

	out, err := exec.Command(bin, "sweep", "--format", "benchstat").CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit for run-only format on sweep")
	}

	if !strings.Contains(string(out), `invalid --format "benchstat"`) {
		t.Errorf("output should reject benchstat for sweep: %s", out)
	}
}

// TestCLI_Soak_FormatSubset pins that soak mode validates against its own
// subset: benchstat is valid for run, but a soak run renders soak tables —
// without the subset check this invocation would run and print text.
func TestCLI_Soak_FormatSubset(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)

	out, err := exec.Command(
		bin, "run", "--backend", "memory", "--soak", "10ms", "--format", "benchstat",
	).CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit for benchstat under soak")
	}

	if !strings.Contains(string(out), "invalid --format") {
		t.Errorf("output should reject benchstat under soak: %s", out)
	}
}

// TestCLI_Layout_InvalidFormat: layout advertises auto, table, text, json —
// anything else must fail loud.
func TestCLI_Layout_InvalidFormat(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)

	out, err := exec.Command(bin, "layout", "--format", "csv").CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit for unsupported layout format")
	}

	if !strings.Contains(string(out), `invalid --format "csv"`) {
		t.Errorf("output should reject csv for layout: %s", out)
	}
}

// TestCLI_Layout_Table pins that layout's advertised table format actually
// renders a table (previously --format table silently rendered text).
func TestCLI_Layout_Table(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)

	out, err := exec.Command(bin, "layout", "--format", "table").CombinedOutput()
	if err != nil {
		t.Fatalf("layout table failed: %v\n%s", err, out)
	}

	output := string(out)
	if !strings.Contains(output, "Priority") || !strings.Contains(output, "Embed") {
		t.Errorf("table output missing Priority/Embed columns: %s", output)
	}
}
