package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// markdownRow returns the cells of the markdown table row that starts with
// name, or nil when there is no such row.
func markdownRow(report, name string) []string {
	for line := range strings.SplitSeq(report, "\n") {
		if !strings.HasPrefix(line, "| "+name+" ") {
			continue
		}

		cells := strings.Split(strings.Trim(line, "|"), "|")
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}

		return cells
	}

	return nil
}

func TestCLI_Compare(t *testing.T) {
	t.Parallel()

	run := mustRunBench(t, t.TempDir(), "compare", "--profile", "dev", "--backends", "memory",
		"--format", "markdown", "--payload-size", "64", "--quiet")

	if !strings.Contains(run.stdout, "Backend") {
		t.Errorf("compare markdown output missing table header:\n%s", run.stdout)
	}

	if row := markdownRow(run.stdout, "memory"); row == nil || row[0] != "memory" {
		t.Errorf("compare markdown output missing memory row:\n%s", run.stdout)
	}
}

// TestCLI_Compare_DiskIsMeasuredPerBackend: each backend is pointed at its
// own store directory, so a disk-backed one reports a real footprint.
func TestCLI_Compare_DiskIsMeasuredPerBackend(t *testing.T) {
	t.Parallel()

	scratch := t.TempDir()

	run := mustRunBench(t, scratch, "compare", "--profile", "dev", "--backends", "memory,sqlite",
		"--format", "markdown", "--payload-size", "64", "--quiet")

	row := markdownRow(run.stdout, "sqlite")
	if row == nil {
		t.Fatalf("compare output missing sqlite row:\n%s", run.stdout)
	}

	if disk := row[len(row)-1]; disk == "0 B" || disk == "-" {
		t.Errorf("sqlite disk = %q, want a measured footprint: %v", disk, row)
	}

	assertNoLeakedStores(t, scratch)
}

// TestCLI_Compare_AliasesAreCanonical: rows are keyed by the canonical name.
func TestCLI_Compare_AliasesAreCanonical(t *testing.T) {
	t.Parallel()

	run := mustRunBench(t, t.TempDir(), "compare", "--profile", "dev", "--backends", "mem,sys-mem",
		"--format", "markdown", "--payload-size", "64", "--quiet")

	for _, name := range []string{"memory", "system-memory"} {
		if markdownRow(run.stdout, name) == nil {
			t.Errorf("compare output missing a %q row:\n%s", name, run.stdout)
		}
	}
}

// TestCLI_Compare_HonorsSkipFlags pins the fix for flags compare used to
// drop: a phase skipped on the command line is skipped in every compared
// backend.
func TestCLI_Compare_HonorsSkipFlags(t *testing.T) {
	t.Parallel()

	run := mustRunBench(t, t.TempDir(), "compare", "--profile", "dev", "--backends", "memory",
		"--skip-mixed", "--format", "json", "--payload-size", "64", "--quiet")

	var results map[string]benchResult
	if err := json.Unmarshal([]byte(run.stdout), &results); err != nil {
		t.Fatalf("invalid JSON comparison: %v\n%s", err, run.stdout)
	}

	if !slices.Contains(results["memory"].SkippedPhases, "mixed workload phase") {
		t.Errorf("--skip-mixed was not honored, skipped = %v", results["memory"].SkippedPhases)
	}
}

// TestCLI_Compare_PassesDirToBackends pins the fix for compare dropping
// --dir: the store lands under it and no temporary store is created.
func TestCLI_Compare_PassesDirToBackends(t *testing.T) {
	t.Parallel()

	scratch := t.TempDir()
	dir := filepath.Join(t.TempDir(), "bench")

	mustRunBench(t, scratch, "compare", "--profile", "dev", "--backends", "sqlite", "--dir", dir,
		"--format", "json", "--payload-size", "64", "--quiet")

	if _, err := os.Stat(filepath.Join(dir, toolDataDir, "sqlite", storeDirName, storeFileName)); err != nil {
		t.Errorf("--dir was not passed to the backend: %v", err)
	}

	assertNoLeakedStores(t, scratch)
}

func TestCLI_Compare_RejectsUnknownAndDuplicateBackends(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		backends  string
		teaches   []string
		noContext []string
	}{
		{name: "unknown backend", backends: "memory,bogus", teaches: []string{"unknown backend", "bogus"}},
		{name: "backend named twice", backends: "memory,mem", teaches: []string{"twice"}},
		{name: "empty list", backends: " ", teaches: []string{"--backends"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			run := runBench(t.TempDir(), "compare", "--profile", "dev", "--backends", tc.backends)

			if !run.failed() {
				t.Fatalf("compare --backends %q should fail", tc.backends)
			}

			for _, teaches := range tc.teaches {
				if !strings.Contains(run.stderr, teaches) {
					t.Errorf("error should mention %q:\n%s", teaches, run.stderr)
				}
			}
		})
	}
}

// TestCLI_Compare_AllFailedExitsNonZero: a comparison in which not one
// backend ran is a failure, not a report of FAILED rows with exit code 0.
func TestCLI_Compare_AllFailedExitsNonZero(t *testing.T) {
	t.Parallel()

	run := runBench(t.TempDir(), "compare", "--profile", "dev", "--backends", "postgres")

	if !run.failed() {
		t.Fatal("compare whose only backend failed must exit non-zero")
	}

	if !strings.Contains(run.stderr, "--dsn") {
		t.Errorf("error should tell the operator what is missing:\n%s", run.stderr)
	}
}

func TestCLI_Compare_ReportsFormats(t *testing.T) {
	t.Parallel()

	cases := []struct {
		format  string
		teaches string
	}{
		{format: "markdown", teaches: "| memory"},
		{format: "csv", teaches: "Backend,Write P50"},
		{format: "json", teaches: `"memory"`},
		{format: "tsv", teaches: "Backend\tWrite P50"},
		{format: "table", teaches: "Backend"},
	}

	for _, tc := range cases {
		t.Run(tc.format, func(t *testing.T) {
			t.Parallel()

			run := mustRunBench(t, t.TempDir(), "compare", "--profile", "dev", "--backends", "memory",
				"--format", tc.format, "--payload-size", "64", "--quiet")

			if !strings.Contains(run.stdout, tc.teaches) {
				t.Errorf("--format %s output should contain %q:\n%s", tc.format, tc.teaches, run.stdout)
			}
		})
	}
}
