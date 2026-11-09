package main

import (
	"encoding/json/v2"
	"os/exec"
	"strings"
	"testing"
)

func TestLayoutCommand_AllLayouts(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)
	out, err := exec.Command(bin, "layout").CombinedOutput()
	if err != nil {
		t.Fatalf("layout command failed: %v\n%s", err, out)
	}

	output := string(out)
	for _, layout := range []string{"KV", "LSM", "Row", "Columnar"} {
		if !strings.Contains(output, layout) {
			t.Errorf("output missing layout %q:\n%s", layout, output)
		}
	}
	for _, pri := range []string{"Balanced", "ReadSpeed", "WriteSpeed", "StorageSpace"} {
		if !strings.Contains(output, pri) {
			t.Errorf("output missing priority %q:\n%s", pri, output)
		}
	}
}

func TestLayoutCommand_PriorityFilter(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)
	out, err := exec.Command(bin, "layout", "--priority", "write-speed").CombinedOutput()
	if err != nil {
		t.Fatalf("layout command failed: %v\n%s", err, out)
	}

	output := string(out)
	if strings.Contains(output, "Balanced") {
		t.Error("output should not contain 'Balanced' when filtered to write-speed")
	}
	if !strings.Contains(output, "WriteSpeed") {
		t.Error("output must contain 'WriteSpeed'")
	}
}

func TestLayoutCommand_LayoutFilter(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)
	out, err := exec.Command(bin, "layout", "--layout", "kv").CombinedOutput()
	if err != nil {
		t.Fatalf("layout command failed: %v\n%s", err, out)
	}

	output := string(out)
	if !strings.Contains(output, "KV") {
		t.Error("output must contain 'KV'")
	}
	if strings.Contains(output, "LSM") {
		t.Error("output should not contain 'LSM' when filtered to kv")
	}
}

func TestLayoutCommand_JSON(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)
	out, err := exec.Command(bin, "layout", "--format", "json").CombinedOutput()
	if err != nil {
		t.Fatalf("layout command failed: %v\n%s", err, out)
	}

	var groups []layoutGroup
	if err := json.Unmarshal(out, &groups); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, out)
	}
	if len(groups) != 4 {
		t.Errorf("expected 4 layout groups, got %d", len(groups))
	}
	for _, g := range groups {
		if len(g.Entries) == 0 {
			t.Errorf("group %q has no entries", g.Layout)
		}
	}
}

func TestLayoutCommand_Verbose(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)
	out, err := exec.Command(bin, "layout", "--verbose", "--layout", "kv", "--priority", "balanced").
		CombinedOutput()
	if err != nil {
		t.Fatalf("layout command failed: %v\n%s", err, out)
	}

	output := string(out)
	for _, token := range []string{"Read=", "Write=", "Storage="} {
		if !strings.Contains(output, token) {
			t.Errorf("verbose output should contain %q", token)
		}
	}
}
