package main

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func layoutLayouts() []string {
	return []string{"KV", "LSM", "Row", "Columnar"}
}

func layoutPriorities() []string {
	return []string{"Balanced", "ReadSpeed", "WriteSpeed", "StorageSpace"}
}

func TestLayoutCommand_AllLayouts(t *testing.T) {
	t.Parallel()

	run := mustRunBench(t, t.TempDir(), "layout")

	for _, layout := range layoutLayouts() {
		if !strings.Contains(run.stdout, layout) {
			t.Errorf("output missing layout %q:\n%s", layout, run.stdout)
		}
	}

	for _, priority := range layoutPriorities() {
		if !strings.Contains(run.stdout, priority) {
			t.Errorf("output missing priority %q:\n%s", priority, run.stdout)
		}
	}
}

func TestLayoutCommand_PriorityFilter(t *testing.T) {
	t.Parallel()

	run := mustRunBench(t, t.TempDir(), "layout", "--priority", "write-speed")

	if strings.Contains(run.stdout, "Balanced") {
		t.Error("output should not contain 'Balanced' when filtered to write-speed")
	}

	if !strings.Contains(run.stdout, "WriteSpeed") {
		t.Error("output must contain 'WriteSpeed'")
	}
}

func TestLayoutCommand_LayoutFilter(t *testing.T) {
	t.Parallel()

	run := mustRunBench(t, t.TempDir(), "layout", "--layout", "kv")

	if !strings.Contains(run.stdout, "KV") {
		t.Error("output must contain 'KV'")
	}

	if strings.Contains(run.stdout, "LSM") {
		t.Error("output should not contain 'LSM' when filtered to kv")
	}
}

func TestLayoutCommand_JSON(t *testing.T) {
	t.Parallel()

	run := mustRunBench(t, t.TempDir(), "layout", "--format", "json")

	var groups []layoutGroup
	if err := json.Unmarshal([]byte(run.stdout), &groups); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, run.stdout)
	}

	if len(groups) != len(layoutLayouts()) {
		t.Errorf("expected %d layout groups, got %d", len(layoutLayouts()), len(groups))
	}

	for _, group := range groups {
		if len(group.Entries) != len(layoutPriorities()) {
			t.Errorf("group %q has %d entries, want one per priority", group.Layout, len(group.Entries))
		}

		for _, entry := range group.Entries {
			if !slices.Contains([]string{"Embed", "Normalize"}, entry.Selected) {
				t.Errorf("group %q selects %q, which the planner never returns", group.Layout, entry.Selected)
			}
		}
	}
}

func TestLayoutCommand_Verbose(t *testing.T) {
	t.Parallel()

	run := mustRunBench(t, t.TempDir(), "layout", "--verbose", "--layout", "kv", "--priority", "balanced")

	for _, token := range []string{"Read=", "Write=", "Storage="} {
		if !strings.Contains(run.stdout, token) {
			t.Errorf("verbose output should contain %q", token)
		}
	}
}

// TestLayoutCommand_RejectsUnknownFilters: a bad --priority or --layout is a
// usage error that names the flag, not a crash or an empty report.
func TestLayoutCommand_RejectsUnknownFilters(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{
		{"layout", "--priority", "bogus"},
		{"layout", "--layout", "bogus"},
	} {
		run := runBench(t.TempDir(), args...)

		if !run.failed() {
			t.Errorf("cqrs-bench %s should fail", strings.Join(args, " "))

			continue
		}

		if want := args[1]; !strings.Contains(run.stderr, want) {
			t.Errorf("error should name %s:\n%s", want, run.stderr)
		}
	}
}
