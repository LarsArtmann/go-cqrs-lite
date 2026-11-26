package main

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

// sweepPoint is one element of a sweep's JSON report.
type sweepPoint struct {
	Parameter string      `json:"parameter"`
	Value     int         `json:"value"`
	Result    benchResult `json:"result"`
}

// mustSweep runs a sweep and returns its decoded JSON points.
func mustSweep(t *testing.T, extra ...string) []sweepPoint {
	t.Helper()

	args := append([]string{"sweep", "--param", "workers", "--backend", "memory", "--profile", "dev",
		"--format", "json", "--quiet"}, extra...)

	run := mustRunBench(t, t.TempDir(), args...)

	var points []sweepPoint
	if err := json.Unmarshal([]byte(run.stdout), &points); err != nil {
		t.Fatalf("invalid JSON sweep: %v\n%s", err, run.stdout)
	}

	return points
}

func TestCLI_Sweep(t *testing.T) {
	t.Parallel()

	points := mustSweep(t, "--values", "1,2,4")

	values := make([]int, 0, len(points))
	for _, point := range points {
		values = append(values, point.Value)
	}

	if !slices.Equal(values, []int{1, 2, 4}) {
		t.Errorf("sweep values = %v, want [1 2 4]", values)
	}

	if len(points) > 0 && points[0].Parameter != "workers" {
		t.Errorf("parameter = %q, want workers", points[0].Parameter)
	}
}

func TestCLI_Sweep_Formats(t *testing.T) {
	t.Parallel()

	cases := []struct {
		format  string
		teaches string
	}{
		{format: "markdown", teaches: "| 1"},
		{format: "table", teaches: "Workers"},
		{format: "csv", teaches: "Workers,Write P50"},
	}

	for _, tc := range cases {
		t.Run(tc.format, func(t *testing.T) {
			t.Parallel()

			args := []string{"sweep", "--param", "workers", "--values", "1,2", "--backend", "memory",
				"--profile", "dev", "--format", tc.format, "--quiet"}

			run := mustRunBench(t, t.TempDir(), args...)

			if !strings.Contains(run.stdout, tc.teaches) {
				t.Errorf("--format %s output should contain %q:\n%s", tc.format, tc.teaches, run.stdout)
			}
		})
	}
}

// TestCLI_Sweep_HonorsSkipFlags pins the fix for flags sweep used to drop.
func TestCLI_Sweep_HonorsSkipFlags(t *testing.T) {
	t.Parallel()

	points := mustSweep(t, "--values", "1,2", "--skip-mixed", "--skip-snapshot")

	if len(points) != 2 {
		t.Fatalf("got %d points, want 2", len(points))
	}

	for _, point := range points {
		for _, label := range []string{"mixed workload phase", "snapshot phase"} {
			if !slices.Contains(point.Result.SkippedPhases, label) {
				t.Errorf("value %d: --skip flag not honored, skipped = %v", point.Value, point.Result.SkippedPhases)
			}
		}
	}
}

// TestCLI_Sweep_ValuesNeedTwoPoints: the error names --values and does not
// borrow the wording of --payload-sizes.
func TestCLI_Sweep_ValuesNeedTwoPoints(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		values  []string
		teaches string
	}{
		{name: "one value", values: []string{"--values", "5"}, teaches: "--values needs at least two"},
		{name: "no values", values: nil, teaches: "--values"},
		{name: "zero", values: []string{"--values", "0,4"}, teaches: "greater than 0"},
		{name: "swallowed flag", values: []string{"--values", "--backend"}, teaches: "looks like a flag name"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			args := append([]string{"sweep", "--param", "workers", "--backend", "memory", "--profile", "dev"},
				tc.values...)

			run := runBench(t.TempDir(), args...)

			if !run.failed() {
				t.Fatalf("sweep %v should fail", tc.values)
			}

			if !strings.Contains(run.stderr, tc.teaches) {
				t.Errorf("error should contain %q:\n%s", tc.teaches, run.stderr)
			}

			if strings.Contains(run.stderr, "mixed workload") || strings.Contains(run.stderr, "payload-sizes") {
				t.Errorf("a --values error must not talk about --payload-sizes:\n%s", run.stderr)
			}
		})
	}
}

// TestCLI_Sweep_UnknownParamLeavesNoStore: an invalid --param used to be
// noticed after the store directory was created and exited the process
// without removing it.
func TestCLI_Sweep_UnknownParamLeavesNoStore(t *testing.T) {
	t.Parallel()

	scratch := t.TempDir()

	run := runBench(scratch, "sweep", "--param", "bogus", "--values", "1,2", "--backend", "sqlite", "--profile", "dev")

	if !run.failed() {
		t.Fatal("sweep with an unknown --param should fail")
	}

	if !strings.Contains(run.stderr, "--param") || !strings.Contains(run.stderr, "bogus") {
		t.Errorf("error should name the flag and the value:\n%s", run.stderr)
	}

	assertNoLeakedStores(t, scratch)
}

// TestCLI_Sweep_EveryPointFailedExitsNonZero.
func TestCLI_Sweep_EveryPointFailedExitsNonZero(t *testing.T) {
	t.Parallel()

	run := runBench(t.TempDir(), "sweep", "--param", "workers", "--values", "1,2", "--backend", "postgres")

	if !run.failed() {
		t.Fatal("a sweep in which every data point failed must exit non-zero")
	}
}
